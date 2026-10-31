// Package api_test checks that the REST API and the MCP server give the same answers: an agent and a
// person looking at the same entity must see the same facts.
package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/raskell-io/it-intelligence/api/mcp"
	"github.com/raskell-io/it-intelligence/api/query"
	"github.com/raskell-io/it-intelligence/api/rest"
	"github.com/raskell-io/it-intelligence/store"
)

// Needs a database with the seed ingested: ITX_TEST_DATABASE_URL after `make dev-ingest`.
func TestRESTAndMCPAgree(t *testing.T) {
	dsn := os.Getenv("ITX_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("ITX_TEST_DATABASE_URL not set")
	}
	ctx := context.Background()
	s, err := store.Open(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	var n int
	s.Pool.QueryRow(ctx, `select count(*) from entity`).Scan(&n)
	if n == 0 {
		t.Skip("database has no model; run make dev-ingest first")
	}
	q := &query.Q{Pool: s.Pool}
	mux := http.NewServeMux()
	(&rest.API{Q: q}).Routes(mux)
	mux.Handle("/mcp", mcp.Handler(mcp.Server(q, "test")))
	srv := httptest.NewServer(mux)
	defer srv.Close()

	restGet := func(path string) any {
		res, err := http.Get(srv.URL + "/api/v1" + path)
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		var v any
		json.NewDecoder(res.Body).Decode(&v)
		return v
	}
	mcpCall := func(tool string, args map[string]any) (any, int) {
		body, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tools/call", "params": map[string]any{"name": tool, "arguments": args}})
		req, _ := http.NewRequest(http.MethodPost, srv.URL+"/mcp", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json, text/event-stream")
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		raw, _ := io.ReadAll(res.Body)
		text := string(raw)
		if i := strings.Index(text, "data: "); i >= 0 {
			text = strings.TrimSpace(text[i+6:])
		}
		var msg struct {
			Result struct {
				Structured any `json:"structuredContent"`
				Content    []struct {
					Text string `json:"text"`
				} `json:"content"`
			} `json:"result"`
		}
		if err := json.Unmarshal([]byte(text), &msg); err != nil {
			t.Fatalf("%s: %v: %s", tool, err, raw)
		}
		size := 0
		for _, c := range msg.Result.Content {
			size += len(c.Text)
		}
		return msg.Result.Structured, size
	}

	esc := url.QueryEscape
	cases := []struct {
		tool string
		args map[string]any
		rest string
	}{
		{"find_owner", map[string]any{"id": "ctx:Namespace/claims-prod"}, "/owner?id=" + esc("ctx:Namespace/claims-prod")},
		{"find_owner", map[string]any{"id": "claims-prod"}, "/owner?id=claims-prod"},
		{"impact_of_change", map[string]any{"id": "ctx:Application/payment-gateway", "depth": 4}, "/impact?id=" + esc("ctx:Application/payment-gateway") + "&depth=4"},
		{"list_decisions", map[string]any{"id": "ctx:Application/claims-intake"}, "/decisions?id=" + esc("ctx:Application/claims-intake")},
		{"render_diagram", map[string]any{"id": "ctx:Application/policy-admin"}, "/diagram?id=" + esc("ctx:Application/policy-admin")},
	}
	for _, c := range cases {
		got, size := mcpCall(c.tool, c.args)
		want := restGet(c.rest)
		if c.tool == "render_diagram" { // REST omits the view and format fields' defaults differently; compare the text
			got, want = got.(map[string]any)["mermaid"], want.(map[string]any)["mermaid"]
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("%s %v: MCP and REST differ\nmcp:  %v\nrest: %v", c.tool, c.args, got, want)
		}
		// Answers are small: the point is to save the agent tokens.
		if size > 6000 {
			t.Errorf("%s answer is %d bytes; keep tool answers small", c.tool, size)
		}
	}
}
