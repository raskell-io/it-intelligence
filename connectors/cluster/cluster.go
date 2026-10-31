// Package cluster reads container clusters: namespaces and workloads (deployments, stateful sets,
// cron jobs). File mode reads `kubectl get -o yaml` lists, one directory per cluster.
package cluster

import (
	"encoding/json"
	"path/filepath"

	c "github.com/raskell-io/it-intelligence/connectors"
)

func init() { c.Register("cluster", New) }

// readAll reads one file from every cluster directory and stamps each item with its cluster name.
func readAll(name string) func(string) ([]json.RawMessage, error) {
	return func(dir string) ([]json.RawMessage, error) {
		dirs, err := c.SubDirs(dir)
		if err != nil {
			return nil, err
		}
		var out []json.RawMessage
		for _, d := range dirs {
			items, err := c.ReadYAMLList(filepath.Join(d, name), "items")
			if err != nil {
				return nil, err
			}
			for _, it := range items {
				var m map[string]any
				if err := json.Unmarshal(it, &m); err != nil {
					return nil, err
				}
				m["cluster"] = filepath.Base(d)
				b, _ := json.Marshal(m)
				out = append(out, b)
			}
		}
		return out, nil
	}
}

func New(cfg c.Config) (c.Connector, error) {
	return &c.FileConnector{SourceName: "cluster", Cfg: cfg, Sets: []c.FileDataset{
		{Name: "namespaces", Read: readAll("namespaces.yaml"), ID: func(r json.RawMessage) string { return c.Field(r, "metadata", "name") }},
		{Name: "workloads", Read: readAll("workloads.yaml"), ID: func(r json.RawMessage) string {
			ns, n := c.Field(r, "metadata", "namespace"), c.Field(r, "metadata", "name")
			if ns == "" || n == "" {
				return ""
			}
			return ns + "/" + n
		}},
	}}, nil
}
