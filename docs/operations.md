# Operations

How to run it, configure it, look into it, and what changes when it runs for a real company.

## Processes

```text
 ┌──────────────┐     ┌──────────────────────────────────┐     ┌────────────┐
 │ web (Next.js)│────▶│ itx serve                        │────▶│ PostgreSQL │
 │ :3000        │     │ :8080 /api/v1  /mcp  /healthz    │     │            │
 └──────────────┘     │ workflow runner (+ schedules)    │     └────────────┘
                      └──────────────────────────────────┘
```

| Process | Command | Notes |
|---|---|---|
| itx | `itx serve [-addr :8080] [-schedules] [-no-runner]` | REST, MCP and the runner in one process; stateless apart from PostgreSQL and `var/` |
| web | `node server.js` (image) or `next start` | proxies `/api/v1` to `ITX_API` at request time |
| PostgreSQL | 17 | the only state that matters |

Several itx replicas can run at once. Runs are claimed with `FOR UPDATE SKIP LOCKED`; a run whose process
died is taken over when its heartbeat is two minutes old; model builds serialise on an advisory lock.

## Configuration

| File or variable | What |
|---|---|
| `ITX_DATABASE_URL` | PostgreSQL connection string (default `postgres://itx:itx@localhost:55432/itx`) |
| `ITX_ROOT` | where `config/` and `seed/` are (default `.`, `/app` in the image) |
| `ITX_ADDR`, `ITX_SCHEDULES=1` | listen address; ingest every source on its interval |
| `config/sources.yaml` | sources (mode, directory, interval, expected cycle), source order, score weights, `reference_time` for the seed |
| `config/mapping.yaml` | the company's conventions: label names, the EA id pattern, ITSM support group suffixes |
| `config/resolver.yaml` | thresholds, confidence formula, rules and their precision, the model provider |
| `config/people.yaml` | who the "act as" picker offers (no authentication in the reference) |
| `config/verify.yaml` | the resolver quality CI accepts |
| `ANTHROPIC_API_KEY`, `OPENAI_API_KEY`, `GITHUB_TOKEN` | only for the anthropic provider, an authenticated openai-compatible endpoint, the git host live mode |

`reference_time` fixes "now" for the Kestrel seed so scores and checks are reproducible. Remove it when
the connectors read live sources.

## The itx command

```text
itx serve                       REST, MCP, runner
itx migrate                     apply the vocabulary and the migrations (every command does this first)
itx seed                        list the configured sources
itx ingest [-source X] [-inline]  run IngestAll or IngestSource and wait; -inline runs it in this process
itx verify [-resolver] [-article] the answer key and the article's claims
itx runs [ID]                   recent runs, or one with its steps
itx resolver record -provider anthropic|openai-compatible [-model M] [-base-url U]
itx snapshot list | diff A B    model snapshots, and what changed between two
```

## Looking into it

- **/runs** in the UI (or `itx runs ID`): every run with each step's attempts, duration, output and error.
  When a scorecard drops, start here.
- **Scorecards**: every signal per data set, with the records that fail it.
- **`dropped`** table: what the last build could not use (dangling references, judgements below the
  threshold), with the reason.
- **`explain_fact`** / the "why?" links: the evidence behind any relationship.
- **`itx snapshot diff`**: facts added and removed between two snapshots, the landscape's changelog.

## Backups and rebuilds

Back up PostgreSQL. Of its tables, three cannot be recomputed: `observation` (what sources said),
`decision_log` (what people decided) and `judgement` (what a model answered; recomputable, at the cost of
asking again). Everything else is rebuilt by the next `Build` step.

## Moving from the reference to a company

What changes, roughly in order:

1. **Connectors.** Live modes against the real tools, with pagination, rate limits, credentials and a
   plan for sources that change their schema. Keep them read-only.
2. **Mapping conventions.** `config/mapping.yaml` and the mappers: your label names, id patterns, naming
   conventions. Add shared keys to deployments and repositories; each one replaces an inference.
3. **Identity.** Real authentication for writes (confirmations, primary declarations); reads can stay
   open.
4. **A model.** A local model through `openai-compatible`, or Claude. Record a sample of its judgements
   and review them before trusting the thresholds.
5. **An answer key of your own.** A few dozen hand-checked applications are enough for `itx verify` to
   measure rule precision on your data instead of the seed's.
6. **Scale.** The whole-model rebuild is fine for thousands of applications. Beyond that, resolve
   incrementally per changed observation, and add history (validity intervals on edges).
7. **A workflow engine,** if workflows start waiting on people or events for days or fan out widely
   (ADR 0003).
