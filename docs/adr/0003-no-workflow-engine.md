# 0003: No workflow engine; a small runner over PostgreSQL

**Status:** accepted, 2026-10-31 (supersedes the brief's Temporal design)

**Context.** Ingestion talks to flaky sources and calls models that time out, so it needs per-step retries,
resumption after a crash, schedules, and a place to see what happened. A durable execution engine (Temporal,
Restate) provides all of that, and also a server, a concept set and an operations burden a reader of a
reference must learn first.

**Decision.** `workflows/runner` (about 300 lines): a workflow is a fixed list of idempotent steps; runs and
step attempts live in two tables; runs are claimed with `FOR UPDATE SKIP LOCKED` and a heartbeat; a run
resumes at the first step that did not succeed; retries back off exponentially with jitter; schedules are
intervals. Steps pass only the run id; whatever they produce goes to PostgreSQL. `/runs` shows every run.

**Consequences.** Nothing to install beyond PostgreSQL, and the whole mechanism fits on a screen. What it
cannot do: wait for a person or an event for days inside a workflow (ConfirmLink is a separate workflow
started by the click), fan out to thousands of parallel activities, or run arbitrary code paths durably.
When those are needed, move to Temporal or Restate: the steps already have the shape of activities or
handlers.
