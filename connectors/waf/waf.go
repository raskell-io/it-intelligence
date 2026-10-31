// Package waf reads the policies of a web application firewall. It is the smallest connector and
// the example in docs/adding-a-source.md.
package waf

import (
	"encoding/json"
	"path/filepath"

	c "github.com/raskell-io/it-intelligence/connectors"
)

func init() { c.Register("waf", New) }

func New(cfg c.Config) (c.Connector, error) {
	return &c.FileConnector{SourceName: "waf", Cfg: cfg, Sets: []c.FileDataset{
		{Name: "policies", Read: func(dir string) ([]json.RawMessage, error) {
			return c.ReadJSONArray(filepath.Join(dir, "policies.json"))
		}, ID: func(r json.RawMessage) string { return c.Field(r, "policy_id") }},
	}}, nil
}
