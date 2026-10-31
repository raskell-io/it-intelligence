// Package resolver decides which observations describe the same thing and how things relate.
//
// It builds the whole Context Model from the latest observations, in passes from cheap and certain to
// expensive and uncertain:
//
//  1. keys: natural keys and shared identifiers (an EA id in a deployment label) give asserted facts
//  2. rules: named, deterministic heuristics with a configured precision give derived facts
//  3. inference: what is left goes to a model, whose recorded judgement gives inferred facts, queued for
//     review when the model is unsure
//
// Build is a pure function of its input: the same observations, judgements and human decisions always
// give the same model. It never calls a model itself; the Judge step does that beforehand and stores the
// judgements, so a rebuild is cheap and reproducible.
package resolver

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"time"

	"github.com/raskell-io/it-intelligence/mapping"
)

// Statuses of edges and aliases.
const (
	Active   = "active"
	Pending  = "pending"
	Rejected = "rejected"
)

// SameAs is the pseudo relation of an identity link: an observation is the same thing as an entity.
const SameAs = "same_as"

// Entity is something the model concluded exists.
type Entity struct {
	ID         string
	Kind       string
	Name       string
	Attributes map[string]any
	Sources    map[string]bool // every source that mentions it
	Hints      map[string]string
}

// Alias records that an entity is known in a source by an external id.
type Alias struct {
	ID          string
	EntityID    string
	Source      string
	Dataset     string
	ExternalID  string
	Basis       mapping.Basis
	Confidence  float64
	Status      string
	Evidence    map[string]any
	ConfirmedBy string
	ConfirmedAt *time.Time
}

// AliasString is "source:dataset/external_id".
func (a *Alias) AliasString() string { return a.Source + ":" + a.Dataset + "/" + a.ExternalID }

// Edge is one relationship as one source (or the resolver) states it.
type Edge struct {
	ID          string
	Kind        string
	From, To    string
	Basis       mapping.Basis
	Confidence  float64
	Status      string
	Source      string
	Evidence    map[string]any
	ConfirmedBy string
	ConfirmedAt *time.Time
}

// Record is what became of one observation: its entity, and whether it resolved.
type Record struct {
	ObservationID int64
	Source        string
	Dataset       string
	ExternalID    string
	EntityID      string
	Resolved      bool
}

// Match is one option on a candidate's shortlist.
type Match struct {
	ID    string            `json:"id"`
	Name  string            `json:"name"`
	Hints map[string]string `json:"hints,omitempty"`
}

// Candidate is a question for the inference pass: which application, if any, does this belong to?
type Candidate struct {
	Key       string            `json:"candidate"` // the observation's alias, "githost:repositories/kestrel/pol-adm-ui"
	Relation  string            `json:"relation"`  // realised_by, deployed_as or same_as
	Subject   string            `json:"subject,omitempty"`
	Name      string            `json:"name"`
	Hints     map[string]string `json:"hints"`
	Shortlist []Match           `json:"shortlist"`
	InputHash string            `json:"input_hash"`
	Team      string            `json:"team,omitempty"` // owning team, for the review queue
	Judged    bool              `json:"judged"`
}

// Judgement is a model's answer to a candidate, as stored.
type Judgement struct {
	Candidate     string   `json:"candidate"`
	Match         string   `json:"match"` // "" means no match
	Relation      string   `json:"relation,omitempty"`
	Confidence    float64  `json:"confidence"`
	Reasons       []string `json:"reasons"`
	Provider      string   `json:"provider"`
	Model         string   `json:"model"`
	PromptVersion string   `json:"prompt_version"`
	InputHash     string   `json:"input_hash"`
}

// Decision is a person's confirmation or rejection of an inferred link.
type Decision struct {
	TargetType   string // edge or alias
	TargetID     string
	Action       string // confirm or reject
	Actor        string
	EvidenceHash string
	Note         string
	At           time.Time
}

// Dropped is something the resolver could not use, kept for the logs and the scorecards.
type Dropped struct {
	Reason string `json:"reason"`
	What   string `json:"what"`
}

// Input is everything Build needs.
type Input struct {
	Assertions  []mapping.Assertions
	SourceOrder []string           // later sources win for names and attributes
	Scores      map[string]float64 // "source/dataset" -> overall score of the last scorecard
	Judgements  map[string]Judgement
	Decisions   []Decision
	Config      Config
}

// Model is the result of a build.
type Model struct {
	Entities   map[string]*Entity
	Aliases    map[string]*Alias
	Edges      map[string]*Edge
	Records    []Record
	Candidates []Candidate
	Dropped    []Dropped

	lastSeen map[string]time.Time
}

func hash(parts ...any) string {
	b, _ := json.Marshal(parts)
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

// EdgeID is stable across rebuilds: the same kind, ends and source always give the same id.
func EdgeID(kind, from, to, source string) string { return "e_" + hash(kind, from, to, source)[:12] }

// AliasID is stable across rebuilds.
func AliasID(entity, alias string) string { return "a_" + hash(entity, alias)[:12] }
