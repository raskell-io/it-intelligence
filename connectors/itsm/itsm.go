// Package itsm reads exports of an IT service management tool: business services and incidents.
package itsm

import (
	"encoding/json"
	"path/filepath"

	c "github.com/raskell-io/it-intelligence/connectors"
)

func init() { c.Register("itsm", New) }

func jsonFile(name string) func(string) ([]json.RawMessage, error) {
	return func(dir string) ([]json.RawMessage, error) { return c.ReadJSONArray(filepath.Join(dir, name)) }
}

func New(cfg c.Config) (c.Connector, error) {
	return &c.FileConnector{SourceName: "itsm", Cfg: cfg, Sets: []c.FileDataset{
		{Name: "business_services", Read: jsonFile("business_services.json"), ID: func(r json.RawMessage) string { return c.Field(r, "sys_id") }},
		{Name: "incidents", Read: jsonFile("incidents.json"), ID: func(r json.RawMessage) string { return c.Field(r, "number") }},
	}}, nil
}
