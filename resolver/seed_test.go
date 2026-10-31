package resolver_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/raskell-io/it-intelligence/connectors"
	_ "github.com/raskell-io/it-intelligence/connectors/cluster"
	_ "github.com/raskell-io/it-intelligence/connectors/earepo"
	_ "github.com/raskell-io/it-intelligence/connectors/gateway"
	_ "github.com/raskell-io/it-intelligence/connectors/githost"
	_ "github.com/raskell-io/it-intelligence/connectors/itsm"
	_ "github.com/raskell-io/it-intelligence/connectors/waf"
	"github.com/raskell-io/it-intelligence/mapping"
	"github.com/raskell-io/it-intelligence/resolver"
	"github.com/raskell-io/it-intelligence/resolver/provider"
)

var ref = time.Date(2026, 11, 12, 3, 0, 0, 0, time.UTC)

// seedModel runs the whole pipeline in memory: extract every seed source, map, build, judge, build.
func seedModel(t *testing.T) *resolver.Model {
	t.Helper()
	ctx := context.Background()
	mctx := mapping.Context{Now: ref, Config: mapping.DefaultConfig()}
	var as []mapping.Assertions
	var id int64
	for _, src := range []string{"earepo", "itsm", "githost", "cluster", "gateway", "waf"} {
		c, err := connectors.New(src, connectors.Config{Mode: "file", Dir: filepath.Join("..", "seed", src), Now: ref})
		if err != nil {
			t.Fatal(err)
		}
		m, _ := mapping.For(src)
		for _, ds := range c.Datasets() {
			for o, err := range c.Extract(ctx, ds, time.Time{}) {
				if err != nil {
					t.Fatal(err)
				}
				id++
				o.ID = id
				a, err := m.Map(mctx, o)
				if err != nil {
					t.Fatal(err)
				}
				as = append(as, a)
			}
		}
	}
	in := resolver.Input{Assertions: as, SourceOrder: []string{"cluster", "gateway", "waf", "githost", "itsm", "earepo"},
		Config: resolver.DefaultConfig(), Judgements: map[string]resolver.Judgement{}}
	m := resolver.Build(in)
	fx, err := provider.NewFixture(filepath.Join("..", "seed", "judgements.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range m.Candidates {
		j, _ := fx.Judge(ctx, c)
		in.Judgements[c.Key] = j
	}
	return resolver.Build(in)
}

func TestSeedModel(t *testing.T) {
	m := seedModel(t)
	counts := map[string]int{}
	for _, e := range m.Entities {
		counts[e.Kind]++
	}
	t.Logf("entities %v, edges %d, aliases %d, candidates %d, dropped %d", counts, len(m.Edges), len(m.Aliases), len(m.Candidates), len(m.Dropped))
	for _, d := range m.Dropped {
		t.Logf("dropped: %s: %s", d.Reason, d.What)
	}
	find := func(kind, from, to string) *resolver.Edge {
		for _, e := range m.Edges {
			if e.Kind == kind && e.From == from && e.To == to {
				return e
			}
		}
		return nil
	}
	e := find("realised_by", "ctx:Application/policy-admin", "ctx:Repository/kestrel/pol-adm-ui")
	if e == nil || e.Status != resolver.Pending || e.Confidence != 0.82 {
		t.Errorf("pol-adm-ui should be pending at 0.82, got %+v", e)
	}
	e = find("owned_by", "ctx:Namespace/claims-prod", "ctx:Team/claims-platform")
	if e == nil || e.Basis != mapping.Derived {
		t.Errorf("claims-prod should be owned by claims-platform, derived; got %+v", e)
	} else {
		t.Logf("claims-prod owner: %.2f %v", e.Confidence, e.Evidence)
	}
	pas := false
	for _, a := range m.Aliases {
		if a.AliasString() == "itsm:business_services/BS-1042" && a.EntityID == "ctx:Application/policy-admin" && a.Status == resolver.Active {
			pas = true
		}
	}
	if !pas {
		t.Error("PAS should be identified as policy-admin")
	}
	if n, _ := m.Entities["ctx:Application/policy-admin"].Attributes["incidents_90d"].(int); n != 3 {
		t.Errorf("policy-admin incidents_90d = %v, want 3", m.Entities["ctx:Application/policy-admin"].Attributes["incidents_90d"])
	}
	if _, ok := m.Entities["ctx:Application/south-integration-hub"]; !ok {
		t.Error("south-integration-hub should exist")
	}
}
