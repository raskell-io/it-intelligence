// Package mapping turns observations into assertions: candidate entities and candidate edges, each
// tagged with the observation it came from. One mapper per connector. Mappers are pure functions; they
// never touch the database, so they are tested with table tests against the seed files.
//
// Company-specific knowledge (label names, naming conventions) lives here and in config/mapping.yaml,
// never in connectors.
package mapping

import (
	"strings"
	"time"

	"github.com/raskell-io/it-intelligence/connectors"
)

// Basis says how a relationship was established.
type Basis string

const (
	Asserted Basis = "asserted"
	Derived  Basis = "derived"
	Inferred Basis = "inferred"
)

// Ref points at an entity, either by its natural key (the part after "ctx:Kind/") or by an alias, an id
// the entity is known by in some source ("earepo:applications" + "APP-0142").
type Ref struct {
	Kind    string
	Key     string
	AliasNS string
	AliasID string
}

// ID returns the Context Model id for a key ref, or "" for an alias ref.
func (r Ref) ID() string {
	if r.Key == "" {
		return ""
	}
	return "ctx:" + r.Kind + "/" + r.Key
}

// Alias returns the alias string of an alias ref, "earepo:applications/APP-0142".
func (r Ref) Alias() string {
	if r.AliasNS == "" {
		return ""
	}
	return r.AliasNS + "/" + r.AliasID
}

// IsZero reports whether the ref points nowhere (an unresolved identity).
func (r Ref) IsZero() bool { return r.Key == "" && r.AliasNS == "" }

// EntityAssertion says "this observation describes that entity".
//
// With a key ref it creates or updates the entity. With an alias ref it says the record is the same
// thing as an entity some other source knows (an ITSM business service carrying an EA id). With a zero
// ref (only Kind set) the identity is unknown and the resolver must decide: that is a candidate for
// inference, and Hints is what the model gets to see.
type EntityAssertion struct {
	Ref        Ref
	Name       string
	Attributes map[string]any
	Hints      map[string]string
	Primary    bool // the record's main entity; used for the resolvability score
}

// EdgeAssertion is a candidate relationship.
type EdgeAssertion struct {
	Kind     string
	From, To Ref
	Basis    Basis
}

// Counter adds to an integer attribute of an entity (incidents in the last 90 days).
type Counter struct {
	Ref   Ref
	Name  string
	Delta int
}

// Assertions is everything one observation says.
type Assertions struct {
	Observation connectors.Observation
	Entities    []EntityAssertion
	Edges       []EdgeAssertion
	Counters    []Counter
}

// OwnAlias is the alias of the observation itself: "itsm:business_services/BS-1042".
func (a *Assertions) OwnAlias() string {
	o := a.Observation
	return o.Source + ":" + o.Dataset + "/" + o.ExternalID
}

// Entity asserts an entity by natural key and returns a ref to it. The first entity asserted by a
// mapper is the record's primary entity.
func (a *Assertions) Entity(kind, key, name string, attrs map[string]any) Ref {
	r := Ref{Kind: kind, Key: key}
	a.Entities = append(a.Entities, EntityAssertion{Ref: r, Name: name, Attributes: attrs, Primary: len(a.Entities) == 0})
	return r
}

// SameAs asserts that this record is the entity known elsewhere by an alias. A zero target means the
// identity is unknown and goes to the resolver.
func (a *Assertions) SameAs(kind string, target Ref, name string, hints map[string]string) Ref {
	if target.IsZero() {
		target = Ref{Kind: kind}
	}
	a.Entities = append(a.Entities, EntityAssertion{Ref: target, Name: name, Hints: hints, Primary: len(a.Entities) == 0})
	o := a.Observation
	return Ref{Kind: kind, AliasNS: o.Source + ":" + o.Dataset, AliasID: o.ExternalID}
}

// Key returns a ref by natural key without asserting the entity exists.
func Key(kind, key string) Ref { return Ref{Kind: kind, Key: key} }

// Alias returns a ref by alias.
func Alias(kind, ns, id string) Ref { return Ref{Kind: kind, AliasNS: ns, AliasID: id} }

// Edge asserts a relationship.
func (a *Assertions) Edge(kind string, from, to Ref, basis Basis) {
	a.Edges = append(a.Edges, EdgeAssertion{Kind: kind, From: from, To: to, Basis: basis})
}

// Count adds to an integer attribute.
func (a *Assertions) Count(r Ref, name string, delta int) {
	a.Counters = append(a.Counters, Counter{Ref: r, Name: name, Delta: delta})
}

// Context is what a mapper may know besides the observation.
type Context struct {
	Now    time.Time // the reference time: "now" for the seed, the wall clock otherwise
	Config Config
}

// Mapper maps the observations of one source.
type Mapper interface {
	Source() string
	Map(ctx Context, o connectors.Observation) (Assertions, error)
	// Fields describes, per data set, which payload fields the mapper needs (completeness) and which
	// field says when the source last changed the record (freshness).
	Fields(dataset string) DatasetFields
}

// DatasetFields is the scoring contract of a data set.
type DatasetFields struct {
	Required []string
	Modified string // payload field with an RFC 3339 time; "" means observed_at
}

var mappers = map[string]Mapper{}

func register(m Mapper) { mappers[m.Source()] = m }

// For returns the mapper of a source.
func For(source string) (Mapper, bool) {
	m, ok := mappers[source]
	return m, ok
}

// Slug normalises a display name into a key: "Claims Platform" -> "claims-platform".
func Slug(s string) string {
	var b strings.Builder
	dash := false
	for _, c := range strings.ToLower(strings.TrimSpace(s)) {
		if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') {
			b.WriteRune(c)
			dash = false
		} else if !dash {
			b.WriteByte('-')
			dash = true
		}
	}
	return strings.Trim(b.String(), "-")
}

// split splits a ";"-separated export field, dropping empty parts.
func split(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ";") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}
