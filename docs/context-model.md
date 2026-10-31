# The Context Model

The Context Model is the ontology: the entity kinds, the relation kinds, and what each may connect. It
lives in `ontology/` as LinkML and is the only place those names are declared. Version 0.1.0 has nine
entity kinds and fifteen relation kinds, a schema one person can explain on a whiteboard.

## Entity kinds

| Kind | Meaning | Id pattern | Comes from |
|---|---|---|---|
| Application | a thing the business recognises as one system, however many parts it has | `ctx:Application/{short_name}` | EA repository, or a deployment label the EA repository does not know |
| Repository | a code repository | `ctx:Repository/{org}/{name}` | git host |
| Workload | a deployment, stateful set or cron job | `ctx:Workload/{namespace}/{name}` | clusters |
| Namespace | a cluster namespace | `ctx:Namespace/{name}` | clusters |
| Team | an accountable team | `ctx:Team/{slug}` | EA repository, git host |
| Capability | a business capability from the EA capability map | `ctx:Capability/{slug}` | EA repository |
| Endpoint | a reachable host and path prefix | `ctx:Endpoint/{host}{path}` | API gateway, WAF |
| Decision | an architecture decision record | `ctx:Decision/{repo}/{number}` | ADR files in repositories |
| MarketUnit | an organisational unit of the group | `ctx:MarketUnit/{slug}` | EA repository |

Some things are deliberately attributes, not kinds: incidents (`Application.incidents_90d`), the WAF
mode (`Endpoint.waf`), the cluster (`Namespace.cluster`). An ITSM business service is not a kind at all:
it resolves to an Application, as an alias. Each could become a kind when a question needs it.

## Relation kinds

```text
  Application ──realised_by──▶ Repository
              ──deployed_as──▶ Workload ──runs_in────▶ Namespace
                                        ──built_from─▶ Repository
              ──exposed_via──▶ Endpoint ──routes_to──▶ Workload
              ──supports─────▶ Capability ──refines──▶ Capability
              ──operated_by──▶ MarketUnit
              ──depends_on───▶ Application
              ──replaced_by──▶ Application
              ──owned_by─────▶ Team ──part_of──▶ MarketUnit

  Repository, Workload, Namespace, Endpoint ──owned_by──▶ Team
  Decision ──constrains──▶ any entity
  Decision ──supersedes──▶ Decision
```

| Relation | From | To | Cardinality |
|---|---|---|---|
| `realised_by` | Application | Repository | many |
| `deployed_as` | Application | Workload | many |
| `runs_in` | Workload | Namespace | one |
| `built_from` | Workload | Repository | one |
| `owned_by` | Application, Repository, Workload, Namespace, Endpoint | Team | one accountable |
| `part_of` | Team | MarketUnit | one |
| `operated_by` | Application | MarketUnit | many |
| `supports` | Application | Capability | many |
| `refines` | Capability | Capability | one parent |
| `exposed_via` | Application | Endpoint | many |
| `routes_to` | Endpoint | Workload | many |
| `depends_on` | Application | Application | many |
| `constrains` | Decision | any entity | many |
| `supersedes` | Decision | Decision | one |
| `replaced_by` | Application | Application | one |

"One" means at most one active edge should win. The store keeps every source's claim as its own row;
answers take the most confident, and conflicting claims show up as an insight instead of being hidden.

## How it is written

LinkML lets a model be written once and generated into the forms each part of the system needs.
Classes carry their attributes; relations are slots. LinkML allows one `domain` per slot, so the full
domain and range of a relation live in annotations:

```yaml
# ontology/relations.yaml
owned_by:
  description: The accountable team.
  range: Team
  annotations: {ctx_domain: "Application Repository Workload Namespace Endpoint", ctx_range: Team, ctx_cardinality: one}
```

## What is generated

`make ontology` writes everything below. CI fails when a committed file differs from what it produces.

| Output | By | Used by |
|---|---|---|
| `store/vocabulary.sql` | `ontology/gen/vocabulary.py` | the store: kind and relation tables, the domain and range trigger |
| `ontology/generated/go/vocab/vocab.go` | the same | every Go package: constants, kinds, relations, attributes |
| `web/lib/generated/vocabulary.ts` | the same | the web UI: `Kind` and `Relation` union types |
| `web/lib/generated/context_model.ts` | LinkML TypeScript | the web UI: enums such as `Basis` and `Standing` |
| `ontology/generated/jsonschema/` | LinkML JSON Schema | anyone validating records |
| `ontology/generated/owl/`, `shacl/` | LinkML OWL, SHACL | anyone reasoning over the model with semantic web tools |
| `ontology/generated/docs/` | LinkML docs | people (generated locally, not committed) |

Remove a relation from the ontology and the database refuses its edges, the Go build fails wherever the
constant is used, and the frontend stops compiling where it names it. That is the right kind of friction.

## Changing it

Add the kind or relation to the YAML, add an entry to `ontology/CHANGELOG.md` naming the question it lets
someone answer, run `make ontology`, and follow the compiler errors through mappers, rules, queries and
the UI. Keep the core small: every new kind has to justify itself by a question.

## Standing

Not all data has the same authority, and the model says so per data set:

- **secondary** (the default): a copy of a system that is authoritative for it. If the model disagrees
  with the source, the source wins and the correction goes there.
- **primary**: the Context Model is authoritative. Declared per data set by `DeclarePrimary`, with an
  owner, a requester and a different approver, all in the decision log.

An entity is primary when one of the data sets it is known by has been declared primary.
