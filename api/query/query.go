// Package query is the one query layer of the platform. REST, MCP and the artifact generators all call
// it, so an agent and a person looking at the same entity see the same facts. Every answer that states
// a relationship carries its basis, confidence, source and edge id, so it can be explained.
package query

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Q runs queries against the store.
type Q struct {
	Pool *pgxpool.Pool
}

// ErrNotFound is returned for unknown ids.
var ErrNotFound = errors.New("not found")

// EntitySummary is one line about an entity.
type EntitySummary struct {
	ID      string `json:"id"`
	Kind    string `json:"kind"`
	Name    string `json:"name"`
	Summary string `json:"summary,omitempty"`
}

// EdgeRef is a relationship as seen from one entity.
type EdgeRef struct {
	Edge       string  `json:"edge"`
	Kind       string  `json:"kind"`
	Direction  string  `json:"direction"` // out or in
	Other      string  `json:"other"`
	OtherName  string  `json:"other_name"`
	Basis      string  `json:"basis"`
	Confidence float64 `json:"confidence"`
	Status     string  `json:"status"`
	Source     string  `json:"source"`
}

// Entity is an entity with its direct edges.
type Entity struct {
	ID         string         `json:"id"`
	Kind       string         `json:"kind"`
	Name       string         `json:"name"`
	Standing   string         `json:"standing"`
	Attributes map[string]any `json:"attributes"`
	Sources    []string       `json:"sources"`
	LastSeen   *time.Time     `json:"last_seen,omitempty"`
	Edges      []EdgeRef      `json:"edges"`
}

// Search finds entities by name or id. Exact and prefix matches rank first.
func (q *Q) Search(ctx context.Context, text, kind string, limit int) ([]EntitySummary, error) {
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	t := strings.ToLower(strings.TrimSpace(text))
	rows, err := q.Pool.Query(ctx, `
		select id, kind, name, coalesce(attributes->>'description', '') from entity
		where ($2 = '' or kind = $2)
		  and ($1 = '' or lower(name) like '%' || $1 || '%' or lower(id) like '%' || $1 || '%'
		       or lower(coalesce(attributes->>'description', '')) like '%' || $1 || '%')
		order by (lower(name) = $1 or lower(id) like '%/' || $1) desc, (lower(name) like $1 || '%') desc, kind, name
		limit $3`, t, kind, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []EntitySummary{}
	for rows.Next() {
		var e EntitySummary
		if err := rows.Scan(&e.ID, &e.Kind, &e.Name, &e.Summary); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// GetEntity returns an entity, its attributes, its standing and its direct edges (rejected ones left out).
func (q *Q) GetEntity(ctx context.Context, id string) (*Entity, error) {
	var e Entity
	var attrs []byte
	err := q.Pool.QueryRow(ctx, `select id, kind, name, standing, attributes, sources, last_seen from entity where id = $1`, id).
		Scan(&e.ID, &e.Kind, &e.Name, &e.Standing, &attrs, &e.Sources, &e.LastSeen)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("entity %s: %w", id, ErrNotFound)
	}
	if err != nil {
		return nil, err
	}
	json.Unmarshal(attrs, &e.Attributes)
	e.Edges, err = q.edgesOf(ctx, id)
	return &e, err
}

func (q *Q) edgesOf(ctx context.Context, id string) ([]EdgeRef, error) {
	rows, err := q.Pool.Query(ctx, `
		select e.id, e.kind, 'out', e.to_id, o.name, e.basis, e.confidence, e.status, e.source
		from edge e join entity o on o.id = e.to_id where e.from_id = $1 and e.status <> 'rejected'
		union all
		select e.id, e.kind, 'in', e.from_id, o.name, e.basis, e.confidence, e.status, e.source
		from edge e join entity o on o.id = e.from_id where e.to_id = $1 and e.status <> 'rejected'
		order by 2, 3, 4, 7 desc`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []EdgeRef{}
	for rows.Next() {
		var r EdgeRef
		if err := rows.Scan(&r.Edge, &r.Kind, &r.Direction, &r.Other, &r.OtherName, &r.Basis, &r.Confidence, &r.Status, &r.Source); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

type edgeRow struct {
	ID, Kind, From, To, Basis, Status, Source string
	Confidence                                float64
	Evidence                                  map[string]any
	ConfirmedBy                               *string
	ConfirmedAt, LastSeen                     *time.Time
}

func (q *Q) edge(ctx context.Context, id string) (*edgeRow, error) {
	var e edgeRow
	var ev []byte
	err := q.Pool.QueryRow(ctx, `select id, kind, from_id, to_id, basis, status, source, confidence, evidence, confirmed_by, confirmed_at, last_seen
		from edge where id = $1`, id).Scan(&e.ID, &e.Kind, &e.From, &e.To, &e.Basis, &e.Status, &e.Source, &e.Confidence, &ev, &e.ConfirmedBy, &e.ConfirmedAt, &e.LastSeen)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("edge %s: %w", id, ErrNotFound)
	}
	if err != nil {
		return nil, err
	}
	json.Unmarshal(ev, &e.Evidence)
	return &e, nil
}

// ---- find_owner ----

// Owner is the answer to "who owns this?".
type Owner struct {
	Entity     string          `json:"entity"`
	Owner      string          `json:"owner,omitempty"`
	OwnerName  string          `json:"owner_name,omitempty"`
	Basis      string          `json:"basis,omitempty"`
	Confidence float64         `json:"confidence,omitempty"`
	Edge       string          `json:"edge,omitempty"`
	Source     string          `json:"source,omitempty"`
	Path       []string        `json:"path,omitempty"`
	LastSeen   *time.Time      `json:"last_seen,omitempty"`
	Others     []EdgeRef       `json:"other_claims,omitempty"`
	Candidates []EntitySummary `json:"candidates,omitempty"` // when free text matched several entities
	Note       string          `json:"note,omitempty"`
}

// FindOwner returns the accountable team of an entity (by id or free text), how the ownership was
// established and how sure the model is. Free text that matches several entities returns them instead
// of a guess.
func (q *Q) FindOwner(ctx context.Context, idOrText string) (*Owner, error) {
	id := idOrText
	if !strings.HasPrefix(id, "ctx:") {
		hits, err := q.Search(ctx, idOrText, "", 5)
		if err != nil {
			return nil, err
		}
		if len(hits) == 0 {
			return nil, fmt.Errorf("nothing matches %q: %w", idOrText, ErrNotFound)
		}
		exact := []EntitySummary{}
		for _, h := range hits {
			if strings.EqualFold(h.Name, idOrText) || strings.HasSuffix(strings.ToLower(h.ID), "/"+strings.ToLower(idOrText)) {
				exact = append(exact, h)
			}
		}
		if len(exact) == 1 {
			id = exact[0].ID
		} else if len(hits) == 1 {
			id = hits[0].ID
		} else {
			return &Owner{Entity: idOrText, Candidates: hits, Note: "ambiguous: call find_owner again with one of these ids"}, nil
		}
	}
	rows, err := q.Pool.Query(ctx, `select e.id from edge e where e.from_id = $1 and e.kind = 'owned_by' and e.status = 'active'
		order by e.confidence desc, case e.basis when 'asserted' then 0 when 'derived' then 1 else 2 end, e.id`, id)
	if err != nil {
		return nil, err
	}
	var ids []string
	for rows.Next() {
		var s string
		rows.Scan(&s)
		ids = append(ids, s)
	}
	rows.Close()
	if len(ids) == 0 {
		if _, err := q.GetEntity(ctx, id); err != nil {
			return nil, err
		}
		return &Owner{Entity: id, Note: "no owner known; nothing in the sources says who owns it"}, nil
	}
	e, err := q.edge(ctx, ids[0])
	if err != nil {
		return nil, err
	}
	o := &Owner{Entity: id, Owner: e.To, Basis: e.Basis, Confidence: e.Confidence, Edge: e.ID, Source: e.Source, LastSeen: e.LastSeen}
	q.Pool.QueryRow(ctx, `select name from entity where id = $1`, e.To).Scan(&o.OwnerName)
	o.Path, err = q.path(ctx, id, e)
	if err != nil {
		return nil, err
	}
	for _, other := range ids[1:] {
		x, err := q.edge(ctx, other)
		if err == nil && x.To != e.To {
			o.Others = append(o.Others, EdgeRef{Edge: x.ID, Kind: x.Kind, Direction: "out", Other: x.To, Basis: x.Basis, Confidence: x.Confidence, Status: x.Status, Source: x.Source})
		}
	}
	return o, nil
}

// path renders how an edge was established, starting at subject: one step per edge of a derived chain.
func (q *Q) path(ctx context.Context, subject string, e *edgeRow) ([]string, error) {
	chain := []*edgeRow{e}
	if ids, ok := e.Evidence["edges"].([]any); ok && len(ids) > 0 {
		chain = nil
		for _, raw := range ids {
			x, err := q.edge(ctx, fmt.Sprint(raw))
			if err != nil {
				return nil, err
			}
			chain = append(chain, x)
		}
	}
	var out []string
	cur := subject
	for _, x := range chain {
		ann := fmt.Sprintf(" (%s, %s)", x.Basis, x.Source)
		switch cur {
		case x.To:
			out = append(out, fmt.Sprintf("%s <- %s - %s%s", cur, x.Kind, x.From, ann))
			cur = x.From
		case x.From:
			out = append(out, fmt.Sprintf("%s - %s -> %s%s", cur, x.Kind, x.To, ann))
			cur = x.To
		default:
			out = append(out, fmt.Sprintf("%s - %s -> %s%s", x.From, x.Kind, x.To, ann))
			cur = x.To
		}
	}
	return out, nil
}

// ---- impact_of_change ----

// Impact lists what depends on an entity, grouped by distance.
type Impact struct {
	Entity     string        `json:"entity"`
	Depth      int           `json:"depth"`
	ByDistance []ImpactLevel `json:"by_distance"`
	Total      int           `json:"total"`
	Truncated  bool          `json:"truncated,omitempty"`
}

// ImpactLevel is one ring of dependants.
type ImpactLevel struct {
	Distance int             `json:"distance"`
	Entities []EntitySummary `json:"entities"`
}

// ImpactOfChange follows depends_on edges backwards: everything that depends on id, up to depth hops.
func (q *Q) ImpactOfChange(ctx context.Context, id string, depth int) (*Impact, error) {
	if depth <= 0 || depth > 6 {
		depth = 4
	}
	if _, err := q.GetEntity(ctx, id); err != nil {
		return nil, err
	}
	rows, err := q.Pool.Query(ctx, impactSQL, id, depth)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	res := &Impact{Entity: id, Depth: depth, ByDistance: []ImpactLevel{}}
	levels := map[int]*ImpactLevel{}
	for rows.Next() {
		var s EntitySummary
		var d int
		if err := rows.Scan(&s.ID, &d, &s.Kind, &s.Name); err != nil {
			return nil, err
		}
		if levels[d] == nil {
			levels[d] = &ImpactLevel{Distance: d}
		}
		if res.Total < 50 {
			levels[d].Entities = append(levels[d].Entities, s)
		} else {
			res.Truncated = true
		}
		res.Total++
	}
	for d := 1; d <= depth; d++ {
		if l := levels[d]; l != nil {
			res.ByDistance = append(res.ByDistance, *l)
		}
	}
	return res, rows.Err()
}

// impactSQL is the article's query: everything that depends on an application, up to n hops away.
const impactSQL = `
with recursive impact(id, depth) as (
  select $1::text, 0
  union
  select e.from_id, i.depth + 1
  from edge e
  join impact i on e.to_id = i.id
  where e.kind = 'depends_on' and e.status = 'active' and i.depth < $2
)
select i.id, min(i.depth) as distance, x.kind, x.name
from impact i join entity x on x.id = i.id
where i.depth > 0
group by i.id, x.kind, x.name
order by distance, i.id`

// ---- explain_fact ----

// Explanation is the evidence behind one edge.
type Explanation struct {
	Edge         string           `json:"edge"`
	Fact         string           `json:"fact"`
	Basis        string           `json:"basis"`
	Confidence   float64          `json:"confidence"`
	Status       string           `json:"status"`
	Source       string           `json:"source"`
	Rule         string           `json:"rule,omitempty"`
	Chain        []string         `json:"chain,omitempty"`
	Observations []ObservationRef `json:"observations,omitempty"`
	Judgement    map[string]any   `json:"judgement,omitempty"`
	ConfirmedBy  *string          `json:"confirmed_by,omitempty"`
	ConfirmedAt  *time.Time       `json:"confirmed_at,omitempty"`
	Rejection    any              `json:"previously_rejected,omitempty"`
	LastSeen     *time.Time       `json:"last_seen,omitempty"`
}

// ObservationRef points at what a source said.
type ObservationRef struct {
	ID         int64     `json:"id"`
	Source     string    `json:"source"`
	Dataset    string    `json:"dataset"`
	ExternalID string    `json:"external_id"`
	ObservedAt time.Time `json:"observed_at"`
}

// ExplainFact returns the sources, observations, basis and confirmations behind one edge.
func (q *Q) ExplainFact(ctx context.Context, edgeID string) (*Explanation, error) {
	e, err := q.edge(ctx, edgeID)
	if err != nil {
		return nil, err
	}
	x := &Explanation{Edge: e.ID, Fact: fmt.Sprintf("%s - %s -> %s", e.From, e.Kind, e.To), Basis: e.Basis, Confidence: e.Confidence,
		Status: e.Status, Source: e.Source, ConfirmedBy: e.ConfirmedBy, ConfirmedAt: e.ConfirmedAt, LastSeen: e.LastSeen}
	if r, ok := e.Evidence["rule"].(string); ok {
		x.Rule = r
		x.Chain, err = q.path(ctx, e.From, e)
		if err != nil {
			return nil, err
		}
	}
	if j, ok := e.Evidence["judgement"].(map[string]any); ok {
		x.Judgement = map[string]any{"match": j["match"], "confidence": j["confidence"], "reasons": j["reasons"],
			"candidate": e.Evidence["candidate"], "provider": e.Evidence["provider"], "model": e.Evidence["model"], "prompt_version": e.Evidence["prompt_version"]}
	}
	x.Rejection = e.Evidence["previously_rejected"]
	if ids, ok := e.Evidence["observations"].([]any); ok {
		var nums []int64
		for _, v := range ids {
			if f, ok := v.(float64); ok {
				nums = append(nums, int64(f))
			}
		}
		rows, err := q.Pool.Query(ctx, `select id, source, dataset, external_id, observed_at from observation where id = any($1) order by id`, nums)
		if err != nil {
			return nil, err
		}
		for rows.Next() {
			var o ObservationRef
			rows.Scan(&o.ID, &o.Source, &o.Dataset, &o.ExternalID, &o.ObservedAt)
			x.Observations = append(x.Observations, o)
		}
		rows.Close()
	}
	return x, nil
}

// ---- list_decisions ----

// DecisionRef is an architecture decision that constrains an entity.
type DecisionRef struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Status    string `json:"status"`
	DecidedOn string `json:"decided_on"`
	URL       string `json:"url"`
	Edge      string `json:"edge"`
	Basis     string `json:"basis"`
}

// ListDecisions returns the architecture decisions that constrain an entity, newest first.
func (q *Q) ListDecisions(ctx context.Context, id string) ([]DecisionRef, error) {
	rows, err := q.Pool.Query(ctx, `
		select d.id, d.name, coalesce(d.attributes->>'status', ''), coalesce(d.attributes->>'decided_on', ''), coalesce(d.attributes->>'url', ''), e.id, e.basis
		from edge e join entity d on d.id = e.from_id
		where e.kind = 'constrains' and e.to_id = $1 and e.status = 'active'
		order by d.attributes->>'decided_on' desc`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []DecisionRef{}
	for rows.Next() {
		var d DecisionRef
		if err := rows.Scan(&d.ID, &d.Title, &d.Status, &d.DecidedOn, &d.URL, &d.Edge, &d.Basis); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

// ---- review queue ----

// ReviewItem is an inferred link waiting for its owning team.
type ReviewItem struct {
	Type       string   `json:"type"` // edge or alias
	ID         string   `json:"id"`
	Relation   string   `json:"relation"`
	From       string   `json:"from"`
	FromName   string   `json:"from_name"`
	To         string   `json:"to"`
	ToName     string   `json:"to_name"`
	Confidence float64  `json:"confidence"`
	Reasons    []string `json:"reasons"`
	Team       string   `json:"team"`
	Previously any      `json:"previously_rejected,omitempty"`
}

// Review lists pending inferred links, for one team (the owner of the application) or all.
func (q *Q) Review(ctx context.Context, team string) ([]ReviewItem, error) {
	if team != "" && !strings.HasPrefix(team, "ctx:") {
		team = "ctx:Team/" + team
	}
	rows, err := q.Pool.Query(ctx, `
		with owner as (
		  select distinct on (from_id) from_id, to_id from edge where kind = 'owned_by' and status = 'active'
		  order by from_id, confidence desc)
		select 'edge', e.id, e.kind, e.from_id, f.name, e.to_id, t.name, e.confidence, e.evidence, coalesce(o.to_id, '')
		from edge e join entity f on f.id = e.from_id join entity t on t.id = e.to_id left join owner o on o.from_id = e.from_id
		where e.status = 'pending' and ($1 = '' or o.to_id = $1)
		union all
		select 'alias', a.id, 'same_as', a.source || ':' || a.dataset || '/' || a.external_id, a.external_id, a.entity_id, t.name, a.confidence, a.evidence, coalesce(o.to_id, '')
		from alias a join entity t on t.id = a.entity_id left join owner o on o.from_id = a.entity_id
		where a.status = 'pending' and ($1 = '' or o.to_id = $1)
		order by 10, 8 desc`, team)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []ReviewItem{}
	for rows.Next() {
		var r ReviewItem
		var ev []byte
		if err := rows.Scan(&r.Type, &r.ID, &r.Relation, &r.From, &r.FromName, &r.To, &r.ToName, &r.Confidence, &ev, &r.Team); err != nil {
			return nil, err
		}
		var e map[string]any
		json.Unmarshal(ev, &e)
		if j, ok := e["judgement"].(map[string]any); ok {
			if rs, ok := j["reasons"].([]any); ok {
				for _, s := range rs {
					r.Reasons = append(r.Reasons, fmt.Sprint(s))
				}
			}
		}
		r.Previously = e["previously_rejected"]
		out = append(out, r)
	}
	return out, rows.Err()
}

// ---- scorecards, data sets, artifacts, runs ----

// Scorecard is the latest score of one data set.
type Scorecard struct {
	Source        string              `json:"source"`
	Dataset       string              `json:"dataset"`
	Standing      string              `json:"standing"`
	Records       int                 `json:"records"`
	Freshness     *float64            `json:"freshness"`
	Completeness  *float64            `json:"completeness"`
	Agreement     *float64            `json:"agreement"`
	Resolvability *float64            `json:"resolvability"`
	Overall       float64             `json:"overall"`
	ComputedAt    time.Time           `json:"computed_at"`
	Failing       map[string][]string `json:"failing,omitempty"`
}

// Scorecards returns the latest scorecard per data set, worst first. withFailing adds the failing
// records behind each number.
func (q *Q) Scorecards(ctx context.Context, withFailing bool) ([]Scorecard, error) {
	rows, err := q.Pool.Query(ctx, `
		select * from (
		  select distinct on (s.source, s.dataset) s.source, s.dataset, coalesce(d.standing, 'secondary'), s.records, s.freshness, s.completeness, s.agreement, s.resolvability, s.overall, s.computed_at, s.failing
		  from scorecard s left join dataset d on d.source = s.source and d.dataset = s.dataset
		  order by s.source, s.dataset, s.id desc) x
		order by overall, source, dataset`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Scorecard{}
	for rows.Next() {
		var s Scorecard
		var failing []byte
		if err := rows.Scan(&s.Source, &s.Dataset, &s.Standing, &s.Records, &s.Freshness, &s.Completeness, &s.Agreement, &s.Resolvability, &s.Overall, &s.ComputedAt, &failing); err != nil {
			return nil, err
		}
		if withFailing {
			json.Unmarshal(failing, &s.Failing)
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// Dataset is a data set's standing.
type Dataset struct {
	Source     string     `json:"source"`
	Dataset    string     `json:"dataset"`
	Standing   string     `json:"standing"`
	Owner      *string    `json:"owner,omitempty"`
	DeclaredBy *string    `json:"declared_by,omitempty"`
	DeclaredAt *time.Time `json:"declared_at,omitempty"`
	Records    int        `json:"records"`
	LastRun    *int64     `json:"last_run,omitempty"`
}

// Datasets lists every data set.
func (q *Q) Datasets(ctx context.Context) ([]Dataset, error) {
	rows, err := q.Pool.Query(ctx, `select source, dataset, standing, owner, declared_by, declared_at, records, last_run from dataset order by source, dataset`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Dataset{}
	for rows.Next() {
		var d Dataset
		if err := rows.Scan(&d.Source, &d.Dataset, &d.Standing, &d.Owner, &d.DeclaredBy, &d.DeclaredAt, &d.Records, &d.LastRun); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

// Artifact is a generated artifact.
type Artifact struct {
	ID        int64          `json:"id"`
	Kind      string         `json:"kind"`
	Name      string         `json:"name"`
	Subject   *string        `json:"subject,omitempty"`
	Snapshot  string         `json:"snapshot"`
	Format    string         `json:"format"`
	Body      string         `json:"body,omitempty"`
	Meta      map[string]any `json:"meta,omitempty"`
	CreatedAt time.Time      `json:"created_at"`
}

// Artifacts lists the artifacts of the latest snapshot, optionally by kind and subject.
func (q *Q) Artifacts(ctx context.Context, kind, subject string, withBody bool) ([]Artifact, error) {
	rows, err := q.Pool.Query(ctx, `
		select id, kind, name, subject, snapshot_id, format, case when $3 then body else '' end, meta, created_at from artifact
		where snapshot_id = (select id from snapshot order by created_at desc, id limit 1)
		  and ($1 = '' or kind = $1) and ($2 = '' or subject = $2)
		order by kind, name`, kind, subject, withBody)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Artifact{}
	for rows.Next() {
		var a Artifact
		var meta []byte
		if err := rows.Scan(&a.ID, &a.Kind, &a.Name, &a.Subject, &a.Snapshot, &a.Format, &a.Body, &meta, &a.CreatedAt); err != nil {
			return nil, err
		}
		json.Unmarshal(meta, &a.Meta)
		out = append(out, a)
	}
	return out, rows.Err()
}

// Finding is one insight hit.
type Finding struct {
	Insight  string   `json:"insight"`
	Severity string   `json:"severity"`
	Entities []string `json:"entities"`
	Message  string   `json:"message"`
	Snapshot string   `json:"snapshot"`
}

// Findings returns the insight findings of the latest snapshot.
func (q *Q) Findings(ctx context.Context) ([]Finding, error) {
	rows, err := q.Pool.Query(ctx, `select insight, severity, entities, message, snapshot_id from finding
		where snapshot_id = (select id from snapshot order by created_at desc, id limit 1) order by insight, entities`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Finding{}
	for rows.Next() {
		var f Finding
		if err := rows.Scan(&f.Insight, &f.Severity, &f.Entities, &f.Message, &f.Snapshot); err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

// RunInfo is a workflow run with its steps.
type RunInfo struct {
	ID         int64            `json:"id"`
	Workflow   string           `json:"workflow"`
	Input      json.RawMessage  `json:"input"`
	Status     string           `json:"status"`
	Actor      *string          `json:"actor,omitempty"`
	CreatedAt  time.Time        `json:"created_at"`
	FinishedAt *time.Time       `json:"finished_at,omitempty"`
	Error      *string          `json:"error,omitempty"`
	Steps      []map[string]any `json:"steps,omitempty"`
}

// Runs lists recent runs; with steps when one id is asked for.
func (q *Q) Runs(ctx context.Context, id int64, limit int) ([]RunInfo, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	rows, err := q.Pool.Query(ctx, `select id, workflow, input, status, actor, created_at, finished_at, error from run
		where ($1 = 0 or id = $1) order by id desc limit $2`, id, limit)
	if err != nil {
		return nil, err
	}
	out := []RunInfo{}
	for rows.Next() {
		var r RunInfo
		if err := rows.Scan(&r.ID, &r.Workflow, &r.Input, &r.Status, &r.Actor, &r.CreatedAt, &r.FinishedAt, &r.Error); err != nil {
			rows.Close()
			return nil, err
		}
		out = append(out, r)
	}
	rows.Close()
	for i := range out {
		srows, err := q.Pool.Query(ctx, `select step, attempt, status, coalesce(error, ''), output, started_at, finished_at
			from run_step where run_id = $1 order by position, attempt`, out[i].ID)
		if err != nil {
			return nil, err
		}
		for srows.Next() {
			var step, status, msg string
			var attempt int
			var output []byte
			var started time.Time
			var finished *time.Time
			srows.Scan(&step, &attempt, &status, &msg, &output, &started, &finished)
			s := map[string]any{"step": step, "attempt": attempt, "status": status, "started_at": started}
			if msg != "" {
				s["error"] = msg
			}
			if finished != nil {
				s["duration_ms"] = finished.Sub(started).Milliseconds()
			}
			var o any
			if json.Unmarshal(output, &o) == nil {
				s["output"] = o
			}
			out[i].Steps = append(out[i].Steps, s)
		}
		srows.Close()
	}
	return out, nil
}

// Stats counts what the model holds, for the home page.
func (q *Q) Stats(ctx context.Context) (map[string]any, error) {
	kinds := map[string]int{}
	rows, err := q.Pool.Query(ctx, `select kind, count(*) from entity group by kind`)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var k string
		var n int
		rows.Scan(&k, &n)
		kinds[k] = n
	}
	rows.Close()
	basis := map[string]int{}
	rows, err = q.Pool.Query(ctx, `select basis || '/' || status, count(*) from edge group by 1`)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var k string
		var n int
		rows.Scan(&k, &n)
		basis[k] = n
	}
	rows.Close()
	var pending int
	q.Pool.QueryRow(ctx, `select (select count(*) from edge where status = 'pending') + (select count(*) from alias where status = 'pending')`).Scan(&pending)
	return map[string]any{"entities": kinds, "edges": basis, "pending_review": pending}, nil
}
