# Architecture

it-intelligence turns the disagreeing exports of a company's IT tools into one model of what exists,
who owns it and how it connects, and serves that model with its evidence. This page walks through the
parts in the order data flows through them, then the rules that hold them together.

If you only read one thing: **the model is a pure function of what the sources said, what a model
judged, and what people decided.** Everything else follows from that.

```text
 sources ─▶ connectors ─▶ observations ─▶ mappers ─▶ assertions ─▶ resolver ─▶ model ─▶ surfaces
           (read only)   (append-only)   (pure)                   keys          entities   REST
                                                                  rules         edges      MCP
                                          judgements (cached) ─▶  inference     aliases    web UI
                                          decision log ────────▶  decisions                artifacts
```

## The parts

| Part | Package | Does | Knows about the ontology? |
|---|---|---|---|
| Connectors | `connectors/*` | read a source, emit one observation per record | no |
| Store | `store` | keeps observations append-only, writes each model build in one transaction | the vocabulary tables only |
| Mappers | `mapping` | turn an observation into candidate entities, edges and aliases | yes, through the generated `vocab` package |
| Resolver | `resolver` | decides identity and relationships: keys, rules, inference, decisions | yes |
| Providers | `resolver/provider` | ask a model about a candidate: recorded fixtures, a local model, Claude | no |
| Workflows | `workflows`, `workflows/runner` | run ingestion as idempotent steps with retries and resume | no |
| Scoring | `workflows/score.go` | compute four signals per data set | no |
| Query layer | `api/query` | every question anyone can ask, in plain SQL | yes |
| Surfaces | `api/rest`, `api/mcp`, `web` | thin adapters over the query layer | through generated types |
| Artifacts | `artifacts` | snapshots, Mermaid diagrams, insight queries | yes |
| Ontology | `ontology` | the single source of every kind and relation name | it is the ontology |

## Data flow, one ingest

`IngestAll` (what `make ingest` runs) is a list of steps. Each step reads from PostgreSQL and writes to
it; nothing passes between steps but the run id.

```text
 Extract earepo │ Extract cluster │ ... │ Judge │ Build │ Score │ Snapshot │ Diagrams │ Insights
 ───────────────┴─────────────────┴─────┴───────┴───────┴───────┴──────────┴──────────┴─────────
  observations    observations           judgement entity     scorecard snapshot  artifact  finding
                                                   alias                            (mermaid) artifact
                                                   edge
                                                   record_status
                                                   dropped
```

1. **Extract** runs each connector. A record whose content did not change keeps its row; only its
   `last_seen_run` moves. A record missing from the extract is gone from the source and drops out of the
   latest set. Nothing is overwritten.
2. **Judge** maps the latest observations, runs keys and rules in memory, and asks the configured model
   about every candidate that has no judgement for its current inputs. It is the only step that calls a
   model, so a timed-out call retries without redoing extraction.
3. **Build** maps and resolves again, this time with every judgement cached, applies the decision log,
   and replaces the model in one transaction. Readers see the old model until the commit.
4. **Score** computes freshness, completeness, agreement and resolvability per data set. The next build
   starts asserted facts with a confidence that reflects these scores.
5. **Snapshot, Diagrams, Insights** export the model as sorted JSON lines, draw a Mermaid neighbourhood
   per application and run every insight query, each stamped with the snapshot id.

`IngestSource` does the same for one source; `ConfirmLink` appends a decision and rebuilds;
`DeclarePrimary` records a data set's new standing and rebuilds.

## Why rebuild everything

The model is rebuilt whole on every build, from the latest observations, the stored judgements and the
decision log (`resolver.Build`). That costs a fraction of a second on the seed and seconds on a few
thousand applications, and it buys three things:

- **Explainability.** Every fact can point to the observations, the rule and chain, or the judgement it
  came from, because it was computed from them a moment ago.
- **Determinism.** The same inputs give the same model. Edge and alias ids are hashes of their content,
  so links to an edge survive rebuilds.
- **Safe change.** A new rule, a fixed mapper or a changed threshold takes effect on the next build,
  everywhere, with no migration of derived data.

It stops being the right choice when the model gets large enough that a rebuild takes minutes. The
next step then is incremental resolution per changed observation, not a different architecture.

## The store

```text
observations   observation        append-only: source, dataset, external_id, payload, content_hash
               dataset            standing (secondary | primary), owner, expected cycle
entities       entity             id, kind, name, standing, attributes (jsonb), sources, last_seen
               alias              every id an entity is known by, with basis, confidence and status
               record_status      what became of each latest observation
edges          edge               kind, from, to, basis, confidence, status, source, evidence
               decision_log       every confirm, reject and primary declaration, append-only
               judgement          every model answer, keyed by candidate and input hash
scorecards     scorecard          the four signals and the records that fail each
artifacts      snapshot, artifact, finding
runs           run, run_step      state of the workflow runner
vocabulary     entity_kind, relation_kind, relation_domain, relation_range   (generated)
```

A trigger on `edge` checks every row against the generated vocabulary: an edge the ontology does not
allow cannot be stored.

## The rules that hold it together

1. **Ontology first.** Every kind and relation name comes from `ontology/`. See
   [context-model.md](context-model.md).
2. **Connectors only read.** Company conventions live in mappers and `config/mapping.yaml`.
3. **Observations are append-only.** Fix a fact at its source, in a mapper or in a rule, never in a row.
4. **Every relationship has a basis, a confidence and a source.** See
   [identity-resolution.md](identity-resolution.md).
5. **No inferred link becomes asserted without a person.** Rejections stick until the evidence changes.
6. **The source wins for secondary data.** Only data sets declared primary are maintained in the model.
7. **One query layer.** REST and MCP adapt `api/query`; a test checks they agree. See
   [agents.md](agents.md).

## Runtime

One binary, `itx`, serves REST, MCP and the workflow runner in one process. Several replicas can run
side by side: the runner claims work with `SELECT ... FOR UPDATE SKIP LOCKED`, a heartbeat lets another
replica take over a run whose process died, and model builds serialise on an advisory lock. The web UI is
a separate Next.js process that proxies `/api/v1` to itx. See [operations.md](operations.md).

## Decisions

The reasons behind the larger choices are in [adr/](adr/): the ontology in LinkML (0001), a generic store
with our own vocabulary generator (0002), no workflow engine (0003), and no code generators for REST and
SQL (0004).
