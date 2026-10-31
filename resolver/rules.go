package resolver

import (
	"sort"
	"strings"
	"time"

	"github.com/raskell-io/it-intelligence/mapping"
	"github.com/raskell-io/it-intelligence/ontology/generated/go/vocab"
)

// Rules are deterministic heuristics with a known precision. Each produces derived edges whose
// evidence names the rule and the edges it rests on, so `explain_fact` can show the chain. A rule never
// overrides what a source asserted: it only fills gaps.

type rule struct {
	name string
	run  func(b *builder, idx *index, precision float64)
}

var rules = []rule{
	{"realised_via_build", realisedViaBuild},
	{"same_normalised_name", sameNormalisedName},
	{"team_owns_single_app", teamOwnsSingleApp},
	{"owner_via_workloads", ownerViaWorkloads},
	{"owner_via_namespace", ownerViaNamespace},
	{"app_owner_via_namespace", appOwnerViaNamespace},
	{"exposed_via_route", exposedViaRoute},
}

// RuleNames lists the rules in the order they run.
func RuleNames() []string {
	var out []string
	for _, r := range rules {
		out = append(out, r.name)
	}
	return out
}

func (b *builder) runRules() {
	for _, r := range rules {
		p, on := b.in.Config.Rules[r.name]
		if !on {
			continue
		}
		r.run(b, b.index(), p) // re-index: each rule sees what the previous ones added
	}
}

// index is a view of the live edges (active, and pending ones where noted).
type index struct {
	out, in map[string][]*Edge // kind|id -> edges, active only
	linked  map[string]bool    // kind|to -> has an active or pending edge
}

func (b *builder) index() *index {
	ix := &index{out: map[string][]*Edge{}, in: map[string][]*Edge{}, linked: map[string]bool{}}
	ids := make([]string, 0, len(b.m.Edges))
	for id := range b.m.Edges {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		e := b.m.Edges[id]
		if e.Status == Active || e.Status == Pending {
			ix.linked[e.Kind+"|"+e.To] = true
			ix.linked[e.Kind+"|from|"+e.From] = true
		}
		if e.Status != Active {
			continue
		}
		ix.out[e.Kind+"|"+e.From] = append(ix.out[e.Kind+"|"+e.From], e)
		ix.in[e.Kind+"|"+e.To] = append(ix.in[e.Kind+"|"+e.To], e)
	}
	return ix
}

func (ix *index) Out(kind, from string) []*Edge { return ix.out[kind+"|"+from] }
func (ix *index) In(kind, to string) []*Edge    { return ix.in[kind+"|"+to] }

// best returns the most confident edge, or nil.
func best(es []*Edge) *Edge {
	var b *Edge
	for _, e := range es {
		if b == nil || e.Confidence > b.Confidence {
			b = e
		}
	}
	return b
}

func (b *builder) entitiesOf(kind string) []*Entity {
	var out []*Entity
	for _, e := range b.m.Entities {
		if e.Kind == kind {
			out = append(out, e)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func (b *builder) derive(name string, precision float64, kind, from, to string, evidence []*Edge, extra map[string]any) {
	// A derived fact is as strong as the rule's precision times the weakest fact it rests on.
	ids := []string{}
	min := 1.0
	for _, e := range evidence {
		ids = append(ids, e.ID)
		if e.Confidence < min {
			min = e.Confidence
		}
	}
	ev := map[string]any{"rule": name, "edges": ids}
	for k, v := range extra {
		ev[k] = v
	}
	b.addEdge(kind, from, to, mapping.Derived, round(precision*min), Active, "resolver", ev, time.Time{})
}

func realisedViaBuild(b *builder, ix *index, p float64) {
	for _, app := range b.entitiesOf(vocab.Application) {
		for _, dep := range ix.Out(vocab.DeployedAs, app.ID) {
			for _, bf := range ix.Out(vocab.BuiltFrom, dep.To) {
				if hasEdge(ix.Out(vocab.RealisedBy, app.ID), bf.To) {
					continue
				}
				b.derive("realised_via_build", p, vocab.RealisedBy, app.ID, bf.To, []*Edge{dep, bf}, nil)
			}
		}
	}
}

func hasEdge(es []*Edge, to string) bool {
	for _, e := range es {
		if e.To == to {
			return true
		}
	}
	return false
}

func (b *builder) normalise(name string) string {
	n := mapping.Slug(name)
	for _, s := range b.in.Config.NameSuffixes {
		n = strings.TrimSuffix(n, s)
	}
	return n
}

func sameNormalisedName(b *builder, ix *index, p float64) {
	apps := b.entitiesOf(vocab.Application)
	byName := map[string]*Entity{}
	for _, a := range apps {
		byName[appKey(a.ID)] = a
		byName[mapping.Slug(a.Name)] = a
	}
	link := func(subject *Entity, rel string) {
		if ix.linked[rel+"|"+subject.ID] {
			return
		}
		n := b.normalise(subject.Name)
		if a, ok := byName[n]; ok {
			b.derive("same_normalised_name", p, rel, a.ID, subject.ID, nil, map[string]any{"names": []string{subject.Name, a.Name}})
		}
	}
	for _, r := range b.entitiesOf(vocab.Repository) {
		link(r, vocab.RealisedBy)
	}
	for _, w := range b.entitiesOf(vocab.Workload) {
		link(w, vocab.DeployedAs)
	}
}

func teamOwnsSingleApp(b *builder, ix *index, p float64) {
	for _, r := range b.entitiesOf(vocab.Repository) {
		if ix.linked[vocab.RealisedBy+"|"+r.ID] {
			continue
		}
		own := best(ix.Out(vocab.OwnedBy, r.ID))
		if own == nil {
			continue
		}
		var apps []*Edge
		for _, e := range ix.In(vocab.OwnedBy, own.To) {
			if b.m.Entities[e.From].Kind == vocab.Application {
				apps = append(apps, e)
			}
		}
		if len(distinctFrom(apps)) == 1 {
			b.derive("team_owns_single_app", p, vocab.RealisedBy, apps[0].From, r.ID, []*Edge{own, apps[0]}, nil)
		}
	}
}

func weakest(es []*Edge) float64 {
	m := 1.0
	for _, e := range es {
		if e.Confidence < m {
			m = e.Confidence
		}
	}
	return m
}

func distinctFrom(es []*Edge) []string {
	seen := map[string]bool{}
	var out []string
	for _, e := range es {
		if !seen[e.From] {
			seen[e.From] = true
			out = append(out, e.From)
		}
	}
	return out
}

// ownerViaWorkloads: a namespace without an owner label is owned by the team that owns every
// application running in it.
func ownerViaWorkloads(b *builder, ix *index, p float64) {
	for _, ns := range b.entitiesOf(vocab.Namespace) {
		if len(ix.Out(vocab.OwnedBy, ns.ID)) > 0 {
			continue
		}
		// The evidence is the strongest chain: the one whose weakest link is most confident.
		teams := map[string][]*Edge{}
		var order []string
		for _, ri := range ix.In(vocab.RunsIn, ns.ID) {
			for _, dep := range ix.In(vocab.DeployedAs, ri.From) {
				own := best(ix.Out(vocab.OwnedBy, dep.From))
				if own == nil || b.m.Entities[own.To].Kind != vocab.Team {
					continue
				}
				chain := []*Edge{ri, dep, own}
				if teams[own.To] == nil {
					order = append(order, own.To)
					teams[own.To] = chain
				} else if weakest(chain) > weakest(teams[own.To]) {
					teams[own.To] = chain
				}
			}
		}
		if len(order) == 1 {
			b.derive("owner_via_workloads", p, vocab.OwnedBy, ns.ID, order[0], teams[order[0]], nil)
		}
	}
}

func ownerViaNamespace(b *builder, ix *index, p float64) {
	for _, w := range b.entitiesOf(vocab.Workload) {
		if len(ix.Out(vocab.OwnedBy, w.ID)) > 0 {
			continue
		}
		ri := best(ix.Out(vocab.RunsIn, w.ID))
		if ri == nil {
			continue
		}
		if own := best(ix.Out(vocab.OwnedBy, ri.To)); own != nil {
			b.derive("owner_via_namespace", p, vocab.OwnedBy, w.ID, own.To, []*Edge{ri, own}, nil)
		}
	}
}

func appOwnerViaNamespace(b *builder, ix *index, p float64) {
	for _, app := range b.entitiesOf(vocab.Application) {
		if len(ix.Out(vocab.OwnedBy, app.ID)) > 0 {
			continue
		}
		teams := map[string][]*Edge{}
		var order []string
		for _, dep := range ix.Out(vocab.DeployedAs, app.ID) {
			ri := best(ix.Out(vocab.RunsIn, dep.To))
			if ri == nil {
				continue
			}
			own := best(ix.Out(vocab.OwnedBy, ri.To))
			if own == nil {
				continue
			}
			if teams[own.To] == nil {
				order = append(order, own.To)
				teams[own.To] = []*Edge{dep, ri, own}
			}
		}
		if len(order) == 1 {
			b.derive("app_owner_via_namespace", p, vocab.OwnedBy, app.ID, order[0], teams[order[0]], nil)
		}
	}
}

func exposedViaRoute(b *builder, ix *index, p float64) {
	for _, ep := range b.entitiesOf(vocab.Endpoint) {
		for _, rt := range ix.Out(vocab.RoutesTo, ep.ID) {
			for _, dep := range ix.In(vocab.DeployedAs, rt.To) {
				if hasEdge(ix.Out(vocab.ExposedVia, dep.From), ep.ID) {
					continue
				}
				b.derive("exposed_via_route", p, vocab.ExposedVia, dep.From, ep.ID, []*Edge{rt, dep}, nil)
			}
		}
	}
}
