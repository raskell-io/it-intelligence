// Package gateway reads the route configuration of an API gateway.
package gateway

import (
	"encoding/json"
	"path/filepath"

	c "github.com/raskell-io/it-intelligence/connectors"
)

func init() { c.Register("gateway", New) }

func New(cfg c.Config) (c.Connector, error) {
	return &c.FileConnector{SourceName: "gateway", Cfg: cfg, Sets: []c.FileDataset{
		{Name: "routes", Read: func(dir string) ([]json.RawMessage, error) {
			return c.ReadYAMLList(filepath.Join(dir, "routes.yaml"), "routes")
		}, ID: func(r json.RawMessage) string { return c.Field(r, "id") }},
	}}, nil
}
