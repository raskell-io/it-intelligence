package resolver

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/raskell-io/it-intelligence/connectors"
	"github.com/raskell-io/it-intelligence/mapping"
	"github.com/raskell-io/it-intelligence/ontology/generated/go/vocab"
)

type builder struct {
	in         Input
	m          *Model
	aliases    map[string]string // alias string -> entity id, active aliases only
	rank       map[string]int
	refsOK     map[int64]bool // observation -> every reference it makes resolved
	retry      []pendingRef
	identified map[string]string // own alias of an identity candidate -> entity, once inference settled it
	primary    map[int64]string
	decisions  map[string]Decision
	lastSeen   map[string]time.Time
	domain     map[string]map[string]bool
	rng        map[string]map[string]bool
}

// Build computes the whole model. See the package documentation for the passes.
func Build(in Input) *Model {
	b := &builder{
		in:         in,
		m:          &Model{Entities: map[string]*Entity{}, Aliases: map[string]*Alias{}, Edges: map[string]*Edge{}},
		aliases:    map[string]string{},
		rank:       map[string]int{},
		refsOK:     map[int64]bool{},
		identified: map[string]string{},
		primary:    map[int64]string{},
		decisions:  map[string]Decision{},
		lastSeen:   map[string]time.Time{},
		domain:     map[string]map[string]bool{},
		rng:        map[string]map[string]bool{},
	}
	for i, s := range in.SourceOrder {
		b.rank[s] = i
	}
	for _, d := range in.Decisions { // in time order, so the latest decision wins
		b.decisions[d.TargetID] = d
	}
	for _, r := range vocab.Relations {
		b.domain[r.Name], b.rng[r.Name] = set(r.Domain), set(r.Range)
	}
	as := append([]mapping.Assertions(nil), in.Assertions...)
	sort.SliceStable(as, func(i, j int) bool {
		oi, oj := as[i].Observation, as[j].Observation
		if b.rank[oi.Source] != b.rank[oj.Source] {
			return b.rank[oi.Source] < b.rank[oj.Source]
		}
		if oi.Dataset != oj.Dataset {
			return oi.Dataset < oj.Dataset
		}
		return oi.ExternalID < oj.ExternalID
	})
	for _, a := range as {
		b.refsOK[a.Observation.ID] = true
	}

	b.entities(as)
	identity := b.sameAs(as)
	b.counters(as)
	b.assertedEdges(as)
	b.runRules()
	b.inference(identity)
	b.retryRefs() // references to records that only inference could identify
	b.runRules()  // facts that follow from inferred links
	b.records(as)
	b.finish()
	return b.m
}

func set(xs []string) map[string]bool {
	m := map[string]bool{}
	for _, x := range xs {
		m[x] = true
	}
	return m
}

func (b *builder) seen(id string, t time.Time) {
	if t.After(b.lastSeen[id]) {
		b.lastSeen[id] = t
	}
}

// ---- pass 1: keys ----

func (b *builder) entities(as []mapping.Assertions) {
	for _, a := range as {
		o := a.Observation
		for _, ea := range a.Entities {
			if ea.Ref.Key == "" {
				continue
			}
			id := ea.Ref.ID()
			e := b.m.Entities[id]
			if e == nil {
				e = &Entity{ID: id, Kind: ea.Ref.Kind, Name: ea.Name, Attributes: map[string]any{}, Sources: map[string]bool{}}
				b.m.Entities[id] = e
			}
			if ea.Name != "" {
				e.Name = ea.Name // sources are in rank order, so the most authoritative name wins
			}
			for k, v := range ea.Attributes {
				e.Attributes[k] = v
			}
			if ea.Hints != nil {
				e.Hints = ea.Hints
			}
			e.Sources[o.Source] = true
			b.seen(id, o.ObservedAt)
			if ea.Primary {
				b.primary[o.ID] = id
				b.addAlias(id, o.Source, o.Dataset, o.ExternalID, mapping.Asserted, b.assertedConfidence(o.Source, o.Dataset),
					map[string]any{"observations": []int64{o.ID}}, o.ObservedAt)
			}
		}
	}
}

func (b *builder) addAlias(entity, source, dataset, ext string, basis mapping.Basis, conf float64, ev map[string]any, t time.Time) *Alias {
	al := &Alias{EntityID: entity, Source: source, Dataset: dataset, ExternalID: ext, Basis: basis,
		Confidence: conf, Status: Active, Evidence: ev}
	al.ID = AliasID(entity, al.AliasString())
	b.m.Aliases[al.ID] = al
	b.aliases[al.AliasString()] = entity
	b.seen(al.ID, t)
	return al
}

// sameAs resolves records that say "I am the entity known elsewhere as X". Records without X become
// identity candidates for the inference pass.
func (b *builder) sameAs(as []mapping.Assertions) []mapping.Assertions {
	var unresolved []mapping.Assertions
	for _, a := range as {
		o := a.Observation
		for _, ea := range a.Entities {
			if ea.Ref.Key != "" {
				continue
			}
			if ea.Ref.AliasNS == "" {
				unresolved = append(unresolved, a)
				b.refsOK[o.ID] = false
				continue
			}
			target, ok := b.aliases[ea.Ref.Alias()]
			if !ok {
				b.drop("dangling same_as", fmt.Sprintf("%s -> %s", a.OwnAlias(), ea.Ref.Alias()))
				b.refsOK[o.ID] = false
				continue
			}
			b.addAlias(target, o.Source, o.Dataset, o.ExternalID, mapping.Asserted, b.assertedConfidence(o.Source, o.Dataset),
				map[string]any{"observations": []int64{o.ID}, "key": ea.Ref.Alias()}, o.ObservedAt)
			b.m.Entities[target].Sources[o.Source] = true
			if ea.Primary {
				b.primary[o.ID] = target
			}
		}
	}
	return unresolved
}

func (b *builder) counters(as []mapping.Assertions) {
	for _, a := range as {
		for _, c := range a.Counters {
			if !b.count(c) {
				b.retry = append(b.retry, pendingRef{obs: a.Observation, own: a.OwnAlias(), counter: &c})
			}
		}
	}
	// Applications without a counter get an explicit zero once IT service management has spoken about
	// them, so "no incidents" and "nobody knows" stay different.
	for _, e := range b.m.Entities {
		if e.Kind == vocab.Application && e.Sources["itsm"] {
			if _, ok := e.Attributes["incidents_90d"]; !ok {
				e.Attributes["incidents_90d"] = 0
			}
		}
	}
}

func (b *builder) count(c mapping.Counter) bool {
	id, ok := b.resolve(c.Ref)
	if !ok {
		return false
	}
	n, _ := b.m.Entities[id].Attributes[c.Name].(int)
	b.m.Entities[id].Attributes[c.Name] = n + c.Delta
	return true
}

// pendingRef is an edge or counter whose reference did not resolve on the first pass.
type pendingRef struct {
	obs     connectors.Observation
	own     string
	edge    *mapping.EdgeAssertion
	counter *mapping.Counter
}

// retryRefs tries the failed references again, after inference may have identified their targets.
// What still fails is dropped and marks its record as unresolved.
func (b *builder) retryRefs() {
	for _, p := range b.retry {
		if p.counter != nil {
			if !b.count(*p.counter) {
				b.refsOK[p.obs.ID] = false
				b.drop("dangling counter", p.own+" "+p.counter.Name+" -> "+refString(p.counter.Ref))
			}
			continue
		}
		if !b.edge(p.obs, *p.edge) {
			b.refsOK[p.obs.ID] = false
			b.drop("dangling edge", fmt.Sprintf("%s %s: %s -> %s", p.own, p.edge.Kind, refString(p.edge.From), refString(p.edge.To)))
		}
	}
	b.retry = nil
}

func (b *builder) resolve(r mapping.Ref) (string, bool) {
	if r.Key != "" {
		id := r.ID()
		_, ok := b.m.Entities[id]
		return id, ok
	}
	if r.AliasNS != "" {
		id, ok := b.aliases[r.Alias()]
		return id, ok
	}
	return "", false
}

func (b *builder) assertedConfidence(source, dataset string) float64 {
	s, ok := b.in.Scores[source+"/"+dataset]
	if !ok {
		s = b.in.Config.DefaultScore
	}
	return round(b.in.Config.AssertedBase + b.in.Config.AssertedSpan*s)
}

func round(f float64) float64 { return float64(int(f*100+0.5)) / 100 }

func (b *builder) assertedEdges(as []mapping.Assertions) {
	for _, a := range as {
		o := a.Observation
		for _, ea := range a.Edges {
			if !b.edge(o, ea) {
				b.retry = append(b.retry, pendingRef{obs: o, own: a.OwnAlias(), edge: &ea})
			}
		}
	}
}

func (b *builder) edge(o connectors.Observation, ea mapping.EdgeAssertion) bool {
	from, ok1 := b.resolve(ea.From)
	to, ok2 := b.resolve(ea.To)
	if !ok1 || !ok2 {
		return false
	}
	e := b.addEdge(ea.Kind, from, to, ea.Basis, b.assertedConfidence(o.Source, o.Dataset), Active, o.Source,
		map[string]any{"observations": []int64{o.ID}}, o.ObservedAt)
	if e != nil {
		b.m.Entities[from].Sources[o.Source] = true
		b.m.Entities[to].Sources[o.Source] = true
	}
	return true
}

func refString(r mapping.Ref) string {
	if r.Key != "" {
		return r.ID()
	}
	return r.Alias()
}

// addEdge adds or merges an edge. It enforces the ontology's domain and range.
func (b *builder) addEdge(kind, from, to string, basis mapping.Basis, conf float64, status, source string, ev map[string]any, t time.Time) *Edge {
	fk, tk := b.m.Entities[from].Kind, b.m.Entities[to].Kind
	if !b.domain[kind][fk] || !b.rng[kind][tk] {
		b.drop("outside the ontology", fmt.Sprintf("%s %s(%s) -> %s(%s)", kind, from, fk, to, tk))
		return nil
	}
	id := EdgeID(kind, from, to, source)
	if e, ok := b.m.Edges[id]; ok {
		if obs, ok := ev["observations"].([]int64); ok {
			prev, _ := e.Evidence["observations"].([]int64)
			e.Evidence["observations"] = append(prev, obs...)
		}
		if conf > e.Confidence {
			e.Confidence = conf
		}
		b.seen(id, t)
		return e
	}
	e := &Edge{ID: id, Kind: kind, From: from, To: to, Basis: basis, Confidence: conf, Status: status, Source: source, Evidence: ev}
	b.m.Edges[id] = e
	b.seen(id, t)
	return e
}

func (b *builder) drop(reason, what string) {
	b.m.Dropped = append(b.m.Dropped, Dropped{Reason: reason, What: what})
}

// ---- records and finishing ----

func (b *builder) records(as []mapping.Assertions) {
	for _, a := range as {
		o := a.Observation
		if id, ok := b.identified[a.OwnAlias()]; ok {
			b.primary[o.ID] = id
			b.refsOK[o.ID] = true
		}
		// A record resolves when its identity and every reference it makes landed on an entity.
		// Whether other sources agree is a different signal (agreement).
		ent := b.primary[o.ID]
		ok := b.refsOK[o.ID]
		b.m.Records = append(b.m.Records, Record{ObservationID: o.ID, Source: o.Source, Dataset: o.Dataset,
			ExternalID: o.ExternalID, EntityID: ent, Resolved: ok})
	}
}

// LastSeen returns when an entity, alias or edge was last confirmed by an observation.
func (m *Model) LastSeen(id string) time.Time { return m.lastSeen[id] }

func (b *builder) finish() {
	// Derived and inferred facts are as recent as the newest fact they rest on.
	for _, e := range b.m.Edges {
		if e.Source != "resolver" {
			continue
		}
		t := b.lastSeen[e.From]
		if u := b.lastSeen[e.To]; u.After(t) {
			t = u
		}
		b.seen(e.ID, t)
	}
	b.m.lastSeen = b.lastSeen
	sort.Slice(b.m.Candidates, func(i, j int) bool { return b.m.Candidates[i].Key < b.m.Candidates[j].Key })
}

// ActiveOut returns the active edges of a kind leaving an entity, sorted by id.
func (m *Model) ActiveOut(kind, from string) []*Edge {
	var out []*Edge
	for _, e := range m.Edges {
		if e.Kind == kind && e.From == from && e.Status == Active {
			out = append(out, e)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func appKey(id string) string { return strings.TrimPrefix(id, "ctx:Application/") }
