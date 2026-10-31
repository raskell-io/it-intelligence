# 0002: A generic store, and our own generator for its vocabulary and the Go types

**Status:** accepted, 2026-10-31

**Context.** The brief proposed Go types generated from JSON Schema with go-jsonschema, or LinkML's Go
generator. Entity attributes live in a `jsonb` column; what Go code needs from the ontology is the names of
kinds and relations, their domain and range, and the attributes of each kind.

**Decision.** `ontology/gen/vocabulary.py` (about 250 lines, standard library plus linkml-runtime) writes
`vocabulary.sql` (kind and relation tables, and a trigger that rejects an edge outside its relation's domain
or range), `vocab.go` (constants and descriptors) and `vocabulary.ts` (union types).

**Consequences.** One generator, three outputs, no extra Go tooling. Removing a relation from the ontology
fails the Go build and the TypeScript build wherever it is still used, and the database refuses edges that
the ontology does not allow.
