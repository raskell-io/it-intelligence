// Package artifacts turns a snapshot of the model into things people read: diagrams as text, insights
// (named queries with a severity), and later drafts. Every artifact records the snapshot it came from,
// so it can be reproduced and a reader knows how current it is.
package artifacts

import (
	"bufio"
	"context"
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/raskell-io/it-intelligence/api/query"
	"github.com/raskell-io/it-intelligence/store"
)

//go:embed insights/*.sql
var insightFiles embed.FS

// Snapshot exports the active model as sorted JSON lines and records its content hash. Two snapshots of
// the same model have the same id, so diffs between snapshots show real change only.
func Snapshot(ctx context.Context, s *store.Store, dir string, now time.Time) (any, error) {
	rows, err := s.Pool.Query(ctx, `
		select json_build_object('type', 'entity', 'id', id, 'kind', kind, 'name', name, 'standing', standing, 'attributes', attributes)::text
		from entity
		union all
		select json_build_object('type', 'edge', 'id', id, 'kind', kind, 'from', from_id, 'to', to_id, 'basis', basis,
		                         'confidence', confidence, 'status', status, 'source', source)::text
		from edge where status <> 'rejected'
		order by 1`)
	if err != nil {
		return nil, err
	}
	var lines []string
	entities, edges := 0, 0
	for rows.Next() {
		var l string
		rows.Scan(&l)
		lines = append(lines, l)
		if strings.HasPrefix(l, `{"type" : "entity"`) {
			entities++
		} else {
			edges++
		}
	}
	rows.Close()
	body := strings.Join(lines, "\n") + "\n"
	h := sha256.Sum256([]byte(body))
	id := "s_" + hex.EncodeToString(h[:6])
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	path := filepath.Join(dir, id+".jsonl")
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		return nil, err
	}
	// Re-running on an unchanged model moves the snapshot to the front instead of duplicating it.
	if _, err := s.Pool.Exec(ctx, `insert into snapshot (id, created_at, entities, edges, path) values ($1, $2, $3, $4, $5)
		on conflict (id) do update set created_at = excluded.created_at`, id, now, entities, edges, path); err != nil {
		return nil, err
	}
	return map[string]any{"snapshot": id, "entities": entities, "edges": edges}, nil
}

func latestSnapshot(ctx context.Context, s *store.Store) (string, error) {
	var id string
	err := s.Pool.QueryRow(ctx, `select id from snapshot order by created_at desc, id limit 1`).Scan(&id)
	return id, err
}

// Diagrams writes one Mermaid diagram per application.
func Diagrams(ctx context.Context, s *store.Store, now time.Time) (any, error) {
	snap, err := latestSnapshot(ctx, s)
	if err != nil {
		return nil, err
	}
	q := &query.Q{Pool: s.Pool}
	apps, err := q.Search(ctx, "", "Application", 50)
	if err != nil {
		return nil, err
	}
	if _, err := s.Pool.Exec(ctx, `delete from artifact where snapshot_id = $1 and kind = 'diagram'`, snap); err != nil {
		return nil, err
	}
	for _, a := range apps {
		m, err := Mermaid(ctx, q, a.ID)
		if err != nil {
			return nil, err
		}
		meta, _ := json.Marshal(map[string]any{"view": "neighbourhood"})
		if _, err := s.Pool.Exec(ctx, `insert into artifact (kind, name, subject, snapshot_id, format, body, meta, created_at)
			values ('diagram', $1, $2, $3, 'mermaid', $4, $5, $6)`, "Neighbourhood of "+a.Name, a.ID, snap, m, meta, now); err != nil {
			return nil, err
		}
	}
	return map[string]any{"snapshot": snap, "diagrams": len(apps)}, nil
}

// Insight is one named query.
type Insight struct {
	Name, Severity, Description, SQL string
}

// LoadInsights reads the embedded insight files.
func LoadInsights() ([]Insight, error) {
	names, err := fs.Glob(insightFiles, "insights/*.sql")
	if err != nil {
		return nil, err
	}
	sort.Strings(names)
	var out []Insight
	for _, n := range names {
		b, _ := insightFiles.ReadFile(n)
		in := Insight{SQL: string(b)}
		sc := bufio.NewScanner(strings.NewReader(in.SQL))
		for sc.Scan() {
			line := sc.Text()
			if !strings.HasPrefix(line, "-- ") {
				break
			}
			k, v, _ := strings.Cut(strings.TrimPrefix(line, "-- "), ": ")
			switch k {
			case "name":
				in.Name = v
			case "severity":
				in.Severity = v
			case "description":
				in.Description = v
			}
		}
		if in.Name == "" {
			return nil, fmt.Errorf("%s: no name header", n)
		}
		out = append(out, in)
	}
	return out, nil
}

// Insights runs every insight on the current model and stores the findings with the snapshot.
func Insights(ctx context.Context, s *store.Store, now time.Time) (any, error) {
	snap, err := latestSnapshot(ctx, s)
	if err != nil {
		return nil, err
	}
	ins, err := LoadInsights()
	if err != nil {
		return nil, err
	}
	if _, err := s.Pool.Exec(ctx, `delete from finding where snapshot_id = $1`, snap); err != nil {
		return nil, err
	}
	s.Pool.Exec(ctx, `delete from artifact where snapshot_id = $1 and kind = 'insight'`, snap)
	counts := map[string]int{}
	for _, in := range ins {
		rows, err := s.Pool.Query(ctx, in.SQL)
		if err != nil {
			return nil, fmt.Errorf("insight %s: %w", in.Name, err)
		}
		type hit struct {
			Entities []string `json:"entities"`
			Message  string   `json:"message"`
		}
		var hits []hit
		for rows.Next() {
			var h hit
			if err := rows.Scan(&h.Entities, &h.Message); err != nil {
				rows.Close()
				return nil, fmt.Errorf("insight %s: %w", in.Name, err)
			}
			hits = append(hits, h)
		}
		rows.Close()
		for _, h := range hits {
			if _, err := s.Pool.Exec(ctx, `insert into finding (snapshot_id, insight, severity, entities, message) values ($1, $2, $3, $4, $5)`,
				snap, in.Name, in.Severity, h.Entities, h.Message); err != nil {
				return nil, err
			}
		}
		body, _ := json.MarshalIndent(map[string]any{"insight": in.Name, "description": in.Description, "severity": in.Severity, "findings": hits}, "", "  ")
		meta, _ := json.Marshal(map[string]any{"severity": in.Severity, "findings": len(hits), "description": in.Description})
		s.Pool.Exec(ctx, `insert into artifact (kind, name, snapshot_id, format, body, meta, created_at) values ('insight', $1, $2, 'json', $3, $4, $5)`,
			in.Name, snap, string(body), meta, now)
		counts[in.Name] = len(hits)
	}
	return map[string]any{"snapshot": snap, "findings": counts}, nil
}
