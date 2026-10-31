// Package mcp is the Model Context Protocol server: the interface for coding agents.
//
// Three rules shape the tools. They are entity-centric: "give me this thing and its neighbourhood"
// works better for an agent than a query language. Every answer carries its provenance, so the agent can
// cite where a fact came from and a person can check it. Answers are small, with ids to follow up on,
// because the point of the platform is to save the agent tokens, not to flood its context window.
//
// Every tool is a thin adapter over api/query, the same functions the REST API calls.
package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/raskell-io/it-intelligence/api/query"
	"github.com/raskell-io/it-intelligence/artifacts"
	"github.com/raskell-io/it-intelligence/ontology/generated/go/vocab"
)

// MaxEdges bounds how many edges get_entity returns; the rest are counted.
const MaxEdges = 40

type SearchIn struct {
	Text string `json:"text" jsonschema:"words to look for in names, ids and descriptions"`
	Kind string `json:"kind,omitempty" jsonschema:"optional entity kind: Application, Repository, Workload, Namespace, Team, Capability, Endpoint, Decision or MarketUnit"`
}
type SearchOut struct {
	Entities []query.EntitySummary `json:"entities"`
}

type IDIn struct {
	ID string `json:"id" jsonschema:"a Context Model id such as ctx:Application/policy-admin"`
}

type EntityOut struct {
	query.Entity
	EdgesTotal int    `json:"edges_total"`
	Note       string `json:"note,omitempty"`
}

type OwnerIn struct {
	ID string `json:"id" jsonschema:"a Context Model id, or a free-text name such as claims-prod"`
}

type ImpactIn struct {
	ID    string `json:"id" jsonschema:"the application that would change, as a Context Model id"`
	Depth int    `json:"depth,omitempty" jsonschema:"how many depends_on hops to follow, 1 to 6, default 4"`
}

type ExplainIn struct {
	Edge string `json:"edge" jsonschema:"an edge id (e_...) from any other answer"`
}

type DecisionsOut struct {
	Entity    string              `json:"entity"`
	Decisions []query.DecisionRef `json:"decisions"`
}

type DiagramIn struct {
	ID   string `json:"id" jsonschema:"the entity to draw"`
	View string `json:"view,omitempty" jsonschema:"only 'neighbourhood' in the reference"`
}
type DiagramOut struct {
	Entity  string `json:"entity"`
	Format  string `json:"format"`
	Mermaid string `json:"mermaid"`
}

// Server builds the MCP server over a query layer.
func Server(q *query.Q, version string) *sdk.Server {
	s := sdk.NewServer(&sdk.Implementation{Name: "it-intelligence", Title: "IT intelligence: the Context Model of the Kestrel Group", Version: version},
		&sdk.ServerOptions{Instructions: "Ask this server what exists in the company's IT, who owns it, how it connects and which " +
			"architecture decisions apply, before you write code against it. Start with search_entities or find_owner; follow ids. " +
			"Every relationship has a basis (asserted, derived, inferred) and a confidence; call explain_fact with its edge id to see why."})

	sdk.AddTool(s, &sdk.Tool{Name: "search_entities", Description: "Find entities (applications, repositories, workloads, namespaces, teams, " +
		"capabilities, endpoints, decisions, market units) by name. Returns ids and one-line summaries; call get_entity on an id for details."},
		func(ctx context.Context, _ *sdk.CallToolRequest, in SearchIn) (*sdk.CallToolResult, SearchOut, error) {
			hits, err := q.Search(ctx, in.Text, in.Kind, 15)
			return nil, SearchOut{Entities: hits}, err
		})

	sdk.AddTool(s, &sdk.Tool{Name: "get_entity", Description: "Get one entity: its attributes, its standing (secondary: mirrored from a " +
		"source; primary: the Context Model is authoritative) and its direct relationships with basis, confidence, source and edge id."},
		func(ctx context.Context, _ *sdk.CallToolRequest, in IDIn) (*sdk.CallToolResult, EntityOut, error) {
			e, err := q.GetEntity(ctx, in.ID)
			if err != nil {
				return nil, EntityOut{}, err
			}
			out := EntityOut{Entity: *e, EdgesTotal: len(e.Edges)}
			if len(out.Edges) > MaxEdges {
				out.Edges = out.Edges[:MaxEdges]
				out.Note = fmt.Sprintf("showing %d of %d relationships", MaxEdges, out.EdgesTotal)
			}
			return nil, out, nil
		})

	sdk.AddTool(s, &sdk.Tool{Name: "find_owner", Description: "Who owns this? Returns the accountable team, how the ownership was " +
		"established (the path through the graph), its basis and confidence. Accepts an id or a free-text name; an ambiguous name returns candidates instead of a guess."},
		func(ctx context.Context, _ *sdk.CallToolRequest, in OwnerIn) (*sdk.CallToolResult, query.Owner, error) {
			o, err := q.FindOwner(ctx, in.ID)
			if err != nil {
				return nil, query.Owner{}, err
			}
			return nil, *o, nil
		})

	sdk.AddTool(s, &sdk.Tool{Name: "impact_of_change", Description: "What breaks if this application changes? Returns every application " +
		"that depends on it, directly or through others, grouped by distance. Use get_entity or find_owner on the results to know whom to tell."},
		func(ctx context.Context, _ *sdk.CallToolRequest, in ImpactIn) (*sdk.CallToolResult, query.Impact, error) {
			r, err := q.ImpactOfChange(ctx, in.ID, in.Depth)
			if err != nil {
				return nil, query.Impact{}, err
			}
			return nil, *r, nil
		})

	sdk.AddTool(s, &sdk.Tool{Name: "explain_fact", Description: "Why does the model believe this relationship? Returns the evidence " +
		"behind one edge: the source observations, the rule and chain for derived facts, the model's judgement and reasons for inferred ones, and who confirmed it."},
		func(ctx context.Context, _ *sdk.CallToolRequest, in ExplainIn) (*sdk.CallToolResult, query.Explanation, error) {
			x, err := q.ExplainFact(ctx, in.Edge)
			if err != nil {
				return nil, query.Explanation{}, err
			}
			return nil, *x, nil
		})

	sdk.AddTool(s, &sdk.Tool{Name: "list_decisions", Description: "Architecture decisions (ADRs) that constrain an entity, newest first, " +
		"with status and a link. Check these before proposing a design."},
		func(ctx context.Context, _ *sdk.CallToolRequest, in IDIn) (*sdk.CallToolResult, DecisionsOut, error) {
			ds, err := q.ListDecisions(ctx, in.ID)
			return nil, DecisionsOut{Entity: in.ID, Decisions: ds}, err
		})

	sdk.AddTool(s, &sdk.Tool{Name: "render_diagram", Description: "A Mermaid flowchart of an entity's neighbourhood: repositories, " +
		"workloads, endpoints, capabilities, dependencies, decisions. Inferred links are dashed and labelled with their confidence."},
		func(ctx context.Context, _ *sdk.CallToolRequest, in DiagramIn) (*sdk.CallToolResult, DiagramOut, error) {
			m, err := artifacts.Mermaid(ctx, q, in.ID)
			return nil, DiagramOut{Entity: in.ID, Format: "mermaid", Mermaid: m}, err
		})

	s.AddResource(&sdk.Resource{URI: "ctx://ontology", Name: "ontology", Title: "The Context Model: entity and relation kinds", MIMEType: "application/json"},
		func(ctx context.Context, req *sdk.ReadResourceRequest) (*sdk.ReadResourceResult, error) {
			b, _ := json.MarshalIndent(map[string]any{"version": vocab.Version, "kinds": vocab.Kinds, "relations": vocab.Relations}, "", "  ")
			return &sdk.ReadResourceResult{Contents: []*sdk.ResourceContents{{URI: req.Params.URI, MIMEType: "application/json", Text: string(b)}}}, nil
		})
	s.AddResource(&sdk.Resource{URI: "ctx://scorecards", Name: "scorecards", Title: "How much each source can be trusted", MIMEType: "application/json"},
		func(ctx context.Context, req *sdk.ReadResourceRequest) (*sdk.ReadResourceResult, error) {
			cards, err := q.Scorecards(ctx, false)
			if err != nil {
				return nil, err
			}
			b, _ := json.MarshalIndent(cards, "", "  ")
			return &sdk.ReadResourceResult{Contents: []*sdk.ResourceContents{{URI: req.Params.URI, MIMEType: "application/json", Text: string(b)}}}, nil
		})
	return s
}

// Handler serves MCP over streamable HTTP.
func Handler(s *sdk.Server) http.Handler {
	return sdk.NewStreamableHTTPHandler(func(*http.Request) *sdk.Server { return s }, &sdk.StreamableHTTPOptions{Stateless: true})
}
