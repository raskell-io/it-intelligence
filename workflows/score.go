package workflows

import (
	"context"
	"encoding/json"
	"sort"
	"time"

	"github.com/raskell-io/it-intelligence/mapping"
	"github.com/raskell-io/it-intelligence/workflows/runner"
)

// score computes a scorecard per data set from the latest observations and the model just built:
//
//	freshness      share of records changed within the data set's expected cycle
//	completeness   share of records with every field the mapper needs
//	agreement      share of the data set's relationships that another source also asserts
//	resolvability  share of records whose identity and references resolved to entities
//
// The overall score is a weighted mean of the signals that apply. The next build starts asserted facts
// with a confidence that reflects it. Every failing record is listed, so every number can be checked.
func (d *Deps) score(ctx context.Context, r runner.Run) (any, error) {
	obs, err := d.Store.LatestObservations(ctx)
	if err != nil {
		return nil, err
	}
	now := d.Config.Now()

	cycles := map[string]time.Duration{}
	rows, err := d.Store.Pool.Query(ctx, `select source, dataset, extract(epoch from expected_cycle)::bigint from dataset`)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var s, n string
		var secs int64
		rows.Scan(&s, &n, &secs)
		cycles[s+"/"+n] = time.Duration(secs) * time.Second
	}
	rows.Close()

	resolved := map[int64]bool{}
	rows, err = d.Store.Pool.Query(ctx, `select observation_id, resolved from record_status`)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var id int64
		var ok bool
		rows.Scan(&id, &ok)
		resolved[id] = ok
	}
	rows.Close()

	// Agreement: for every active asserted edge, which observations it rests on, and whether another
	// source has a row for the same relationship.
	type edgeRow struct {
		obs    []int64
		agreed bool
	}
	var edges []edgeRow
	rows, err = d.Store.Pool.Query(ctx, `
		select coalesce(e.evidence->'observations', '[]'::jsonb),
		       exists (select 1 from edge o where o.kind = e.kind and o.from_id = e.from_id and o.to_id = e.to_id
		               and o.source <> e.source and o.source <> 'resolver' and o.status = 'active')
		from edge e where e.source <> 'resolver' and e.status = 'active'`)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var raw []byte
		var er edgeRow
		rows.Scan(&raw, &er.agreed)
		json.Unmarshal(raw, &er.obs)
		edges = append(edges, er)
	}
	rows.Close()

	type card struct {
		records                    int
		fresh, complete, resolvedN int
		edges, agreed              int
		failing                    map[string][]string
	}
	cards := map[string]*card{}
	obsSet := map[int64]string{}
	for _, o := range obs {
		key := o.Source + "/" + o.Dataset
		c := cards[key]
		if c == nil {
			c = &card{failing: map[string][]string{}}
			cards[key] = c
		}
		obsSet[o.ID] = key
		c.records++
		m, _ := mapping.For(o.Source)
		f := m.Fields(o.Dataset)
		var payload map[string]any
		json.Unmarshal(o.Payload, &payload)

		modified := o.ObservedAt
		if f.Modified != "" {
			if s, ok := lookup(payload, f.Modified).(string); ok {
				if t, err := time.Parse(time.RFC3339, s); err == nil {
					modified = t
				}
			}
		}
		if now.Sub(modified) <= cycles[key] {
			c.fresh++
		} else {
			c.failing["freshness"] = appendCapped(c.failing["freshness"], o.ExternalID)
		}
		complete := true
		for _, field := range f.Required {
			if empty(lookup(payload, field)) {
				complete = false
			}
		}
		if complete {
			c.complete++
		} else {
			c.failing["completeness"] = appendCapped(c.failing["completeness"], o.ExternalID)
		}
		if resolved[o.ID] {
			c.resolvedN++
		} else {
			c.failing["resolvability"] = appendCapped(c.failing["resolvability"], o.ExternalID)
		}
	}
	for _, e := range edges {
		seen := map[string]bool{}
		for _, id := range e.obs {
			key, ok := obsSet[id]
			if !ok || seen[key] {
				continue
			}
			seen[key] = true
			cards[key].edges++
			if e.agreed {
				cards[key].agreed++
			}
		}
	}

	keys := make([]string, 0, len(cards))
	for k := range cards {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	w := d.Config.Weights
	out := map[string]float64{}
	for _, k := range keys {
		c := cards[k]
		n := float64(c.records)
		fresh, complete, resolv := ratio(c.fresh, n), ratio(c.complete, n), ratio(c.resolvedN, n)
		var agree *float64
		if c.edges > 0 {
			a := ratio(c.agreed, float64(c.edges))
			agree = &a
		}
		sum, weight := w["freshness"]*fresh+w["completeness"]*complete+w["resolvability"]*resolv,
			w["freshness"]+w["completeness"]+w["resolvability"]
		if agree != nil {
			sum += w["agreement"] * *agree
			weight += w["agreement"]
		}
		overall := round2(sum / weight)
		failing, _ := json.Marshal(c.failing)
		src, dsName := split(k)
		if _, err := d.Store.Pool.Exec(ctx, `insert into scorecard (source, dataset, run_id, records, freshness, completeness, agreement, resolvability, overall, failing, computed_at)
			values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
			src, dsName, r.ID, c.records, fresh, complete, agree, resolv, overall, failing, now); err != nil {
			return nil, err
		}
		out[k] = overall
	}
	return out, nil
}

func split(k string) (string, string) {
	for i := range k {
		if k[i] == '/' {
			return k[:i], k[i+1:]
		}
	}
	return k, ""
}

func ratio(a int, n float64) float64 {
	if n == 0 {
		return 0
	}
	return round2(float64(a) / n)
}

func round2(f float64) float64 { return float64(int(f*100+0.5)) / 100 }

func appendCapped(xs []string, x string) []string {
	if len(xs) < 500 {
		return append(xs, x)
	}
	return xs
}

// lookup reads a dotted path ("metadata.labels") from a decoded JSON object.
func lookup(m map[string]any, path string) any {
	var v any = m
	start := 0
	for i := 0; i <= len(path); i++ {
		if i == len(path) || path[i] == '.' {
			mm, ok := v.(map[string]any)
			if !ok {
				return nil
			}
			v = mm[path[start:i]]
			start = i + 1
		}
	}
	return v
}

func empty(v any) bool {
	switch x := v.(type) {
	case nil:
		return true
	case string:
		return x == ""
	case []any:
		return len(x) == 0
	case map[string]any:
		return len(x) == 0
	}
	return false
}
