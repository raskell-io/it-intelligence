# For agents: the MCP server

A coding agent starts every session knowing nothing about the company it works for. Given an MCP
endpoint into the Context Model, it can ask who owns a service, what depends on it and which decision
already ruled out the approach it was about to take, before it writes a line of code, in a few tool calls
instead of twenty.

## Connecting

itx serves MCP over streamable HTTP at `/mcp`. This repository carries a project-scoped configuration,
so Claude Code picks it up when you open the repository with itx running:

```json
{
  "mcpServers": {
    "it-intelligence": { "type": "http", "url": "http://localhost:8080/mcp" }
  }
}
```

In a company, the same entry points at the shared instance and goes into the template every new
repository starts from.

## Tools

| Tool | Input | Returns |
|---|---|---|
| `search_entities` | text, optional kind | ids, kinds, names and one-line summaries |
| `get_entity` | id | attributes, standing and direct relationships (at most 40, with a count) |
| `find_owner` | id or free-text name | the owning team, basis, confidence, edge id and the path that established it; candidates if the name is ambiguous |
| `impact_of_change` | id, depth (default 4) | every application that depends on it, grouped by distance |
| `explain_fact` | edge id | the observations, the rule and chain, or the model's judgement behind one relationship, and who confirmed it |
| `list_decisions` | id | architecture decisions that constrain the entity, newest first |
| `render_diagram` | id | a Mermaid diagram of the neighbourhood; inferred links dashed with their confidence |

Two resources: `ctx://ontology` (kinds and relations) and `ctx://scorecards` (how far each source can be
trusted).

## Three rules

- **Entity-centric.** "Give me this thing and its neighbourhood" works better for an agent than a query
  language.
- **Provenance in every answer.** Each relationship carries its basis, confidence, source and edge id, so
  the agent can cite where a fact came from and a person can check it with `explain_fact`.
- **Small answers.** Ids to follow up on, not subgraphs. `api/parity_test.go` fails when an answer grows
  past 6 KB; the point is to save the agent tokens, not to flood its context.

## One implementation

Every tool is a thin adapter over `api/query`, the same functions the REST API calls. An agent and a
person looking at the same entity see the same facts; the parity test checks it for every tool that has a
REST twin.

## An example

```text
> who owns claims-prod and what depends on it?

find_owner {"id": "claims-prod"}
  owner: ctx:Team/claims-platform, derived, 0.92
  path:  ctx:Namespace/claims-prod <- runs_in - ctx:Workload/claims-prod/claims-settlement (asserted, cluster)
         ctx:Workload/claims-prod/claims-settlement <- deployed_as - ctx:Application/claims-settlement (asserted, cluster)
         ctx:Application/claims-settlement - owned_by -> ctx:Team/claims-platform (asserted, itsm)

get_entity {"id": "ctx:Namespace/claims-prod"}
  runs_in from: claims-intake, claims-settlement, policy-admin, policy-admin-ui (workloads)

impact_of_change {"id": "ctx:Application/policy-admin"}
  distance 1: broker-portal, claims-intake, claims-settlement, customer-portal, policy-batch
  distance 2: kestrel-mobile
```

The answer names the team, says the ownership was derived rather than labelled, and shows the three facts
it rests on; then it follows the applications in the namespace to what depends on them. A useful
follow-up for the team: put an owner label on the namespace, and the derivation becomes an assertion.
