// Package verify checks the running system against the seed's answer key (seed/truth.yaml) and against
// the concrete claims the companion article makes. CI runs it on every pull request: if the code and the
// article drift apart, it says so.
package verify

import (
	"context"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	yaml "go.yaml.in/yaml/v3"

	"github.com/raskell-io/it-intelligence/api/query"
	"github.com/raskell-io/it-intelligence/artifacts"
)

type truthEdge struct {
	Kind string `yaml:"kind"`
	From string `yaml:"from"`
	To   string `yaml:"to"`
}

type truth struct {
	Entities []struct {
		ID      string   `yaml:"id"`
		Aliases []string `yaml:"aliases"`
	} `yaml:"entities"`
	Edges     []truthEdge `yaml:"edges"`
	NotLinked []truthEdge `yaml:"not_linked"`
	Findings  []struct {
		Insight  string   `yaml:"insight"`
		Entities []string `yaml:"entities"`
	} `yaml:"findings"`
}

// Baseline is the minimum resolver quality CI accepts (config/verify.yaml).
type Baseline struct {
	EdgePrecision  float64 `yaml:"edge_precision"`
	EdgeRecall     float64 `yaml:"edge_recall"`
	AliasPrecision float64 `yaml:"alias_precision"`
	AliasRecall    float64 `yaml:"alias_recall"`
}

// Report collects results.
type Report struct {
	w      io.Writer
	failed int
}

func (r *Report) check(ok bool, name, detail string) {
	mark := "PASS"
	if !ok {
		mark = "FAIL"
		r.failed++
	}
	fmt.Fprintf(r.w, "  %s  %s", mark, name)
	if detail != "" {
		fmt.Fprintf(r.w, "  (%s)", detail)
	}
	fmt.Fprintln(r.w)
}

// Failed reports whether any check failed.
func (r *Report) Failed() bool { return r.failed > 0 }

// Run executes the checks.
func Run(ctx context.Context, q *query.Q, truthPath, baselinePath string, resolverChecks, articleChecks bool, w io.Writer) (*Report, error) {
	b, err := os.ReadFile(truthPath)
	if err != nil {
		return nil, err
	}
	var t truth
	if err := yaml.Unmarshal(b, &t); err != nil {
		return nil, err
	}
	var base Baseline
	if b, err := os.ReadFile(baselinePath); err == nil {
		yaml.Unmarshal(b, &base)
	}
	r := &Report{w: w}
	if resolverChecks {
		fmt.Fprintln(w, "resolver against seed/truth.yaml")
		if err := r.resolver(ctx, q, &t, base); err != nil {
			return nil, err
		}
	}
	if articleChecks {
		fmt.Fprintln(w, "claims of the article against the running system")
		if err := r.article(ctx, q, &t); err != nil {
			return nil, err
		}
	}
	return r, nil
}

func (r *Report) resolver(ctx context.Context, q *query.Q, t *truth, base Baseline) error {
	want := map[string]bool{}
	kinds := map[string]bool{}
	for _, e := range t.Edges {
		want[e.Kind+"|"+e.From+"|"+e.To] = true
		kinds[e.Kind] = true
	}
	rows, err := q.Pool.Query(ctx, `select kind, from_id, to_id, basis, coalesce(evidence->>'rule', '') from edge where status = 'active'`)
	if err != nil {
		return err
	}
	got := map[string]bool{}
	type stat struct{ n, ok int }
	byBasis, byRule := map[string]*stat{}, map[string]*stat{}
	for rows.Next() {
		var k, f, to, basis, rule string
		rows.Scan(&k, &f, &to, &basis, &rule)
		if !kinds[k] {
			continue
		}
		key := k + "|" + f + "|" + to
		hit := want[key]
		got[key] = true
		count := func(m map[string]*stat, name string) {
			if name == "" {
				return
			}
			if m[name] == nil {
				m[name] = &stat{}
			}
			m[name].n++
			if hit {
				m[name].ok++
			}
		}
		count(byBasis, basis)
		count(byRule, rule)
	}
	rows.Close()
	tp := 0
	for k := range got {
		if want[k] {
			tp++
		}
	}
	prec, rec := frac(tp, len(got)), frac(tp, len(want))
	r.check(prec >= base.EdgePrecision, "edge precision", fmt.Sprintf("%.3f, %d of %d active relationships are true; baseline %.3f", prec, tp, len(got), base.EdgePrecision))
	r.check(rec >= base.EdgeRecall, "edge recall", fmt.Sprintf("%.3f, %d of %d true relationships found; baseline %.3f", rec, tp, len(want), base.EdgeRecall))
	for _, name := range sortedKeys(byBasis) {
		s := byBasis[name]
		fmt.Fprintf(r.w, "        basis %-9s precision %.3f (%d of %d)\n", name, frac(s.ok, s.n), s.ok, s.n)
	}
	for _, name := range sortedKeys(byRule) {
		s := byRule[name]
		fmt.Fprintf(r.w, "        rule %-24s measured precision %.3f (%d of %d)\n", name, frac(s.ok, s.n), s.ok, s.n)
	}
	var missing []string
	for k := range want {
		if !got[k] {
			missing = append(missing, k)
		}
	}
	sort.Strings(missing)
	for i, m := range missing {
		if i == 8 {
			fmt.Fprintf(r.w, "        ... and %d more not found\n", len(missing)-8)
			break
		}
		fmt.Fprintf(r.w, "        not found: %s\n", strings.ReplaceAll(m, "|", " "))
	}
	for _, e := range t.NotLinked {
		var n int
		q.Pool.QueryRow(ctx, `select count(*) from edge where kind = $1 and from_id = $2 and to_id = $3 and status = 'active' and confidence >= 0.8 and source = 'resolver'`, e.Kind, e.From, e.To).Scan(&n)
		r.check(n == 0, "resolver makes no confident false link "+e.From+" "+e.Kind+" "+e.To, "")
	}

	wantAlias := map[string]string{}
	for _, e := range t.Entities {
		for _, a := range e.Aliases {
			wantAlias[a] = e.ID
		}
	}
	rows, err = q.Pool.Query(ctx, `select source || ':' || dataset || '/' || external_id, entity_id from alias
		where status = 'active' and source in ('earepo', 'itsm') and dataset in ('applications', 'business_services', 'capabilities', 'market_units')`)
	if err != nil {
		return err
	}
	atp, an := 0, 0
	seen := map[string]bool{}
	for rows.Next() {
		var a, ent string
		rows.Scan(&a, &ent)
		an++
		if wantAlias[a] == ent {
			atp++
			seen[a] = true
		}
	}
	rows.Close()
	ap, ar := frac(atp, an), frac(len(seen), len(wantAlias))
	r.check(ap >= base.AliasPrecision, "identity precision", fmt.Sprintf("%.3f; baseline %.3f", ap, base.AliasPrecision))
	r.check(ar >= base.AliasRecall, "identity recall", fmt.Sprintf("%.3f; baseline %.3f", ar, base.AliasRecall))
	return nil
}

func (r *Report) article(ctx context.Context, q *query.Q, t *truth) error {
	// "The scorecards page shows the EA repository export scoring worst on freshness."
	cards, err := q.Scorecards(ctx, false)
	if err != nil {
		return err
	}
	sum, n := map[string]float64{}, map[string]int{}
	for _, c := range cards {
		if c.Freshness != nil {
			sum[c.Source] += *c.Freshness
			n[c.Source]++
		}
	}
	worst, worstV := "", 2.0
	for s := range sum {
		if v := sum[s] / float64(n[s]); v < worstV {
			worst, worstV = s, v
		}
	}
	r.check(worst == "earepo", "the EA repository scores worst on freshness", fmt.Sprintf("worst: %s at %.2f", worst, worstV))

	// "The review queue for the claims platform team holds the inferred link from pol-adm-ui."
	items, err := q.Review(ctx, "claims-platform")
	if err != nil {
		return err
	}
	found := false
	for _, it := range items {
		if it.From == "ctx:Application/policy-admin" && it.To == "ctx:Repository/kestrel/pol-adm-ui" && it.Confidence > 0.819 && it.Confidence < 0.821 && len(it.Reasons) == 3 {
			found = true
		}
	}
	r.check(found, "claims-platform's queue holds pol-adm-ui -> policy-admin at 0.82 with three reasons", fmt.Sprintf("%d item(s) in the queue", len(items)))

	// find_owner on claims-prod answers with a path through the graph.
	o, err := q.FindOwner(ctx, "ctx:Namespace/claims-prod")
	if err != nil {
		return err
	}
	r.check(o.Owner == "ctx:Team/claims-platform" && o.Basis == "derived" && len(o.Path) == 3,
		"find_owner(claims-prod) is claims-platform, derived, with a three-step path", fmt.Sprintf("%s, %s, %.2f, %d steps", o.Owner, o.Basis, o.Confidence, len(o.Path)))

	// "everything that depends on payment-gateway, up to four hops away"
	imp, err := q.ImpactOfChange(ctx, "ctx:Application/payment-gateway", 4)
	if err != nil {
		return err
	}
	r.check(len(imp.ByDistance) >= 2, "impact_of_change(payment-gateway, 4) finds dependants at several distances", fmt.Sprintf("%d dependants in %d rings", imp.Total, len(imp.ByDistance)))

	// "Inferred edges are dashed and labelled with their confidence."
	m, err := artifacts.Mermaid(ctx, q, "ctx:Application/policy-admin")
	if err != nil {
		return err
	}
	dashed := strings.Count(m, "-.->")
	r.check(dashed == 1 && strings.Contains(m, "inferred 0.82"), "the policy-admin diagram has exactly one dashed edge, at 0.82", fmt.Sprintf("%d dashed", dashed))

	// Every planted problem is found, and nothing else.
	fs, err := q.Findings(ctx)
	if err != nil {
		return err
	}
	key := func(insight string, ents []string) string {
		e := append([]string(nil), ents...)
		sort.Strings(e)
		return insight + " " + strings.Join(e, ",")
	}
	got := map[string]bool{}
	for _, f := range fs {
		got[key(f.Insight, f.Entities)] = true
	}
	want := map[string]bool{}
	for _, f := range t.Findings {
		k := key(f.Insight, f.Entities)
		want[k] = true
		r.check(got[k], "finds "+f.Insight+": "+strings.Join(f.Entities, ", "), "")
	}
	var extra []string
	for k := range got {
		if !want[k] {
			extra = append(extra, k)
		}
	}
	sort.Strings(extra)
	r.check(len(extra) == 0, "no findings beyond the planted ones", strings.Join(extra, "; "))
	return nil
}

func frac(a, b int) float64 {
	if b == 0 {
		return 0
	}
	return float64(a) / float64(b)
}

func sortedKeys[V any](m map[string]V) []string {
	var out []string
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
