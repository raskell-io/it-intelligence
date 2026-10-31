# it-intelligence: build brief for the reference Enterprise Context Platform

> Repository: `raskell-io/it-intelligence` (local only as of 2026-10-31; not yet on GitHub).
>
> **Build status (2026-10-31):** phases 0 to 5 are built, and phase 6 except drafts. `make up seed ingest
> verify` passes from a clean Docker start; every article check passes. Where the build departed from this
> brief, the brief has been updated and an ADR in `docs/adr/` explains why: own vocabulary generator instead
> of go-jsonschema (0002), no workflow engine (0003), no oapi-codegen or sqlc (0004). Still open: draft
> generators with the citation check, the Turtle export and SHACL check, a cluster live mode, publishing images.
> Companion to two raskell.io articles: *IT Intelligence: The Missing Data Layer of Enterprise IT*
> (`content/articles/it-intelligence.md`) and *Building IT Intelligence: A Reference Implementation in One
> Repository* (`content/articles/building-it-intelligence.md`). The second article describes this repository.
> Where this brief and that article disagree, this brief is the newer design; section 15 lists the changes the
> article needs.

This document is the complete brief for building the project. It is written for a Claude Code agent (or a human
engineer) starting from an empty repository. Read it fully before writing code. When this brief and your own
judgement disagree, write an ADR in `docs/adr/` and ask the owner before acting on it.

---

## 0. How to work on this

- **Work in phases** (section 13). Each phase ends with a working demo, green CI and updated docs. Do not start a
  phase before the previous one passes its acceptance criteria.
- **The article is a contract.** Every claim the companion article makes about the running system (a score, a
  queued link, a tool answer, a command) becomes an assertion in `make verify` (section 11). If the code and the
  article drift, CI says so.
- **The ontology comes first.** Nothing in `store/`, `mapping/`, `api/` or `web/` names an entity kind or a
  relation kind that is not in `ontology/`. Generated code is the only way those names enter the code base.
- **Boring over clever.** This is a reference people read in an afternoon. Prefer one obvious way, small files,
  plain SQL, and no framework the reader has to learn first.
- **Ask the owner before:** changing the licence, renaming the repository, adding a dependency outside
  Apache-2.0, MIT, BSD, ISC or MPL-2.0, adding an entity kind or relation kind beyond section 4, pushing to a public
  remote, or tagging a release.
- **Anonymity.** The Kestrel Group is fictional. Never use the name, products, internal tool names, naming
  conventions or tech stack of any real company in seed data, examples, comments or docs. Generic product
  categories only ("EA repository", "ITSM tool", "git host").

---

## 1. Mission

Show, in one repository anyone can clone and run on a laptop, what an Enterprise Context Platform is made of: an
ontology over IT, observations kept apart from the entities concluded from them, relationships with provenance and
confidence, sources scored in public, and one model served the same way to people (Web UI), systems (REST) and
agents (MCP).

**One sentence:** *it-intelligence turns the disagreeing exports of a company's IT tools into one sourced model of
what exists, who owns it and how it connects, and serves that model to people and agents with its evidence.*

### What success looks like

A reader runs four commands, opens a browser, and within ten minutes has seen: the EA repository scoring worst on
freshness, a review queue with a model's reasons for a link it is unsure about, a dashed edge in a generated
diagram, and a coding agent answering "who owns claims-prod and what depends on it?" in two tool calls with a cited
path. Then they open `ontology/context_model.yaml` and understand all of it.

### Two lives

The repository starts as a **reference** for the article. The owner may later grow it into a maintained open source
project. Design for the first, do not block the second: keep connectors and mappers pluggable, keep Kestrel-specific
knowledge in `mapping/` and `seed/`, never in the core. Nothing here is designed after or for any employer's
platform.

---

## 2. Users and use cases

| User | Surface | Use case in the demo |
|---|---|---|
| Coding agent in a repository (Claude Code and others) | MCP | ask who owns a service, what depends on it, which decisions constrain it, before writing code |
| Architect or engineer | Web UI | browse entities, explore a neighbourhood, read the scorecards, read generated insights and diagrams |
| Owning team | Web UI | confirm or reject inferred links in its review queue |
| Data set owner | Web UI, REST | request that a data set be declared primary |
| Another system | REST | pull entities and edges into portfolio, risk or cost reporting |
| Reader of the article | the repository | read the ontology, the store schema and one connector, and recognise every snippet from the post |

---

## 3. Principles (non-negotiable)

1. **Ontology first, generated everywhere.** One LinkML schema; JSON Schema, SQL vocabulary, OWL, SHACL, Go, and
   TypeScript types and docs are generated from it. CI fails when generated files are stale.
2. **Observations are append-only.** Nothing a source said is ever overwritten. Entities and edges can always be
   rebuilt from observations plus human decisions.
3. **Every relationship is a row** with a kind, a basis (`asserted`, `derived`, `inferred`), a confidence, a source,
   evidence (observation ids or a rule name or a model judgement), and a last-seen time.
4. **No inferred link becomes asserted without a person.** Confirmation and rejection are recorded with who and when,
   and both survive re-ingestion: a rejected link is never proposed again from the same evidence.
5. **The source wins for secondary data.** The model never edits secondary data; corrections go to the source. Only
   data sets declared primary are maintained in the model.
6. **One query layer.** REST, MCP and the artifact generators call the same Go package. An agent and a person looking
   at the same entity see the same facts.
7. **Answers carry provenance and stay small.** Every API and MCP answer that states a fact can say where it came
   from. MCP answers return ids to follow, not subgraphs.
8. **Runs offline by default.** `make up && make seed && make ingest` needs no API key and no network beyond pulling
   images. The model-assisted parts run against recorded judgements unless a provider is configured (section 7.4).
9. **Honest gaps.** What the reference leaves out is listed in the README and the article, and the code does not
   pretend otherwise (no fake auth, no fake scale).

---

## 4. The Context Model

### 4.1 Entity kinds (nine)

| Kind | Meaning | Typical sources |
|---|---|---|
| `Application` | a thing the business recognises as one system, however many parts it has | earepo, itsm (business services), resolution |
| `Repository` | a code repository | githost |
| `Workload` | a deployed unit on a cluster (deployment, stateful set, cron job) | cluster |
| `Namespace` | a cluster namespace; cluster name is an attribute | cluster |
| `Team` | an accountable team | githost, earepo, itsm |
| `Capability` | a business capability from the EA capability map, hierarchical | earepo |
| `Endpoint` | an externally or internally reachable host plus path prefix | gateway, waf |
| `Decision` | an architecture decision record | githost (ADR files in repositories) |
| `MarketUnit` | an organisational unit of the group | earepo |

Deliberately not kinds in the reference: business service (an ITSM observation that resolves to an `Application`),
incident (an attribute rollup on `Application`: count and last incident in 90 days), WAF policy (an attribute of
`Endpoint`: `waf: enforced | monitor | none`), cluster (an attribute of `Namespace`), data object (out of scope,
listed as a gap). Each of these is a reasonable kind in a real deployment; the reference keeps the core at nine to
honour "keep the core small".

### 4.2 Relation kinds (fifteen)

| Relation | Domain | Range | Cardinality | Typical basis |
|---|---|---|---|---|
| `realised_by` | Application | Repository | many | asserted (key), inferred |
| `deployed_as` | Application | Workload | many | asserted (label), derived |
| `runs_in` | Workload | Namespace | one | asserted |
| `built_from` | Workload | Repository | one | asserted (image source label), inferred |
| `owned_by` | Application, Repository, Workload, Namespace, Endpoint | Team | one accountable | asserted, derived |
| `part_of` | Team | MarketUnit | one | asserted |
| `operated_by` | Application | MarketUnit | many | asserted (earepo), derived |
| `supports` | Application | Capability | many | asserted (earepo) |
| `refines` | Capability | Capability | one parent | asserted |
| `exposed_via` | Application | Endpoint | many | derived (endpoint routes to a workload of the app) |
| `routes_to` | Endpoint | Workload | many | asserted (gateway) |
| `depends_on` | Application | Application | many | asserted (earepo interfaces), derived (gateway calls) |
| `constrains` | Decision | any entity | many | asserted (ADR front matter), inferred |
| `supersedes` | Decision | Decision | one | asserted |
| `replaced_by` | Application | Application | one | asserted (earepo successor field) |

"Cardinality one" means the store allows several candidate rows but at most one with `status = active` and the
highest confidence wins in answers; conflicts are surfaced as an insight (`conflicting_owner`), not hidden.

### 4.3 How it is written in LinkML

LinkML has no top-level `relations:` key. Relations are **slots** with `domain`, `range`, `multivalued`, and
annotations for what the store needs:

```yaml
# ontology/relations.yaml (trimmed)
id: https://example.org/it-intelligence/relations
name: relations
imports: [linkml:types, entities]
slots:
  realised_by:
    domain: Application
    range: Repository
    multivalued: true
    annotations: {ctx_relation: true, inverse_label: realises}
  owned_by:
    range: Team
    multivalued: false
    annotations: {ctx_relation: true, accountable: true}
    # domains listed on each class via slot_usage, LinkML allows one domain per slot
```

Files:

```text
ontology/
  context_model.yaml    the root schema: imports entities and relations, prefixes, Standing and Basis enums
  entities.yaml         the nine classes, their attributes and identity keys
  relations.yaml        the fifteen relation slots
  CHANGELOG.md          every change to a kind or relation, with the question it lets someone answer
  generated/            do not edit; see 4.4
```

Attributes per kind live on the class (`Application.lifecycle: Lifecycle` enum with `planned`, `production`,
`retiring`, `retired`; `Repository.truck_factor: integer`; `Endpoint.exposure: internal | internet`;
`Endpoint.waf: enforced | monitor | none`). Each class carries an annotation `ctx_id_pattern` (for example
`Workload: "{namespace}/{name}"`) that the mapping helpers use to build stable ids.

### 4.4 Generation

`make ontology` runs (Python via `uv`, pinned in `mise.toml`):

| Output | Generator | Consumed by |
|---|---|---|
| `generated/jsonschema/*.json` | LinkML JSON Schema | Go validation of entity attributes on write; `go-jsonschema` input |
| `generated/go/vocab/vocab.go` | the custom generator in `ontology/gen/` (ADR 0002): kinds, relations, attributes, enums | every Go package |
| `generated/ts/context_model.ts`, `generated/ts/vocabulary.ts` | LinkML TypeScript (enums, interfaces) and the custom generator (`Kind` and `Relation` union types) | `web/` |
| `generated/sql/vocabulary.sql` | **a small custom generator in `ontology/gen/`** (section 5.2), not LinkML SQL DDL | `store/` migrations |
| `generated/owl/context_model.owl.ttl`, `generated/shacl/context_model.shacl.ttl` | LinkML OWL, SHACL | anyone using semantic web tools; a CI job validates a Turtle export of the seeded model against the SHACL shapes |
| `generated/docs/` | LinkML docs | served by the web UI at `/ontology` |

Why a custom SQL generator: LinkML's SQL DDL generator creates one table per class, which contradicts the store
design (one `entity` table, one `edge` table, kinds as data). The vocabulary generator emits the kind and relation
tables plus the constraint that every edge respects its relation's domain and range. Keep it under 200 lines of
Python, standard library only plus `linkml-runtime` to load the schema.

---

## 5. The store (PostgreSQL)

### 5.1 Tables

The article says "five things": observations, entities, edges, scorecards and artifacts. The full schema adds the
small tables those five need to be correct. Group them in `store/schema.sql` under those five headings so the
article stays true.

```text
observations   observation        append-only, what a source said
               dataset            (source, dataset) registry: standing, owner, declared_at, expected_cycle

entities       entity             id, kind, name, standing, attributes jsonb, first_seen, last_seen
               alias              (entity_id, source, dataset, external_id) every source id an entity is known by

edges          edge               one row per (kind, from_id, to_id, source): basis, confidence, status,
                                  evidence jsonb, confirmed_by, confirmed_at, last_seen
               decision_log       every confirm, reject and declare-primary, who, when, why (append-only)

runs           run, run_step      workflow state for the runner (section 8)

scorecards     scorecard          (source, dataset, ingest_run): the four signals, overall, computed_at

artifacts      snapshot           id, created_at, content hash, path to the JSONL export
               artifact           id, kind, subject, snapshot_id, format, body, created_at
```

Additions to the article's `edge` table:

- `status text not null` with `active`, `pending` (inferred, waiting for review), `rejected`. The review queue is
  `where status = 'pending'`. Rejected rows stay, so the resolver can see them.
- `unique (kind, from_id, to_id, source)`. Several sources asserting the same link are several rows; that is what
  the `agreement` score counts.
- `evidence` shape by basis: asserted `{"observations": [ids]}`, derived `{"rule": "...", "edges": [ids]}`,
  inferred `{"judgement": {...}, "provider": "...", "model": "...", "prompt_version": "..."}`.

### 5.2 Generated vocabulary

`store/migrations/0001_vocabulary.sql` is a copy of `ontology/generated/sql/vocabulary.sql`:

```sql
create table entity_kind   (kind text primary key, description text not null);
create table relation_kind (kind text primary key, cardinality text not null, description text not null);
create table relation_domain (relation text references relation_kind, kind text references entity_kind,
                              primary key (relation, kind));
create table relation_range  (relation text references relation_kind, kind text references entity_kind,
                              primary key (relation, kind));
-- plus a trigger on edge that checks from/to kinds against domain/range
```

`entity.kind` and `edge.kind` reference these tables, so an edge the ontology does not allow cannot be written.

### 5.3 Migrations and queries

- Migrations: plain numbered SQL files, applied by `itx migrate` (a few lines of Go over `pgx`; no ORM, no
  migration framework the reader must learn).
- Queries: plain SQL over pgx in `api/query` (ADR 0004). The article's impact query is `impactSQL` there,
  verbatim apart from the status filter.
- Indexes: `edge (to_id, kind)`, `edge (from_id, kind)`, `alias (source, external_id)`, `observation (source,
  dataset, external_id, observed_at desc)`.

### 5.4 Snapshots

`BuildArtifacts` writes the active entities and edges as sorted JSONL to `var/snapshots/<id>.jsonl` (gitignored,
a volume in compose), records the content hash in `snapshot`, and stamps every artifact with the snapshot id.
`itx snapshot diff <a> <b>` prints added, removed and changed facts; that is how the article's "diff a landscape
over a quarter" claim is demonstrated. Edge history with validity intervals is out of scope (section 14).

---

## 6. Connectors and mapping

### 6.1 Connectors

The `Connector` interface is the article's, unchanged (`connectors/connector.go`). One package per source kind:

| Connector | Datasets | Seed format (imitates) | Live mode in the reference |
|---|---|---|---|
| `earepo` | applications, capabilities, interfaces, market_units, teams | CSV export of an EA tool | no |
| `itsm` | business_services, configuration_items, incidents | JSON export of an ITSM tool | no |
| `githost` | repositories, teams, codeowners, commit_stats, decisions (ADR files) | JSON as from a git host API | yes: GitHub REST for one org, read-only token |
| `cluster` | namespaces, workloads | `kubectl get -o yaml` lists | yes: Kubernetes API from the current kubeconfig context |
| `waf` | policies | JSON policy export | no |
| `gateway` | routes | YAML route config | no |

Every connector supports `file` mode (reads `seed/<source>/`) and declares whether it supports `live` mode. Live
modes exist to show that the interface is real, not to be production connectors (section 14).

### 6.2 Mapping

One mapper per connector in `mapping/`, as in the article: observations in, `Assertions` out (candidate entities,
candidate edges, aliases), each tagged with its observation. Mappers are pure functions; they are tested with table
tests against the seed files and never touch the database.

The deployment label convention for Kestrel is `app.kestrel.example/application-id`, carrying the EA application
id. It is Kestrel-specific and therefore lives in `mapping/cluster.go`, configurable in `config/mapping.yaml`.

### 6.3 Adding a source

Three files, as the article promises: `connectors/<name>/`, `mapping/<name>.go`, `seed/<name>/`. Plus one line in
`config/sources.yaml` (schedule, expected cycle, scoring weights). `docs/adding-a-source.md` walks through it with
the `waf` connector as the example, because it is the smallest.

---

## 7. Identity resolution (`resolver/`)

### 7.1 Passes

1. **Keys.** Aliases with a shared external id (EA application id in a deployment label, a CMDB reference in a
   repository's `catalog-info.yaml`, an ITSM business service's EA id field) merge into one entity. Basis
   `asserted`.
2. **Rules.** Deterministic, each with a name and a measured precision on the seed truth file (section 11):
   `same_normalised_name`, `team_owns_single_app`, `namespace_prefix_convention`, `owner_via_namespace`
   (the article's derived ownership). Basis `derived`, confidence = rule precision x source score.
3. **Inference.** Leftover candidates get a shortlist of at most five plausible matches (by trigram name similarity,
   same owner, same namespace), and a model returns the judgement JSON from the article. Basis `inferred`.

### 7.2 Thresholds

From `config/resolver.yaml`, defaults as in the article: at or above 0.9 `active`, 0.6 to 0.9 `pending` (queued for
the owning team of the target entity), below 0.6 dropped and logged to `var/log/resolver-dropped.jsonl`.

### 7.3 Rejection memory

Before proposing a link, the resolver checks for a `rejected` row with the same (kind, from, to). It proposes again
only if the evidence hash differs (new observations), and the review queue shows the earlier rejection next to it.

### 7.4 Model providers

`resolver/provider` has one interface (`Judge(ctx, Candidate, []Match) (Judgement, error)`) and three
implementations:

| Provider | When | Notes |
|---|---|---|
| `fixture` | default, CI, offline demo | replays `seed/judgements.jsonl`, keyed by candidate and match ids; deterministic, so the article's 0.82 is reproducible |
| `openai-compatible` | local models (Ollama, llama.cpp server, vLLM) and any compatible hosted endpoint | structured output via JSON schema |
| `anthropic` | hosted | structured output via tool use |

`itx resolver record` runs a real provider over the seed and rewrites `seed/judgements.jsonl`, so the fixture stays
honest: it was produced by a model, once, and committed. Prompts live in `resolver/prompts/` with a version string
that goes into every edge's evidence.

---

## 8. Workflows (`workflows/`, plain Go over PostgreSQL)

No workflow engine. The Go backend runs the ingestion workflows itself, and PostgreSQL keeps their state. For the
reference that is enough: ingestion is a short, linear sequence of steps, the volume is small, and a reader should
not need to learn Temporal or Restate to follow it. The four workflows of the article stay:
`IngestSource`, `ConfirmLink`, `DeclarePrimary`, `BuildArtifacts`.

### 8.1 The runner (about 300 lines, `workflows/runner/`)

```go
// workflows/runner/runner.go (shape, refine in code)
type Step struct {
	Name    string
	Run     func(ctx context.Context, run RunID) error // must be idempotent
	Timeout time.Duration
	Retry   RetryPolicy // max attempts, backoff
}

type Workflow struct {
	Name  string
	Steps []Step
}
```

- **State in two tables:** `run` (id, workflow, input jsonb, status `queued | running | succeeded | failed`,
  timestamps, actor) and `run_step` (run id, step, attempt, status, error, started_at, finished_at).
- **Claiming work:** runners poll with `select ... for update skip locked`, so several `itx` replicas can run
  without coordination and a run is never executed twice at the same time. A heartbeat column lets a stale claim
  (crashed process) be taken over after a timeout.
- **Resume:** a run restarts at the first step not marked `succeeded`. Steps pass a run id, never data; whatever a
  step produces it writes to PostgreSQL. That is what makes resume trivial and keeps the runner small.
- **Retries:** per step, exponential backoff with jitter, attempts recorded in `run_step`, so every failure stays
  visible.
- **Schedules:** `config/sources.yaml` holds an interval per source; a ticker in the runner enqueues
  `IngestSource` when the last successful run is older than the interval (no cron syntax, no separate scheduler).
- **Visibility:** `/runs` in the web UI and `GET /api/v1/runs` list every run with its steps, attempts, errors and
  durations. This replaces "one click away in the Temporal UI" in the article.

### 8.2 Rules for steps

- **Idempotent.** Observation inserts carry a content hash; edge writes are upserts on the unique key. A retried
  step changes nothing the first attempt already did.
- **Model calls are steps of their own** (`Resolve` splits into `ResolveKeys`, `ResolveRules`, `ResolveInfer`), so a
  timed-out model call retries without redoing extraction.
- **Score uses the previous scorecard for confidence.** Edge confidence depends on the source score, and the score of
  this run is computed after resolution. Use the last completed scorecard; document this in the code and the
  article.

### 8.3 The workflows (as built)

| Workflow | Steps | Trigger |
|---|---|---|
| `IngestSource` | Extract, Judge, Build, Score | interval per source, `itx ingest -source X`, `POST /api/v1/runs` |
| `IngestAll` | Extract per source (earepo first), Judge, Build, Score, Snapshot, Diagrams, Insights | `make ingest` |
| `ConfirmLink` | Apply (append to `decision_log`), Build, Score | confirm or reject in the UI |
| `DeclarePrimary` | Apply (requester and a different approver), Build | `POST /datasets/{source}/{dataset}/primary` |
| `BuildArtifacts` | Snapshot, Diagrams, Insights (Drafts not built yet) | nightly interval |

The model is rebuilt whole on every Build: `resolver.Build` is a pure function of the latest observations,
the stored judgements and the decision log. Judge is the only step that calls a model and caches every
answer, so Build never does. Edge and alias ids are hashes of their content, stable across rebuilds.

`make ingest` runs `IngestSource` for earepo first (it defines applications, capabilities and market units), then
the other five in parallel, then `BuildArtifacts`.

### 8.4 Why not a workflow engine, and when to add one

Write this into `docs/adr/0003-no-workflow-engine.md` and the article. A durable execution engine (Temporal, Restate)
earns its place when workflows are long-running, wait on people or external events for days, fan out widely, or
must survive arbitrary code paths, not just a fixed list of steps. A real deployment with hundreds of sources and
slow, rate-limited APIs gets there. The runner is built so that moving is mechanical: steps are already idempotent
functions taking a run id, which is exactly the shape of a Temporal activity or a Restate handler.

The runner lives in the same `itx` binary and process as the API, so one image and one process serve everything.

---

## 9. Scoring

The four signals and their definitions are the article's. Make each one a named SQL query in
`store/queries/score_*.sql`, computed per (source, dataset, ingest run):

- `freshness`: share of records whose source-side modification time, or a confirmation in `decision_log`, falls
  within the data set's `expected_cycle`.
- `completeness`: share of records with every field the mapper marks required (mappers declare them).
- `agreement`: share of a data set's facts (edges and identifying attributes) that at least one other source also
  asserts.
- `resolvability`: share of records whose alias resolved to an entity.

`overall` is a weighted mean, weights per source in `config/sources.yaml`. Every number in the UI links to the
records behind it (`/scorecards/<source>/<dataset>/<signal>` lists them). Scorecards are also written as YAML to
`var/scorecards/` in the shape of the first article's illustrative example, for people who want them in Git.

---

## 10. Surfaces

### 10.1 One binary: `itx`

```text
itx serve      REST on /api/v1, MCP on /mcp (streamable HTTP), artifacts on /artifacts, health on /healthz
itx runs       list and inspect workflow runs (the runner itself starts inside `itx serve`)
itx migrate    apply store migrations
itx seed       load nothing into the model; registers seed paths and data sets (sources are read at ingest)
itx ingest     start IngestSource for one or all sources and wait
itx snapshot   list, export, diff
itx resolver   record fixtures, explain a candidate
itx verify     the article checks (section 11)
```

Layout: `cmd/itx/` is the only `main`. `api/query/` is the shared query layer; `api/rest/` and `api/mcp/` are thin
adapters over it. A test iterates every MCP tool and asserts that its REST twin returns the same JSON for the same
input.

### 10.2 REST

Hand-written OpenAPI 3.1 in `api/openapi.yaml`, handlers on the standard library mux (ADR 0004). As built,
entity ids travel as an `id` query parameter (`/entity?id=`, `/owner?id=`, `/impact?id=`, `/decisions?id=`,
`/diagram?id=`) rather than in the path; the list below is the original plan.

```text
GET  /api/v1/entities?q=&kind=            search
GET  /api/v1/entities/{id}                entity, attributes, standing, direct edges
GET  /api/v1/entities/{id}/owner          find_owner
GET  /api/v1/entities/{id}/impact?depth=  impact_of_change
GET  /api/v1/entities/{id}/decisions      list_decisions
GET  /api/v1/entities/{id}/diagram?view=  Mermaid text
GET  /api/v1/edges/{id}/explain           explain_fact
GET  /api/v1/review?team=                 pending edges for a team
POST /api/v1/edges/{id}/confirm|reject    starts ConfirmLink
GET  /api/v1/scorecards                   latest per source and data set
GET  /api/v1/datasets                     standing, owner, declared_at
POST /api/v1/datasets/{id}/primary        starts DeclarePrimary
GET  /api/v1/artifacts?kind=&subject=     artifacts with snapshot ids
GET  /api/v1/export.ttl                   the active model as Turtle, for the SHACL check
```

Entity ids contain `:` and `/`; they are URL-encoded in paths. Write endpoints require an `X-Actor` header naming a
seeded person (section 10.5).

### 10.3 MCP

The official Go SDK (`github.com/modelcontextprotocol/go-sdk`), streamable HTTP at `/mcp`. The seven tools of the
article, with the article's input and output shapes. Design rules, enforced by tests:

- Every tool output that states a relationship includes the edge `basis`, `confidence`, `source` and edge id, so the
  agent can call `explain_fact`.
- Output budget: no tool returns more than 50 items or about 4 KB of JSON by default; larger results are truncated
  with a `next` cursor and a count.
- Tool descriptions say when to use the tool and what to call next ("use `explain_fact` with an edge id to see why").
- `find_owner` accepts an id or free text; free text goes through `search_entities` first and returns the candidates
  if the match is ambiguous, rather than guessing.

Also expose two MCP resources: `ctx://ontology` (the generated docs as Markdown) and `ctx://scorecards`.
The repository's `.mcp.json` points at `http://localhost:8080/mcp`, as in the article.

### 10.4 Web (`web/`, Next.js, app router, TypeScript)

The five views of the article: entity pages, graph view (Cytoscape.js; render Mermaid only for artifacts), review
queue per team, scorecards, artifacts. Plus `/ontology` (generated docs) and an "act as" picker. Types come from
`ontology/generated/ts/`; a CI step runs `tsc --noEmit` after `make ontology`, so an ontology change that the
frontend has not caught up with fails the build. The frontend talks only to the REST API; no direct database access.

### 10.5 Identity in the reference

There is no authentication, as the article says. Writes still need a name, because confirmations carry one: the UI
has an "act as" picker over the seeded people of Kestrel, and the API takes `X-Actor`. The README says plainly that
this is a placeholder for real identity, and the API refuses writes without the header.

---

## 11. The Kestrel Group seed, and its answer key

### 11.1 Shape

Three market units (North, Central, South), about forty applications, twelve teams, about seventy repositories,
two clusters (`cl-a`, `cl-b`), one WAF, one API gateway, eight to ten ADRs, roughly 25 capabilities in a two-level
map. Names, people and hosts are invented; hosts use `kestrel.example`. People are role-like invented names.

Generate the seed with a script (`seed/gen/`, deterministic, seeded RNG), commit the output, and commit the script.
Hand-place the planted problems below on top of the generated background, so they read like real mess and not like
noise.

### 11.2 Planted problems (each one must be findable)

| # | Mess | Where it shows |
|---|---|---|
| 1 | Policy Administration is "Policy Administration" (earepo), "PAS" (itsm), `pol-adm-svc` and `pol-adm-ui` (githost), `policy-admin` in `claims-prod` (cluster). Only `pol-adm-svc` carries a key. | resolution; `pol-adm-ui` inferred at 0.82, pending for claims-platform |
| 2 | Two applications claim the same repository in earepo | `conflicting_realisation` insight; agreement score |
| 3 | One namespace has no owner label | owner derived through workload and application, not asserted; one namespace with no owner at all |
| 4 | Market unit South runs its own integration platform, present in cluster and gateway, absent from earepo | an `Application` created from cluster and gateway only, standing secondary, flagged `not_in_ea` |
| 5 | earepo lists an application as production that was switched off a year ago (no workload, no commits, no incidents) | `ghost_application` insight; earepo freshness |
| 6 | earepo says retired, workloads still run | `retired_but_running` insight |
| 7 | An internet endpoint with `waf: none` | `exposed_without_waf` insight |
| 8 | Capability "Payments" supported by two platforms in market unit Central | `duplicate_capability_support` insight |
| 9 | A repository with one committer in twelve months behind a production application | `truck_factor_one` insight |
| 10 | A production application with no owner at confidence 0.8 or higher | `unowned_production_app` insight (the article's SQL) |
| 11 | earepo maintained yearly: most records untouched for more than 180 days | earepo is the worst source on freshness |

### 11.3 The answer key

`seed/truth.yaml` lists the entities Kestrel really has, which aliases belong to each, the true edges, and the
expected insight findings. It is used two ways:

- **Resolver quality.** `itx verify --resolver` reports precision and recall per pass against the truth, and CI
  fails if they fall below the committed baseline. This is what makes rule confidence (7.1) a measurement.
- **Article checks.** `itx verify --article` asserts every concrete claim of the companion article:

  ```text
  earepo has the lowest freshness of all sources
  claims-platform's review queue holds pol-adm-ui -> policy-admin at 0.82 with three reasons
  find_owner(ctx:Namespace/claims-prod) returns claims-platform, derived, with a three-step path
  impact_of_change(ctx:Application/payment-gateway, 4) is non-empty and grouped by distance
  the policy-admin diagram has exactly one dashed edge, labelled "inferred 0.82"
  every planted insight in 11.2 is found, and nothing else at error level
  ```

`make verify` runs both after a fresh `make up seed ingest`, in CI, on every pull request.

---

## 12. Artifacts (`artifacts/`)

Three generator families, each a Go package that reads a snapshot through `api/query`:

- **Diagrams.** Mermaid flowchart of an entity's neighbourhood; inferred edges dashed with confidence; C4-style
  context view per market unit (Mermaid `C4Context` or plain flowchart, decided in an ADR).
- **Insights.** One SQL file per insight in `artifacts/insights/` with a front matter comment (`name`,
  `description`, `severity`, `threshold`). The runner executes all, stores findings with entity ids.
- **Drafts.** Current-state section, ADR draft, consolidation business case. Generated by a model (same providers
  as the resolver, `fixture` by default with recorded outputs). Citation rule, enforced by a validator: every
  sentence that states a fact ends with one or more citations `[ctx:Kind/id]` or `[edge:123]`, every citation must
  exist in the snapshot, and a draft with an uncited factual paragraph is rejected and logged, not stored. Drafts
  are Markdown with a header naming the snapshot and the words "Draft, generated, check every citation".

---

## 13. Phased plan and acceptance criteria

### Phase 0: foundations
Repository skeleton (section 16), `mise.toml` pinning Go, Node, Python and uv, AGENTS.md and CLAUDE.md, licence,
CI skeleton. Ontology complete (nine kinds, fifteen relations), all generators wired, `make ontology` and the
stale-check in CI. ADRs: Go type generator, Mermaid C4 or flowchart, licence, no workflow engine (8.4).
*Accept:* `make ontology` is clean on a fresh clone; CI fails when a generated file is edited by hand.

### Phase 1: store, connectors, mapping, seed
Schema and migrations, vocabulary trigger, the six connectors in file mode, six mappers with table tests, the seed
generator and planted problems, `truth.yaml`. Steps run synchronously from `itx ingest` until the runner exists in phase 3.
Key pass of resolution only.
*Accept:* ingest of all six sources produces asserted entities and edges; an edge that violates domain or range is
rejected by the database; mapper tests pass.

### Phase 2: resolution and review
Rule pass, inference pass with the three providers, fixtures recorded once with a real model, thresholds, rejection
memory, `decision_log`. `itx verify --resolver` with baseline.
*Accept:* `pol-adm-ui` lands pending at 0.82; rejecting it and re-ingesting does not re-queue it; resolver
precision and recall meet the baseline.

### Phase 3: workflows and scoring
The runner (8.1), the four workflows, intervals, idempotent steps, `/runs`, the four scores, scorecard YAML export.
*Accept:* killing `itx serve` mid-ingest and restarting it resumes the run at the failed step with no duplicate rows; earepo has the
lowest freshness.

### Phase 4: API and MCP
`api/query`, REST with OpenAPI, MCP with seven tools and two resources, parity test, output budget test,
`.mcp.json`. `X-Actor` on writes.
*Accept:* Claude Code in the repository answers "who owns claims-prod and what depends on it?" with a cited path in
two tool calls (record the transcript in `docs/demo/agent-transcript.md`).

### Phase 5: web
The five views, `/ontology`, act-as picker, typed from the ontology.
*Accept:* every article claim in section "Running it" can be seen in the browser; screenshots in `docs/demo/`.

### Phase 6: artifacts
Snapshots and diff, diagram generators, the insight runner with every planted insight, drafts with the citation
validator.
*Accept:* `itx verify --article` passes end to end.

### Phase 7: release as reference
`deploy/helm/` chart (itx and web; PostgreSQL as a dependency or external), README with the
four commands and the gaps list, CONTRIBUTING.md, `docs/adding-a-source.md`, align the article (section 15).
Owner decides on going public.
*Accept:* a fresh clone on a clean machine reaches the browser demo in under ten minutes with `make up seed ingest`.

---

## 14. Non-goals (say so in the README)

Authentication and authorisation; production-grade connectors (pagination, rate limits, credentials, schema drift);
edge history with validity intervals; scale beyond a few thousand entities (indexes exist, nothing more); incremental
artifact builds; ontology governance beyond code review; data objects and classification; any real company's data
or conventions.

---

## 15. Changes the article needs

Apply these to `content/articles/building-it-intelligence.md`. Numbers are what the running system
produces on the seed after one `make ingest`.

1. **Temporal is gone (ADR 0003).** The shape diagram, the stack table (Temporal moves to alternatives, next
   to Restate), layer five and the `IngestSource` snippet. Replace the snippet with a `runner.Workflow{Steps:
   ...}` definition (`workflows/workflows.go`), describe the runner (two tables, `skip locked`, heartbeat,
   per-step retries, resume), point to `/runs` instead of the Temporal UI, and say when an engine becomes
   worth it. The workflow table gets `IngestAll` and the step names Extract, Judge, Build, Score.
2. **Relations snippet is not valid LinkML.** Show slots with `domain`, `range`, `multivalued` and the
   `ctx_domain`/`ctx_range` annotations (`ontology/relations.yaml`); add `entities.yaml` and `relations.yaml`
   to the tree.
3. **Nine kinds:** add `MarketUnit` to the trimmed model. Fifteen relation kinds, exactly.
4. **"SQL for the store"** becomes "the store's vocabulary": kind and relation tables and a trigger that
   rejects edges outside their domain and range (ADR 0001, 0002).
5. **Store:** keep the five headings, add one sentence for aliases, the data set registry, the decision log,
   judgements, snapshots and run state. In the `edge` snippet: text ids (stable hashes), `status`, the unique
   key, `double precision` confidence.
6. **Mapping snippet:** `a.Ref("Application", app)` becomes `mapping.Alias(vocab.Application,
   "earepo:applications", app)`; constants come from the generated `vocab` package; the method is
   `Map(ctx, o)`.
7. **Identity resolution:** the `same_as` case (PAS, the ITSM business service) resolves at 0.93; describe
   the shortlist (owning team first, then similar names, at most five).
8. **Scoring:** resolvability is "every identity and reference resolved"; agreement is what punishes facts
   only one source holds. Asserted facts start at `0.7 + 0.3 x` the source's last overall score. On the seed:
   earepo applications 0.58, capabilities 0.45; cluster 0.75; itsm business services 0.99.
9. **find_owner example:** confidence is 0.92 after the first ingest (0.88 after the second, once real
   scores apply), and the path runs through `claims-settlement` (all three claims applications qualify;
   the strongest chain wins, ties by id). Use the real output of `curl localhost:8080/api/v1/owner?id=claims-prod`.
10. **Running it:** `make up`, `make seed`, `make ingest`, `make verify`, then open localhost:3000. No API key:
    judgements are recorded fixtures; `config/resolver.yaml` switches to a local or hosted model.
11. **Identity:** the "act as" picker and `X-Actor`.
12. **Add the strongest paragraph:** the seed ships with an answer key, and `itx verify` checks this post's
    claims against the running system in CI. Measured: edge precision 0.996, recall 0.994, identity 1.0;
    every rule's precision is measured, and one (`team_owns_single_app`) measures 0 of 1, which is the kind
    of thing the reference is meant to show.
13. **Drafts** (ADR and business case generators) are not built: either cut that paragraph to a sentence or
    build them before publishing.

## 16. Repository layout

```text
it-intelligence/
  AGENTS.md  CLAUDE.md  README.md  CONTRIBUTING.md  LICENSE-APACHE  LICENSE-MIT  BRIEF.md
  mise.toml  Makefile  go.mod  .mcp.json
  ontology/      context_model.yaml  entities.yaml  relations.yaml  CHANGELOG.md  gen/  generated/
  cmd/itx/       the one binary
  connectors/    connector.go  earepo/ itsm/ githost/ cluster/ waf/ gateway/
  mapping/       one mapper per connector, assertions.go
  resolver/      keys.go rules.go infer.go provider/ prompts/
  workflows/     runner/ workflows.go score.go
  store/         store.go migrations/ vocabulary.sql (generated)
  api/           query/ rest/ mcp/ openapi.yaml
  artifacts/     diagrams/ insights/ drafts/
  web/           Next.js app
  config/        sources.yaml resolver.yaml mapping.yaml
  seed/          earepo/ itsm/ githost/ cluster/ waf/ gateway/ truth.yaml judgements.jsonl gen/
  deploy/        compose.yaml helm/
  docs/          adr/ adding-a-source.md demo/
  var/           gitignored: snapshots, scorecards, logs
```

One Go module at the root, Go 1.25 or newer; tools pinned in `mise.toml`. Make targets: `ontology`, `generate`, `up`, `down`, `seed`, `ingest`, `verify`,
`test`, `lint`, `demo` (all of `up seed ingest verify`).

---

## 17. Open questions for the owner

1. **Licence:** Apache-2.0 OR MIT is the default here. Apache-2.0 alone if the project may grow and
   a patent grant matters.
2. **Name:** `it-intelligence` matches the article. If it becomes a maintained project it may want a proper name;
   decide before the first public push, since the article links the URL.
3. **Frontend:** the article says Next.js; `raskell-io/blueprint-app` uses SvelteKit. Next.js is kept to match the
   article and the larger audience; switch only together with the article.
4. **Live connectors:** GitHub and Kubernetes only, or none in the first release.
5. **Timeline:** the article is dated 2026-11-16. Phases 0 to 4 are the minimum for the article to be true; phases 5
   and 6 carry the screenshots and the artifact claims.

---

## 18. References

- LinkML, https://linkml.io/ and its generators, https://linkml.io/linkml/generators/
- PostgreSQL `SELECT ... FOR UPDATE SKIP LOCKED`, https://www.postgresql.org/docs/current/sql-select.html#SQL-FOR-UPDATE-SHARE
- Temporal, https://temporal.io/ and Restate, https://restate.dev/ (the upgrade path in 8.4, not used)
- Model Context Protocol Go SDK, https://github.com/modelcontextprotocol/go-sdk
- pgx, https://github.com/jackc/pgx
- PostgreSQL recursive queries, https://www.postgresql.org/docs/current/queries-with.html
- Apache AGE, https://age.apache.org/ (named in the article as the next step, not used here)
- Mermaid, https://mermaid.js.org/ ; Cytoscape.js, https://js.cytoscape.org/
- Backstage software catalog `catalog-info.yaml`, https://backstage.io/docs/features/software-catalog/
