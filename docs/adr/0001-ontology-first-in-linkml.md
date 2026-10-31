# 0001: The ontology comes first, in LinkML

**Status:** accepted, 2026-10-31

**Context.** Every part of the platform names entity kinds and relation kinds: the store, the mappers, the
resolver, the API, the MCP tools, the frontend. If each declares them on its own, they drift.

**Decision.** One LinkML schema (`ontology/`) declares them. `make ontology` generates JSON Schema, OWL,
SHACL, TypeScript, docs, and, through a small generator of our own (`ontology/gen/vocabulary.py`), the store
vocabulary, a Go package and TypeScript union types. CI fails when a generated file is stale.

Relations are LinkML slots. LinkML allows one `domain` per slot, so the full domain and range live in
`ctx_domain` / `ctx_range` annotations, which the vocabulary generator reads.

**Consequences.** An ontology change is one pull request that touches every layer, and the compilers show
where. LinkML's SQL generator is not used: it creates a table per class, and the store has one entity table
and one edge table, with kinds as data (ADR 0002). LinkML 1.12's TypeScript generator emits an undefined
`date` type; the Makefile rewrites it to `string`.
