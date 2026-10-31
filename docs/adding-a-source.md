# Adding a source

A source is three files and one line of configuration. The WAF connector is the smallest, so it is the
example.

1. **A connector** (`connectors/waf/waf.go`). It reads the source and emits one observation per record. It
   knows nothing about the ontology. File mode needs a `FileDataset` per data set: how to read it and which
   field is the record's id. A live mode implements `connectors.Connector` against the source's API.
2. **A mapper** (in `mapping/mappers.go`, or a file of its own). It turns an observation into assertions:
   `a.Entity(kind, key, name, attributes)` for what the record describes, `a.Edge(relation, from, to,
   mapping.Asserted)` for what it says about relationships, `mapping.Alias(...)` to point at an entity by
   another source's id. `Fields(dataset)` declares which fields the record needs (completeness) and which
   field says when it last changed (freshness).
3. **A seed export** (`seed/waf/policies.json`), produced by `seed/gen`, and a table test in
   `mapping/mappers_test.go`.
4. **One line** in `config/sources.yaml`: mode, directory, interval, expected cycle. Add the source to
   `source_order` where its names and attributes should rank.

Register the connector with a blank import in `cmd/itx/main.go`. Then `make dev-ingest`: the source gets a
scorecard, its records join the model, and `itx verify` tells you whether anything else moved.

The cheapest win in any source is a shared key: a field that carries the EA application id turns an
inference into an assertion.
