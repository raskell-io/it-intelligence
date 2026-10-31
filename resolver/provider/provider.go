// Package provider asks a model which application, if any, a candidate belongs to.
//
// Three implementations: fixture (default; replays seed/judgements.jsonl, offline and deterministic),
// openai-compatible (local models through Ollama, llama.cpp or vLLM, or any compatible endpoint) and
// anthropic. The inputs are metadata and the prompt is small, so a local model works.
package provider

import (
	"bufio"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/raskell-io/it-intelligence/resolver"
)

// Provider judges one candidate.
type Provider interface {
	Name() string
	Model() string
	Judge(ctx context.Context, c resolver.Candidate) (resolver.Judgement, error)
}

// Config selects and configures a provider (config/resolver.yaml, section provider).
type Config struct {
	Kind     string `yaml:"kind"`     // fixture | openai-compatible | anthropic
	Model    string `yaml:"model"`    // model id
	BaseURL  string `yaml:"base_url"` // openai-compatible only
	Fixtures string `yaml:"fixtures"` // fixture only; default seed/judgements.jsonl
}

// New builds the configured provider.
func New(c Config) (Provider, error) {
	switch c.Kind {
	case "", "fixture":
		path := c.Fixtures
		if path == "" {
			path = "seed/judgements.jsonl"
		}
		return NewFixture(path)
	case "openai-compatible":
		return newOpenAI(c)
	case "anthropic":
		return newAnthropic(c)
	}
	return nil, fmt.Errorf("unknown provider %q", c.Kind)
}

//go:embed prompt.md
var promptTemplate string

// Prompt renders the instruction and the candidate for a model. Every provider sends the same text.
func Prompt(c resolver.Candidate) (system, user string) {
	in, _ := json.MarshalIndent(map[string]any{
		"candidate": c.Key, "relation": c.Relation, "name": c.Name, "hints": c.Hints, "shortlist": c.Shortlist,
	}, "", "  ")
	return promptTemplate, string(in)
}

// Schema is the JSON schema every provider asks the model to answer in.
var Schema = map[string]any{
	"type": "object",
	"properties": map[string]any{
		"match":      map[string]any{"type": "string", "description": "id of the matching application from the shortlist, or empty for none"},
		"confidence": map[string]any{"type": "number", "description": "0 to 1"},
		"reasons":    map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
	},
	"required":             []string{"match", "confidence", "reasons"},
	"additionalProperties": false,
}

type answer struct {
	Match      string   `json:"match"`
	Confidence float64  `json:"confidence"`
	Reasons    []string `json:"reasons"`
}

func finish(p Provider, c resolver.Candidate, raw string) (resolver.Judgement, error) {
	var a answer
	if err := json.Unmarshal([]byte(strings.TrimSpace(raw)), &a); err != nil {
		return resolver.Judgement{}, fmt.Errorf("%s: answer is not the schema: %w: %q", p.Name(), err, raw)
	}
	if a.Confidence < 0 || a.Confidence > 1 {
		return resolver.Judgement{}, fmt.Errorf("%s: confidence %v out of range", p.Name(), a.Confidence)
	}
	return resolver.Judgement{Candidate: c.Key, Match: a.Match, Relation: c.Relation, Confidence: a.Confidence,
		Reasons: a.Reasons, Provider: p.Name(), Model: p.Model(), PromptVersion: resolver.PromptVersion, InputHash: c.InputHash}, nil
}

// ---- fixture ----

// Fixture replays judgements a real model produced once (`itx resolver record`), keyed by candidate.
type Fixture struct {
	byCandidate map[string]resolver.Judgement
}

func NewFixture(path string) (*Fixture, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	fx := &Fixture{byCandidate: map[string]resolver.Judgement{}}
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		if strings.TrimSpace(sc.Text()) == "" {
			continue
		}
		var j resolver.Judgement
		if err := json.Unmarshal(sc.Bytes(), &j); err != nil {
			return nil, fmt.Errorf("%s: %w", path, err)
		}
		fx.byCandidate[j.Candidate] = j
	}
	return fx, sc.Err()
}

func (f *Fixture) Name() string  { return "fixture" }
func (f *Fixture) Model() string { return "recorded" }

func (f *Fixture) Judge(_ context.Context, c resolver.Candidate) (resolver.Judgement, error) {
	j, ok := f.byCandidate[c.Key]
	if !ok {
		j = resolver.Judgement{Reasons: []string{"no recorded judgement for this candidate"}}
	}
	j.Candidate, j.Relation, j.InputHash = c.Key, c.Relation, c.InputHash
	j.Provider, j.PromptVersion = f.Name(), resolver.PromptVersion
	if j.Model == "" || j.Model == "recorded" {
		j.Model = "recorded"
	}
	return j, nil
}

// Record writes judgements in the fixture format.
func Record(path string, js []resolver.Judgement) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	for _, j := range js {
		out := map[string]any{"candidate": j.Candidate, "match": j.Match, "relation": j.Relation,
			"confidence": j.Confidence, "reasons": j.Reasons, "model": j.Model}
		b, _ := json.Marshal(out)
		if _, err := f.Write(append(b, '\n')); err != nil {
			return err
		}
	}
	return nil
}
