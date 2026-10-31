package githost

import (
	"context"
	"encoding/json"
	"fmt"
	"iter"
	"net/http"
	"os"
	"strings"
	"time"

	c "github.com/raskell-io/it-intelligence/connectors"
)

// live reads one GitHub organisation through the REST API, read-only. It exists to show that the
// Connector interface is real; it is not a production connector (no rate-limit handling beyond
// stopping, no commit statistics, no decisions). Options: org (required), token from GITHUB_TOKEN.
type live struct {
	cfg   c.Config
	org   string
	token string
	http  *http.Client
}

func newLive(cfg c.Config) (c.Connector, error) {
	org := cfg.Opts["org"]
	if org == "" {
		return nil, fmt.Errorf("githost live mode needs opts.org")
	}
	return &live{cfg: cfg, org: org, token: os.Getenv("GITHUB_TOKEN"), http: &http.Client{Timeout: 30 * time.Second}}, nil
}

func (l *live) Name() string       { return "githost" }
func (l *live) Datasets() []string { return []string{"teams", "repositories"} }

func (l *live) get(ctx context.Context, url string, out any) (next string, err error) {
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	req.Header.Set("Accept", "application/vnd.github+json")
	if l.token != "" {
		req.Header.Set("Authorization", "Bearer "+l.token)
	}
	resp, err := l.http.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GET %s: %s", url, resp.Status)
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return "", err
	}
	return nextLink(resp.Header.Get("Link")), nil
}

func (l *live) Extract(ctx context.Context, dataset string, _ time.Time) iter.Seq2[c.Observation, error] {
	return func(yield func(c.Observation, error) bool) {
		var url string
		switch dataset {
		case "repositories":
			url = fmt.Sprintf("https://api.github.com/orgs/%s/repos?per_page=100", l.org)
		case "teams":
			url = fmt.Sprintf("https://api.github.com/orgs/%s/teams?per_page=100", l.org)
		default:
			yield(c.Observation{}, fmt.Errorf("githost live: no data set %q", dataset))
			return
		}
		for url != "" {
			var page []map[string]any
			next, err := l.get(ctx, url, &page)
			if err != nil {
				yield(c.Observation{}, err)
				return
			}
			for _, item := range page {
				rec := item
				if dataset == "repositories" { // shape it like the seed export
					rec = map[string]any{"full_name": item["full_name"], "name": item["name"],
						"description": item["description"], "readme_excerpt": item["description"],
						"archived": item["archived"], "pushed_at": item["pushed_at"]}
				}
				b, _ := json.Marshal(rec)
				id := c.Field(b, "full_name")
				if dataset == "teams" {
					id = c.Field(b, "slug")
				}
				if !yield(c.Observation{Source: "githost", Dataset: dataset, ExternalID: id, Payload: b, ObservedAt: time.Now().UTC()}, nil) {
					return
				}
			}
			url = next
		}
	}
}

// nextLink extracts rel="next" from a GitHub Link header.
func nextLink(h string) string {
	for _, part := range strings.Split(h, ",") {
		if !strings.Contains(part, `rel="next"`) {
			continue
		}
		start, end := strings.Index(part, "<"), strings.Index(part, ">")
		if start >= 0 && end > start {
			return part[start+1 : end]
		}
	}
	return ""
}
