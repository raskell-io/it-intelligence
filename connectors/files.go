package connectors

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"iter"
	"os"
	"path/filepath"
	"sort"
	"time"

	yaml "go.yaml.in/yaml/v3"
)

// Helpers for file mode: every seed export is a CSV with a header, a JSON array, or a YAML list.
// Each record becomes a JSON object, so mappers only ever see JSON.

// ReadCSV returns each row as a JSON object keyed by the header.
func ReadCSV(path string) ([]json.RawMessage, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	cr := csv.NewReader(f)
	header, err := cr.Read()
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	var out []json.RawMessage
	for {
		row, err := cr.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("%s: %w", path, err)
		}
		m := map[string]string{}
		for i, h := range header {
			if i < len(row) {
				m[h] = row[i]
			}
		}
		b, _ := json.Marshal(m)
		out = append(out, b)
	}
	return out, nil
}

// ReadJSONArray returns the elements of a JSON array file.
func ReadJSONArray(path string) ([]json.RawMessage, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var out []json.RawMessage
	if err := json.Unmarshal(b, &out); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return out, nil
}

// ReadYAMLList returns the elements under key (for example "items" or "routes") as JSON objects.
func ReadYAMLList(path, key string) ([]json.RawMessage, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var doc map[string]any
	if err := yaml.Unmarshal(b, &doc); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	items, _ := doc[key].([]any)
	var out []json.RawMessage
	for _, it := range items {
		j, err := json.Marshal(it)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", path, err)
		}
		out = append(out, j)
	}
	return out, nil
}

// SubDirs lists the directories inside dir, sorted (cluster exports have one per cluster).
func SubDirs(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, e := range entries {
		if e.IsDir() {
			out = append(out, filepath.Join(dir, e.Name()))
		}
	}
	sort.Strings(out)
	return out, nil
}

// Field reads a string field from a JSON object, for building external ids.
func Field(raw json.RawMessage, path ...string) string {
	var v any
	if json.Unmarshal(raw, &v) != nil {
		return ""
	}
	for _, p := range path {
		m, ok := v.(map[string]any)
		if !ok {
			return ""
		}
		v = m[p]
	}
	s, _ := v.(string)
	return s
}

// FileDataset describes one data set of a file-mode connector.
type FileDataset struct {
	Name string
	Read func(dir string) ([]json.RawMessage, error)
	ID   func(rec json.RawMessage) string
}

// FileConnector is the shared implementation of file mode: read each export, emit one observation
// per record. Live modes are separate implementations of Connector.
type FileConnector struct {
	SourceName string
	Cfg        Config
	Sets       []FileDataset
}

func (c *FileConnector) Name() string { return c.SourceName }

func (c *FileConnector) Datasets() []string {
	var out []string
	for _, s := range c.Sets {
		out = append(out, s.Name)
	}
	return out
}

func (c *FileConnector) Extract(_ context.Context, dataset string, _ time.Time) iter.Seq2[Observation, error] {
	return func(yield func(Observation, error) bool) {
		for _, s := range c.Sets {
			if s.Name != dataset {
				continue
			}
			recs, err := s.Read(c.Cfg.Dir)
			if err != nil {
				yield(Observation{}, err)
				return
			}
			for _, rec := range recs {
				id := s.ID(rec)
				if id == "" {
					if !yield(Observation{}, fmt.Errorf("%s/%s: record without id: %s", c.SourceName, dataset, rec)) {
						return
					}
					continue
				}
				o := Observation{Source: c.SourceName, Dataset: dataset, ExternalID: id, Payload: rec, ObservedAt: c.Cfg.Now}
				if !yield(o, nil) {
					return
				}
			}
			return
		}
		yield(Observation{}, fmt.Errorf("%s: no data set %q", c.SourceName, dataset))
	}
}
