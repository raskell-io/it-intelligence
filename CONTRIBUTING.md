# Contributing

Thank you for taking it apart. The most useful contribution is a report that the model got something
wrong on your own exports: which source said what, what the model concluded, and what is true. Open an
issue with the `explain_fact` output of the edge in question.

Code contributions follow `AGENTS.md`. In short: the ontology comes first and is generated from; connectors
only read; mappers carry company conventions; every relationship keeps its basis, confidence and source;
REST and MCP share one query layer; `make test` and `itx verify` pass.

A new source is three files and a line of configuration: see `docs/adding-a-source.md`.

By contributing you agree that your contribution is licensed under Apache-2.0 OR MIT, like the rest.
