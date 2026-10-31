// Package rest is the REST API, for the web UI and for other systems. Like the MCP server it is a thin
// adapter over api/query. Entity ids contain ":" and "/", so they travel as the id query parameter.
// The contract is api/openapi.yaml.
package rest

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"strconv"
	"time"

	yaml "go.yaml.in/yaml/v3"

	"github.com/raskell-io/it-intelligence/api/query"
	"github.com/raskell-io/it-intelligence/artifacts"
	"github.com/raskell-io/it-intelligence/ontology/generated/go/vocab"
	"github.com/raskell-io/it-intelligence/workflows"
)

// Enqueuer starts workflow runs.
type Enqueuer interface {
	Enqueue(ctx context.Context, workflow string, input any, actor string) (int64, error)
}

// Person is someone the "act as" picker offers (config/people.yaml).
type Person struct {
	ID   string `json:"id" yaml:"id"`
	Name string `json:"name" yaml:"name"`
	Team string `json:"team" yaml:"team"`
}

// LoadPeople reads config/people.yaml.
func LoadPeople(path string) ([]Person, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var doc struct {
		People []Person `yaml:"people"`
	}
	err = yaml.Unmarshal(b, &doc)
	return doc.People, err
}

// API holds what the handlers need.
type API struct {
	Q      *query.Q
	Runs   Enqueuer
	People []Person
	Now    func() time.Time // the reference time, so the UI can tell what is stale
}

// Routes registers every endpoint on mux.
func (a *API) Routes(mux *http.ServeMux) {
	get := func(p string, h func(*http.Request) (any, error)) { mux.HandleFunc("GET /api/v1"+p, wrap(h)) }
	post := func(p string, h func(*http.Request) (any, error)) { mux.HandleFunc("POST /api/v1"+p, wrap(h)) }

	get("/stats", func(r *http.Request) (any, error) {
		st, err := a.Q.Stats(r.Context())
		if err == nil && a.Now != nil {
			st["now"] = a.Now()
		}
		return st, err
	})
	get("/ontology", func(r *http.Request) (any, error) {
		return map[string]any{"version": vocab.Version, "kinds": vocab.Kinds, "relations": vocab.Relations, "enums": vocab.Enums}, nil
	})
	get("/people", func(r *http.Request) (any, error) { return map[string]any{"people": a.People}, nil })
	get("/entities", func(r *http.Request) (any, error) {
		hits, err := a.Q.Search(r.Context(), r.URL.Query().Get("q"), r.URL.Query().Get("kind"), atoi(r.URL.Query().Get("limit"), 20))
		return map[string]any{"entities": hits}, err
	})
	get("/entity", func(r *http.Request) (any, error) { return a.Q.GetEntity(r.Context(), r.URL.Query().Get("id")) })
	get("/owner", func(r *http.Request) (any, error) { return a.Q.FindOwner(r.Context(), r.URL.Query().Get("id")) })
	get("/impact", func(r *http.Request) (any, error) {
		return a.Q.ImpactOfChange(r.Context(), r.URL.Query().Get("id"), atoi(r.URL.Query().Get("depth"), 4))
	})
	get("/decisions", func(r *http.Request) (any, error) {
		id := r.URL.Query().Get("id")
		ds, err := a.Q.ListDecisions(r.Context(), id)
		return map[string]any{"entity": id, "decisions": ds}, err
	})
	get("/diagram", func(r *http.Request) (any, error) {
		id := r.URL.Query().Get("id")
		m, err := artifacts.Mermaid(r.Context(), a.Q, id)
		return map[string]any{"entity": id, "format": "mermaid", "mermaid": m}, err
	})
	get("/edges/{edge}/explain", func(r *http.Request) (any, error) { return a.Q.ExplainFact(r.Context(), r.PathValue("edge")) })
	get("/review", func(r *http.Request) (any, error) {
		items, err := a.Q.Review(r.Context(), r.URL.Query().Get("team"))
		return map[string]any{"items": items}, err
	})
	get("/scorecards", func(r *http.Request) (any, error) {
		cards, err := a.Q.Scorecards(r.Context(), r.URL.Query().Get("failing") == "true")
		return map[string]any{"scorecards": cards}, err
	})
	get("/datasets", func(r *http.Request) (any, error) {
		ds, err := a.Q.Datasets(r.Context())
		return map[string]any{"datasets": ds}, err
	})
	get("/artifacts", func(r *http.Request) (any, error) {
		q := r.URL.Query()
		as, err := a.Q.Artifacts(r.Context(), q.Get("kind"), q.Get("subject"), q.Get("body") == "true")
		return map[string]any{"artifacts": as}, err
	})
	get("/findings", func(r *http.Request) (any, error) {
		fs, err := a.Q.Findings(r.Context())
		return map[string]any{"findings": fs}, err
	})
	get("/runs", func(r *http.Request) (any, error) {
		runs, err := a.Q.Runs(r.Context(), 0, atoi(r.URL.Query().Get("limit"), 30))
		return map[string]any{"runs": runs}, err
	})
	get("/runs/{id}", func(r *http.Request) (any, error) {
		runs, err := a.Q.Runs(r.Context(), int64(atoi(r.PathValue("id"), 0)), 1)
		if err == nil && len(runs) == 0 {
			err = query.ErrNotFound
		}
		if err != nil {
			return nil, err
		}
		return runs[0], nil
	})

	// Writes start workflows and need a name: confirmations carry a person's name.
	post("/runs", func(r *http.Request) (any, error) {
		actor, err := a.actor(r)
		if err != nil {
			return nil, err
		}
		var body struct {
			Workflow string          `json:"workflow"`
			Input    json.RawMessage `json:"input"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			return nil, badRequest(err.Error())
		}
		if body.Workflow != "IngestSource" && body.Workflow != "IngestAll" && body.Workflow != "BuildArtifacts" {
			return nil, badRequest("only IngestSource, IngestAll and BuildArtifacts start here")
		}
		var in any = body.Input
		if len(body.Input) == 0 {
			in = map[string]any{}
		}
		id, err := a.Runs.Enqueue(r.Context(), body.Workflow, in, actor)
		return map[string]any{"run": id}, err
	})
	for _, target := range []string{"edges", "aliases"} {
		t := target
		for _, action := range []string{"confirm", "reject"} {
			act := action
			post("/"+t+"/{id}/"+act, func(r *http.Request) (any, error) {
				actor, err := a.actor(r)
				if err != nil {
					return nil, err
				}
				var body struct {
					Note string `json:"note"`
				}
				json.NewDecoder(r.Body).Decode(&body)
				typ := map[string]string{"edges": "edge", "aliases": "alias"}[t]
				id, err := a.Runs.Enqueue(r.Context(), "ConfirmLink", workflows.LinkInput{TargetType: typ, TargetID: r.PathValue("id"), Action: act, Note: body.Note}, actor)
				return map[string]any{"run": id}, err
			})
		}
	}
	post("/datasets/{source}/{dataset}/primary", func(r *http.Request) (any, error) {
		actor, err := a.actor(r)
		if err != nil {
			return nil, err
		}
		var body struct {
			Owner     string `json:"owner"`
			Requester string `json:"requester"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			return nil, badRequest(err.Error())
		}
		id, err := a.Runs.Enqueue(r.Context(), "DeclarePrimary", workflows.PrimaryInput{Source: r.PathValue("source"), Dataset: r.PathValue("dataset"),
			Owner: body.Owner, Requester: body.Requester, Approver: actor}, actor)
		return map[string]any{"run": id}, err
	})
}

func (a *API) actor(r *http.Request) (string, error) {
	id := r.Header.Get("X-Actor")
	if id == "" {
		return "", &httpError{http.StatusUnauthorized, "writes need an X-Actor header naming a person from config/people.yaml (the reference has no authentication)"}
	}
	for _, p := range a.People {
		if p.ID == id {
			return id, nil
		}
	}
	return "", &httpError{http.StatusForbidden, "unknown actor " + id}
}

type httpError struct {
	code int
	msg  string
}

func (e *httpError) Error() string { return e.msg }

func badRequest(msg string) error { return &httpError{http.StatusBadRequest, msg} }

func wrap(h func(*http.Request) (any, error)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		v, err := h(r)
		w.Header().Set("Content-Type", "application/json")
		if err != nil {
			code := http.StatusInternalServerError
			var he *httpError
			switch {
			case errors.As(err, &he):
				code = he.code
			case errors.Is(err, query.ErrNotFound):
				code = http.StatusNotFound
			}
			w.WriteHeader(code)
			json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}
		enc := json.NewEncoder(w)
		enc.SetEscapeHTML(false)
		enc.SetIndent("", "  ")
		enc.Encode(v)
	}
}

func atoi(s string, def int) int {
	if n, err := strconv.Atoi(s); err == nil {
		return n
	}
	return def
}
