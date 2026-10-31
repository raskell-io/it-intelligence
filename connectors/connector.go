// Package connectors reads sources and emits observations. A connector knows nothing about the
// ontology: what it reads becomes an observation, and the mapping package decides what it means.
package connectors

import (
	"context"
	"encoding/json"
	"fmt"
	"iter"
	"sort"
	"time"
)

// Observation is what a source said about one record at a point in time.
type Observation struct {
	ID         int64 // set by the store
	Source     string
	Dataset    string
	ExternalID string
	Payload    json.RawMessage
	ObservedAt time.Time
}

// Connector reads one kind of source.
type Connector interface {
	Name() string
	Datasets() []string
	Extract(ctx context.Context, dataset string, since time.Time) iter.Seq2[Observation, error]
}

// Factory builds a connector from its configuration (a seed directory in file mode).
type Factory func(cfg Config) (Connector, error)

// Config is what every connector gets: where to read from and the clock to stamp observations with.
type Config struct {
	Mode string    // "file" (the seed) or "live"
	Dir  string    // file mode: the directory with the exports
	Now  time.Time // observation time; fixed for the seed so runs are reproducible
	Opts map[string]string
}

var registry = map[string]Factory{}

// Register makes a connector available by name. Called from each connector package's init.
func Register(name string, f Factory) { registry[name] = f }

// New builds a registered connector.
func New(name string, cfg Config) (Connector, error) {
	f, ok := registry[name]
	if !ok {
		return nil, fmt.Errorf("no connector %q", name)
	}
	return f(cfg)
}

// Names lists the registered connectors.
func Names() []string {
	var out []string
	for n := range registry {
		out = append(out, n)
	}
	sort.Strings(out)
	return out
}
