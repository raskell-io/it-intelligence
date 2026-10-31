// Package githost reads a git host: repositories (with commit statistics and catalog-info keys),
// teams, and architecture decision records found in repositories.
//
// File mode reads the seed. Live mode (live.go) reads one organisation from the GitHub REST API.
package githost

import (
	"encoding/json"
	"fmt"
	"path/filepath"

	c "github.com/raskell-io/it-intelligence/connectors"
)

func init() { c.Register("githost", New) }

func jsonFile(name string) func(string) ([]json.RawMessage, error) {
	return func(dir string) ([]json.RawMessage, error) { return c.ReadJSONArray(filepath.Join(dir, name)) }
}

func New(cfg c.Config) (c.Connector, error) {
	if cfg.Mode == "live" {
		return newLive(cfg)
	}
	return &c.FileConnector{SourceName: "githost", Cfg: cfg, Sets: []c.FileDataset{
		{Name: "teams", Read: jsonFile("teams.json"), ID: func(r json.RawMessage) string { return c.Field(r, "slug") }},
		{Name: "repositories", Read: jsonFile("repositories.json"), ID: func(r json.RawMessage) string { return c.Field(r, "full_name") }},
		{Name: "decisions", Read: jsonFile("decisions.json"), ID: func(r json.RawMessage) string {
			var d struct {
				Repository string `json:"repository"`
				Number     int    `json:"number"`
			}
			if json.Unmarshal(r, &d) != nil || d.Repository == "" {
				return ""
			}
			return fmt.Sprintf("%s/%d", d.Repository, d.Number)
		}},
	}}, nil
}
