// Package config loads config/*.yaml.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	yaml "go.yaml.in/yaml/v3"

	"github.com/raskell-io/it-intelligence/mapping"
	"github.com/raskell-io/it-intelligence/resolver"
	"github.com/raskell-io/it-intelligence/resolver/provider"
)

// Source is one entry of config/sources.yaml.
type Source struct {
	Mode  string            `yaml:"mode"`
	Dir   string            `yaml:"dir"`
	Every Duration          `yaml:"every"`
	Cycle Duration          `yaml:"cycle"`
	Opts  map[string]string `yaml:"opts"`
}

// Config is the whole configuration.
type Config struct {
	Root          string
	ReferenceTime *time.Time         `yaml:"reference_time"`
	SourceOrder   []string           `yaml:"source_order"`
	Weights       map[string]float64 `yaml:"weights"`
	Sources       map[string]Source  `yaml:"sources"`

	Mapping  mapping.Config
	Resolver resolver.Config
	Provider provider.Config
}

// Now is the reference time when one is configured (the seed), the wall clock otherwise.
func (c *Config) Now() time.Time {
	if c.ReferenceTime != nil {
		return *c.ReferenceTime
	}
	return time.Now().UTC()
}

// SourceNames lists the configured sources in source order.
func (c *Config) SourceNames() []string {
	var out []string
	seen := map[string]bool{}
	// earepo first: it defines applications, capabilities and market units.
	order := append([]string{"earepo"}, c.SourceOrder...)
	for _, s := range order {
		if _, ok := c.Sources[s]; ok && !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	var rest []string
	for s := range c.Sources {
		if !seen[s] {
			rest = append(rest, s)
		}
	}
	sort.Strings(rest)
	return append(out, rest...)
}

// Path resolves a path relative to the repository root.
func (c *Config) Path(p string) string {
	if filepath.IsAbs(p) {
		return p
	}
	return filepath.Join(c.Root, p)
}

// Load reads config/ under root.
func Load(root string) (*Config, error) {
	c := &Config{Root: root}
	b, err := os.ReadFile(filepath.Join(root, "config", "sources.yaml"))
	if err != nil {
		return nil, err
	}
	if err := yaml.Unmarshal(b, c); err != nil {
		return nil, fmt.Errorf("sources.yaml: %w", err)
	}
	if c.Mapping, err = mapping.LoadConfig(filepath.Join(root, "config", "mapping.yaml")); err != nil {
		return nil, fmt.Errorf("mapping.yaml: %w", err)
	}
	if c.Resolver, err = resolver.LoadConfig(filepath.Join(root, "config", "resolver.yaml")); err != nil {
		return nil, fmt.Errorf("resolver.yaml: %w", err)
	}
	var p struct {
		Provider provider.Config `yaml:"provider"`
	}
	b, _ = os.ReadFile(filepath.Join(root, "config", "resolver.yaml"))
	if err := yaml.Unmarshal(b, &p); err != nil {
		return nil, err
	}
	c.Provider = p.Provider
	if c.Provider.Fixtures == "" {
		c.Provider.Fixtures = filepath.Join(root, "seed", "judgements.jsonl")
	}
	if len(c.Weights) == 0 {
		c.Weights = map[string]float64{"freshness": 0.3, "completeness": 0.2, "agreement": 0.25, "resolvability": 0.25}
	}
	return c, nil
}

// Duration accepts Go durations plus a "d" suffix for days ("180d").
type Duration time.Duration

func (d *Duration) UnmarshalYAML(n *yaml.Node) error {
	s := strings.TrimSpace(n.Value)
	if strings.HasSuffix(s, "d") {
		days, err := strconv.Atoi(strings.TrimSuffix(s, "d"))
		if err != nil {
			return fmt.Errorf("duration %q: %w", s, err)
		}
		*d = Duration(time.Duration(days) * 24 * time.Hour)
		return nil
	}
	v, err := time.ParseDuration(s)
	if err != nil {
		return fmt.Errorf("duration %q: %w", s, err)
	}
	*d = Duration(v)
	return nil
}
