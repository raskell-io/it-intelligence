# it-intelligence

A reference implementation of an **Enterprise Context Platform**: one model of what exists in a company's
IT, who owns it, how it connects and whether it matches the strategy, built from the disagreeing exports
of the company's own tools, and served with its evidence to people (web UI), systems (REST) and coding
agents (MCP).

It is the companion of two articles on raskell.io: *IT Intelligence: The Missing Data Layer of Enterprise
IT* (the argument) and *Building IT Intelligence: A Reference Implementation in One Repository* (this
repository, layer by layer).

Inside is the **Kestrel Group**, a fictional financial group with three market units, 38 applications,
12 teams, two clusters, a WAF and an API gateway. Its seed data is deliberately messy, and it comes with an
answer key, so the claims the article makes are checked in CI.

## Run it

You need Docker. Nothing else, and no API key.

```sh
git clone https://github.com/raskell-io/it-intelligence
cd it-intelligence
make up       # PostgreSQL, itx (REST, MCP, workflow runner) and the web UI
make seed     # register the Kestrel Group sources
make ingest   # extract, judge, build, score and build artifacts
make verify   # the resolver against the answer key, the article's claims against the system
open http://localhost:3000
```

Three places are worth a look after the first ingest:

- **Scorecards**: the EA repository export scores worst on freshness. Most of its records were last touched
  more than 180 days ago, among them an application that was switched off a year ago.
- **Review queue**: the claims platform team has an inferred link waiting, from the repository
  `pol-adm-ui` to Policy Administration, at 0.82, with the model's three reasons.
- **A coding agent**: this repository carries a `.mcp.json`. Open Claude Code here and ask *"who owns
  claims-prod and what depends on it?"*; it answers in two tool calls and cites its path through the graph.

## What is in it

```text
ontology/      the Context Model in LinkML: 9 entity kinds, 15 relation kinds; generated/ is make ontology
connectors/    one package per source kind; they read and emit observations, nothing else
mapping/       observations -> candidate entities and edges, one mapper per connector
resolver/      identity resolution: keys, rules, then a model; providers for recorded, local and hosted models
workflows/     IngestSource, IngestAll, ConfirmLink, DeclarePrimary, BuildArtifacts; runner/ runs them
store/         PostgreSQL: schema, migrations, the vocabulary generated from the ontology
api/           query/ is the one query layer; rest/ and mcp/ are thin adapters over it
artifacts/     Mermaid diagrams, insight queries (insights/*.sql), snapshots
web/           Next.js: entity pages, graph, review queue, scorecards, artifacts, runs, ontology
seed/          the Kestrel Group: exports of six tools, truth.yaml (the answer key), judgements.jsonl
config/        sources, mapping conventions, resolver thresholds and rules, people, verify baseline
cmd/itx/       the one binary: serve, ingest, verify, runs, resolver record, snapshot
deploy/        docker compose for a laptop, a Helm chart for a cluster
docs/          ADRs, adding a source
```

The stack is open source and unsurprising: LinkML, Go, PostgreSQL, the official MCP Go SDK, Next.js,
Mermaid. There is no workflow engine: ingestion is a short list of steps, and a ~300-line runner over
PostgreSQL retries and resumes them (see `docs/adr/0003-no-workflow-engine.md` for when that stops being
enough).

## Without Docker for itx and the UI

```sh
make dev-db       # PostgreSQL in a container on :55432
make dev-ingest   # build itx and ingest the seed, in-process
make dev-serve    # REST on :8080/api/v1, MCP on :8080/mcp
make dev-web      # the UI on :3000
make dev-verify
```

Go, Node and uv versions are pinned in `mise.toml`. `make ontology` regenerates everything from the LinkML
schema (it needs uv); CI fails when a generated file is out of date.

## Using a real model

The resolver asks a model about what keys and rules cannot settle. By default it replays
`seed/judgements.jsonl`, so the demo is offline and reproducible. To use a model, set `provider` in
`config/resolver.yaml`:

- `openai-compatible` with a local model, for example Ollama (`base_url: http://localhost:11434/v1`). The
  inputs are metadata and the prompts are small; a local model does the job.
- `anthropic`, with `ANTHROPIC_API_KEY` set. The default Docker image is built without this provider
  (`-tags noanthropic`) because compiling its SDK needs more memory than a 2 GB Docker VM has; build with
  `--build-arg GO_TAGS=` or run itx on the host.

`itx resolver record -provider anthropic` re-records the fixtures with a real model.

## What it leaves out

A reference earns trust by being honest about its gaps.

- **Identity and access.** No authentication. Writes carry a name from the "act as" picker
  (`config/people.yaml`), because confirmations and primary declarations need one; reads are open, as they
  should be.
- **Real connectors at scale.** File mode reads the seed; only the git host has a (small) live mode for one
  GitHub organisation. Pagination, rate limits, credentials and schema drift are the unglamorous work left.
- **History.** Observations are append-only; entities and edges are rebuilt in place. Point-in-time
  questions need edge history with validity intervals. Snapshots and `itx snapshot diff` are the cheap
  substitute.
- **Scale.** The whole model is rebuilt on every ingest. Fine for thousands of entities, not for millions.
- **Drafts.** The generators for ADR drafts and business cases, with their citation check, are not built yet.
- **Ontology governance.** Code review supports it; it does not answer who may add an entity kind.

## Licence

Apache-2.0 OR MIT, at your option. The Kestrel Group and everything about it is fictional.
