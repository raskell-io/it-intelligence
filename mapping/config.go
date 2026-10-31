package mapping

import (
	"os"
	"regexp"

	yaml "go.yaml.in/yaml/v3"
)

// Config holds the company-specific conventions the mappers rely on (config/mapping.yaml).
type Config struct {
	// ApplicationLabel is the workload label that carries the EA application id.
	ApplicationLabel string `yaml:"application_label"`
	// SourceRepositoryLabel is the workload label naming the repository the image is built from.
	SourceRepositoryLabel string `yaml:"source_repository_label"`
	// NamespaceOwnerLabel is the namespace label naming the owning team.
	NamespaceOwnerLabel string `yaml:"namespace_owner_label"`
	// EAIDPattern matches application ids of the EA repository; other label values are applications the
	// EA repository does not know.
	EAIDPattern string `yaml:"ea_id_pattern"`
	// SupportGroupSuffixes are stripped from ITSM support group names to find the team.
	SupportGroupSuffixes []string `yaml:"support_group_suffixes"`

	eaID *regexp.Regexp
}

// DefaultConfig matches the Kestrel Group seed.
func DefaultConfig() Config {
	return Config{
		ApplicationLabel:      "app.kestrel.example/application-id",
		SourceRepositoryLabel: "source.kestrel.example/repository",
		NamespaceOwnerLabel:   "team.kestrel.example/owner",
		EAIDPattern:           `^APP-\d{4}$`,
		SupportGroupSuffixes:  []string{" Support", " L2", " Operations"},
	}
}

// LoadConfig reads config/mapping.yaml over the defaults.
func LoadConfig(path string) (Config, error) {
	c := DefaultConfig()
	if path != "" {
		b, err := os.ReadFile(path)
		if err != nil && !os.IsNotExist(err) {
			return c, err
		}
		if err == nil {
			if err := yaml.Unmarshal(b, &c); err != nil {
				return c, err
			}
		}
	}
	return c, nil
}

// IsEAID reports whether a value looks like an EA application id.
func (c *Config) IsEAID(s string) bool {
	if c.eaID == nil {
		c.eaID = regexp.MustCompile(c.EAIDPattern)
	}
	return c.eaID.MatchString(s)
}
