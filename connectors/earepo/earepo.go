// Package earepo reads exports of an enterprise architecture repository: the application portfolio,
// the capability map, market units, teams and interfaces. The reference reads CSV exports only.
package earepo

import (
	"encoding/json"
	"path/filepath"

	c "github.com/raskell-io/it-intelligence/connectors"
)

func init() { c.Register("earepo", New) }

func csvFile(name string) func(string) ([]json.RawMessage, error) {
	return func(dir string) ([]json.RawMessage, error) { return c.ReadCSV(filepath.Join(dir, name)) }
}

func New(cfg c.Config) (c.Connector, error) {
	return &c.FileConnector{SourceName: "earepo", Cfg: cfg, Sets: []c.FileDataset{
		{Name: "market_units", Read: csvFile("market_units.csv"), ID: func(r json.RawMessage) string { return c.Field(r, "mu_id") }},
		{Name: "teams", Read: csvFile("teams.csv"), ID: func(r json.RawMessage) string { return c.Field(r, "team_id") }},
		{Name: "capabilities", Read: csvFile("capabilities.csv"), ID: func(r json.RawMessage) string { return c.Field(r, "cap_id") }},
		{Name: "applications", Read: csvFile("applications.csv"), ID: func(r json.RawMessage) string { return c.Field(r, "app_id") }},
		{Name: "interfaces", Read: csvFile("interfaces.csv"), ID: func(r json.RawMessage) string {
			return c.Field(r, "from_app") + ">" + c.Field(r, "to_app")
		}},
	}}, nil
}
