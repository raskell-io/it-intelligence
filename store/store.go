// Package store is the PostgreSQL persistence of the Context Model: observations, entities, edges,
// scorecards and artifacts. Plain SQL over pgx; no ORM, no migration framework.
package store

import (
	"context"
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/raskell-io/it-intelligence/connectors"
	"github.com/raskell-io/it-intelligence/resolver"
)

//go:embed vocabulary.sql
var vocabularySQL string

//go:embed migrations/*.sql
var migrations embed.FS

// Store wraps a connection pool.
type Store struct {
	Pool *pgxpool.Pool
}

// Open connects to PostgreSQL.
func Open(ctx context.Context, url string) (*Store, error) {
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		return nil, err
	}
	cfg.ConnConfig.RuntimeParams["timezone"] = "UTC"
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("connect to %s: %w", redact(url), err)
	}
	return &Store{Pool: pool}, nil
}

func redact(url string) string {
	if at := strings.LastIndex(url, "@"); at > 0 {
		if i := strings.Index(url, "://"); i > 0 {
			return url[:i+3] + "***" + url[at:]
		}
	}
	return url
}

// Close closes the pool.
func (s *Store) Close() { s.Pool.Close() }

// Migrate applies the vocabulary (idempotent, every time: the ontology may have changed) and every
// numbered migration not applied yet.
func (s *Store) Migrate(ctx context.Context) error {
	if _, err := s.Pool.Exec(ctx, `create table if not exists schema_migration (name text primary key, applied_at timestamptz not null default now())`); err != nil {
		return err
	}
	// The vocabulary must exist before the first migration references it, and is re-applied after the
	// migrations so a changed ontology reaches an existing database.
	if _, err := s.Pool.Exec(ctx, vocabularySQL); err != nil {
		return fmt.Errorf("vocabulary: %w", err)
	}
	names, err := fs.Glob(migrations, "migrations/*.sql")
	if err != nil {
		return err
	}
	sort.Strings(names)
	for _, n := range names {
		var done bool
		if err := s.Pool.QueryRow(ctx, `select exists(select 1 from schema_migration where name = $1)`, n).Scan(&done); err != nil {
			return err
		}
		if done {
			continue
		}
		b, _ := migrations.ReadFile(n)
		tx, err := s.Pool.Begin(ctx)
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, string(b)); err != nil {
			tx.Rollback(ctx)
			return fmt.Errorf("%s: %w", n, err)
		}
		if _, err := tx.Exec(ctx, `insert into schema_migration (name) values ($1)`, n); err != nil {
			tx.Rollback(ctx)
			return err
		}
		if err := tx.Commit(ctx); err != nil {
			return err
		}
	}
	return nil
}

// ---- data sets and observations ----

// RegisterDataset makes sure a data set exists, with its expected cycle.
func (s *Store) RegisterDataset(ctx context.Context, source, dataset string, cycle time.Duration) error {
	_, err := s.Pool.Exec(ctx, `insert into dataset (source, dataset, expected_cycle) values ($1, $2, $3)
		on conflict (source, dataset) do update set expected_cycle = excluded.expected_cycle`, source, dataset, cycle)
	return err
}

func contentHash(b []byte) string {
	// Canonicalise through a map so key order in the export does not matter.
	var v any
	if json.Unmarshal(b, &v) == nil {
		b, _ = json.Marshal(v)
	}
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:16])
}

// WriteObservations stores one extract of a data set. A record whose content did not change only has
// its last_seen_run moved forward; nothing is ever overwritten. Records missing from the extract are
// gone from the source and drop out of the latest set.
func (s *Store) WriteObservations(ctx context.Context, runID int64, source, dataset string, obs []connectors.Observation, at time.Time) (inserted int, err error) {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx)
	for _, o := range obs {
		tag, err := tx.Exec(ctx, `insert into observation (source, dataset, external_id, payload, content_hash, observed_at, first_run, last_seen_run)
			values ($1, $2, $3, $4, $5, $6, $7, $7)
			on conflict (source, dataset, external_id, content_hash) do update set last_seen_run = excluded.last_seen_run`,
			o.Source, o.Dataset, o.ExternalID, []byte(o.Payload), contentHash(o.Payload), o.ObservedAt, runID)
		if err != nil {
			return 0, err
		}
		_ = tag
	}
	if err := tx.QueryRow(ctx, `select count(*) from observation where source = $1 and dataset = $2 and first_run = $3`,
		source, dataset, runID).Scan(&inserted); err != nil {
		return 0, err
	}
	if _, err := tx.Exec(ctx, `update dataset set last_run = $3, last_extract_at = $4, records = $5 where source = $1 and dataset = $2`,
		source, dataset, runID, at, len(obs)); err != nil {
		return 0, err
	}
	return inserted, tx.Commit(ctx)
}

// LatestObservations returns, for every data set, the newest content of every record the last extract
// returned.
func (s *Store) LatestObservations(ctx context.Context) ([]connectors.Observation, error) {
	rows, err := s.Pool.Query(ctx, `
		select distinct on (o.source, o.dataset, o.external_id) o.id, o.source, o.dataset, o.external_id, o.payload, o.observed_at
		from observation o join dataset d on d.source = o.source and d.dataset = o.dataset
		where o.last_seen_run = d.last_run
		order by o.source, o.dataset, o.external_id, o.id desc`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []connectors.Observation
	for rows.Next() {
		var o connectors.Observation
		var payload []byte
		if err := rows.Scan(&o.ID, &o.Source, &o.Dataset, &o.ExternalID, &payload, &o.ObservedAt); err != nil {
			return nil, err
		}
		o.Payload = payload
		out = append(out, o)
	}
	return out, rows.Err()
}

// ---- inputs of a build ----

// LatestScores returns the overall score of the last scorecard of every data set.
func (s *Store) LatestScores(ctx context.Context) (map[string]float64, error) {
	rows, err := s.Pool.Query(ctx, `select distinct on (source, dataset) source, dataset, overall from scorecard order by source, dataset, id desc`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]float64{}
	for rows.Next() {
		var src, ds string
		var v float64
		if err := rows.Scan(&src, &ds, &v); err != nil {
			return nil, err
		}
		out[src+"/"+ds] = v
	}
	return out, rows.Err()
}

// Judgements returns the newest judgement per candidate.
func (s *Store) Judgements(ctx context.Context) (map[string]resolver.Judgement, error) {
	rows, err := s.Pool.Query(ctx, `select distinct on (candidate) result from judgement order by candidate, id desc`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]resolver.Judgement{}
	for rows.Next() {
		var b []byte
		if err := rows.Scan(&b); err != nil {
			return nil, err
		}
		var j resolver.Judgement
		if err := json.Unmarshal(b, &j); err != nil {
			return nil, err
		}
		out[j.Candidate] = j
	}
	return out, rows.Err()
}

// SaveJudgement caches a model's answer.
func (s *Store) SaveJudgement(ctx context.Context, j resolver.Judgement) error {
	b, _ := json.Marshal(j)
	_, err := s.Pool.Exec(ctx, `insert into judgement (candidate, input_hash, provider, model, prompt_version, result) values ($1, $2, $3, $4, $5, $6)`,
		j.Candidate, j.InputHash, j.Provider, j.Model, j.PromptVersion, b)
	return err
}

// Decisions returns the confirmations and rejections, oldest first.
func (s *Store) Decisions(ctx context.Context) ([]resolver.Decision, error) {
	rows, err := s.Pool.Query(ctx, `select target_type, target_id, action, actor, coalesce(evidence_hash, ''), coalesce(note, ''), at
		from decision_log where target_type in ('edge', 'alias') order by id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []resolver.Decision
	for rows.Next() {
		var d resolver.Decision
		if err := rows.Scan(&d.TargetType, &d.TargetID, &d.Action, &d.Actor, &d.EvidenceHash, &d.Note, &d.At); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

// LogDecision appends to the decision log.
func (s *Store) LogDecision(ctx context.Context, targetType, targetID, action, actor, note, evidenceHash string) error {
	_, err := s.Pool.Exec(ctx, `insert into decision_log (target_type, target_id, action, actor, note, evidence_hash) values ($1, $2, $3, $4, nullif($5, ''), nullif($6, ''))`,
		targetType, targetID, action, actor, note, evidenceHash)
	return err
}

// ---- writing a build ----

// WriteModel replaces the model with a new build in one transaction. Readers keep seeing the previous
// model until the commit. Standing comes from the data sets: an entity is primary when one of the data
// sets it is known by has been declared primary.
func (s *Store) WriteModel(ctx context.Context, buildID int64, m *resolver.Model) error {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `select pg_advisory_xact_lock(4711)`); err != nil {
		return err
	}
	for _, q := range []string{`delete from edge`, `delete from alias`, `delete from record_status`, `delete from entity`, `delete from dropped`} {
		if _, err := tx.Exec(ctx, q); err != nil {
			return err
		}
	}
	primary := map[string]bool{}
	rows, err := tx.Query(ctx, `select source || ':' || dataset from dataset where standing = 'primary'`)
	if err != nil {
		return err
	}
	for rows.Next() {
		var k string
		rows.Scan(&k)
		primary[k] = true
	}
	rows.Close()
	standing := map[string]string{}
	for _, a := range m.Aliases {
		if a.Status == resolver.Active && primary[a.Source+":"+a.Dataset] {
			standing[a.EntityID] = "primary"
		}
	}

	ts := func(id string) *time.Time {
		t := m.LastSeen(id)
		if t.IsZero() {
			return nil
		}
		return &t
	}
	var ents [][]any
	for _, e := range sortedEntities(m) {
		attrs, _ := json.Marshal(e.Attributes)
		var srcs []string
		for s := range e.Sources {
			srcs = append(srcs, s)
		}
		sort.Strings(srcs)
		st := standing[e.ID]
		if st == "" {
			st = "secondary"
		}
		ents = append(ents, []any{e.ID, e.Kind, e.Name, st, attrs, srcs, ts(e.ID)})
	}
	if _, err := tx.CopyFrom(ctx, pgx.Identifier{"entity"}, []string{"id", "kind", "name", "standing", "attributes", "sources", "last_seen"}, pgx.CopyFromRows(ents)); err != nil {
		return fmt.Errorf("entities: %w", err)
	}
	var als [][]any
	for _, a := range m.Aliases {
		ev, _ := json.Marshal(a.Evidence)
		als = append(als, []any{a.ID, a.EntityID, a.Source, a.Dataset, a.ExternalID, string(a.Basis), a.Confidence, a.Status, ev, a.ConfirmedBy, a.ConfirmedAt, ts(a.ID)})
	}
	if _, err := tx.CopyFrom(ctx, pgx.Identifier{"alias"}, []string{"id", "entity_id", "source", "dataset", "external_id", "basis", "confidence", "status", "evidence", "confirmed_by", "confirmed_at", "last_seen"}, pgx.CopyFromRows(nullStrings(als, 9))); err != nil {
		return fmt.Errorf("aliases: %w", err)
	}
	// Edges go through INSERT, not COPY, so the vocabulary trigger checks every one of them.
	batch := &pgx.Batch{}
	for _, e := range sortedEdges(m) {
		ev, _ := json.Marshal(e.Evidence)
		var by *string
		if e.ConfirmedBy != "" {
			by = &e.ConfirmedBy
		}
		batch.Queue(`insert into edge (id, kind, from_id, to_id, basis, confidence, status, source, evidence, confirmed_by, confirmed_at, last_seen)
			values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`,
			e.ID, e.Kind, e.From, e.To, string(e.Basis), e.Confidence, e.Status, e.Source, ev, by, e.ConfirmedAt, ts(e.ID))
	}
	if err := tx.SendBatch(ctx, batch).Close(); err != nil {
		return fmt.Errorf("edges: %w", err)
	}
	var recs [][]any
	for _, r := range m.Records {
		var ent *string
		if r.EntityID != "" {
			e := r.EntityID
			ent = &e
		}
		recs = append(recs, []any{r.ObservationID, ent, r.Resolved})
	}
	if _, err := tx.CopyFrom(ctx, pgx.Identifier{"record_status"}, []string{"observation_id", "entity_id", "resolved"}, pgx.CopyFromRows(recs)); err != nil {
		return fmt.Errorf("records: %w", err)
	}
	for _, d := range m.Dropped {
		if _, err := tx.Exec(ctx, `insert into dropped (build_id, reason, what) values ($1, $2, $3)`, buildID, d.Reason, d.What); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

// nullStrings turns empty strings in column col into NULLs.
func nullStrings(rows [][]any, col int) [][]any {
	for _, r := range rows {
		if s, ok := r[col].(string); ok && s == "" {
			r[col] = nil
		}
	}
	return rows
}

func sortedEntities(m *resolver.Model) []*resolver.Entity {
	var out []*resolver.Entity
	for _, e := range m.Entities {
		out = append(out, e)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func sortedEdges(m *resolver.Model) []*resolver.Edge {
	var out []*resolver.Edge
	for _, e := range m.Edges {
		out = append(out, e)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}
