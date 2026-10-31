package resolver

import (
	"os"

	yaml "go.yaml.in/yaml/v3"
)

// Config is config/resolver.yaml.
type Config struct {
	// Inferred links at or above Accept are active; between Review and Accept they wait for the owning
	// team; below Review they are dropped and logged.
	Accept float64 `yaml:"accept"`
	Review float64 `yaml:"review"`
	// Asserted facts start at AssertedBase + AssertedSpan * the source's last overall score, so a fact
	// from a well-kept source outranks the same claim from a neglected one.
	AssertedBase float64 `yaml:"asserted_base"`
	AssertedSpan float64 `yaml:"asserted_span"`
	// DefaultScore is used before a data set has its first scorecard.
	DefaultScore float64 `yaml:"default_score"`
	// Rules: name -> precision. A rule not listed here is off. Precisions are measured against the seed's
	// answer key by `itx verify --resolver`; update them when the measurement says so.
	Rules map[string]float64 `yaml:"rules"`
	// NameSuffixes are stripped before names are compared (pol-adm-svc -> pol-adm).
	NameSuffixes []string `yaml:"name_suffixes"`
	// ShortlistSize bounds what the model sees per candidate.
	ShortlistSize int `yaml:"shortlist_size"`
}

// DefaultConfig is the configuration the article describes.
func DefaultConfig() Config {
	return Config{
		Accept: 0.9, Review: 0.6,
		AssertedBase: 0.7, AssertedSpan: 0.3, DefaultScore: 0.9,
		Rules: map[string]float64{
			"realised_via_build":      0.95,
			"same_normalised_name":    0.9,
			"team_owns_single_app":    0.7,
			"owner_via_workloads":     0.95,
			"owner_via_namespace":     0.9,
			"app_owner_via_namespace": 0.75,
			"exposed_via_route":       0.95,
		},
		NameSuffixes:  []string{"-svc", "-service"},
		ShortlistSize: 5,
	}
}

// LoadConfig reads config/resolver.yaml over the defaults.
func LoadConfig(path string) (Config, error) {
	c := DefaultConfig()
	b, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return c, nil
	}
	if err != nil {
		return c, err
	}
	err = yaml.Unmarshal(b, &c)
	return c, err
}
