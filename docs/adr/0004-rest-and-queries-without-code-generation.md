# 0004: REST and queries without code generators

**Status:** accepted, 2026-10-31

**Context.** The brief proposed oapi-codegen for the REST server and sqlc for queries.

**Decision.** Handlers are written against the standard library's `net/http` mux; queries are plain SQL in
`api/query` over pgx. The contract is `api/openapi.yaml`, written by hand.

**Consequences.** Two fewer tools for a reader to learn, and the query layer reads top to bottom. The cost is
that the OpenAPI file can drift from the handlers; `api/parity_test.go` keeps REST and MCP in step, and a
contract test against the OpenAPI file is a reasonable next step.
