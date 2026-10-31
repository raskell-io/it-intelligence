# it-intelligence. Every target runs through mise so the pinned tool versions are used.
SHELL := /bin/bash
MISE  := mise exec --
# The LinkML version every generator runs with. Change it on purpose, in its own commit, and regenerate.
LINKML ?= linkml==1.12.1
LINKML_RUN := $(MISE) uvx --from $(LINKML)
SCHEMA := ontology/context_model.yaml
GEN    := ontology/generated
COMPOSE := docker compose -f deploy/compose.yaml

.PHONY: help ontology ontology-check build test lint up down seed ingest verify demo seed-gen dev-db dev-ingest dev-serve dev-web dev-verify clean

help: ## list targets
	@grep -E '^[a-z-]+:.*## ' $(MAKEFILE_LIST) | awk -F':.*## ' '{printf "  %-16s %s\n", $$1, $$2}'

ontology: ## regenerate everything in ontology/generated from the LinkML schema
	@mkdir -p $(GEN)/jsonschema $(GEN)/ts $(GEN)/owl $(GEN)/shacl $(GEN)/docs
	$(LINKML_RUN) gen-json-schema $(SCHEMA) > $(GEN)/jsonschema/context_model.schema.json
	$(LINKML_RUN) gen-typescript $(SCHEMA) | sed -E 's/: date([,;])/: string\1/' > $(GEN)/ts/context_model.ts  # LinkML 1.12 emits an undefined 'date' type
	$(LINKML_RUN) gen-owl $(SCHEMA) > $(GEN)/owl/context_model.owl.ttl
	$(LINKML_RUN) gen-shacl $(SCHEMA) > $(GEN)/shacl/context_model.shacl.ttl
	$(LINKML_RUN) gen-doc -d $(GEN)/docs $(SCHEMA)
	$(LINKML_RUN) python ontology/gen/vocabulary.py
	cp $(GEN)/sql/vocabulary.sql store/vocabulary.sql
	mkdir -p web/lib/generated && cp $(GEN)/ts/*.ts web/lib/generated/
	$(MISE) gofmt -w $(GEN)/go

ontology-check: ontology ## fail if generated files differ from what is committed
	git diff --exit-code -- $(GEN) store/vocabulary.sql web/lib/generated ':!$(GEN)/docs'

build: ## build the itx binary into bin/
	$(MISE) go build -o bin/itx ./cmd/itx

test: ## unit tests (mappers, resolver, runner, query layer)
	$(MISE) go test ./...

lint: ## go vet
	$(MISE) go vet ./...

up: ## start PostgreSQL, itx and web with docker compose
	$(COMPOSE) up -d --build

down: ## stop and remove the compose stack and its data
	$(COMPOSE) down -v

seed: ## register the Kestrel Group seed sources and data sets
	$(COMPOSE) exec itx itx seed

ingest: ## ingest every seeded source, rebuild the model, score, build artifacts
	$(COMPOSE) exec itx itx ingest

verify: ## check the resolver against the answer key and the article's claims against the running system
	$(COMPOSE) exec itx itx verify

demo: up seed ingest verify ## the whole thing, from a fresh clone

seed-gen: ## regenerate the Kestrel seed exports and the answer key from seed/gen
	$(MISE) go run ./seed/gen

# ---- without Docker for itx and web: PostgreSQL in a container, everything else on the host ----
DEV_DB := postgres://itx:itx@localhost:55432/itx

dev-db: ## start a PostgreSQL container on port 55432 for local development
	docker start itx-pg 2>/dev/null || docker run -d --name itx-pg -e POSTGRES_USER=itx -e POSTGRES_PASSWORD=itx -e POSTGRES_DB=itx -p 55432:5432 postgres:17-alpine

dev-ingest: build ## ingest the seed into the local database, in this process
	ITX_DATABASE_URL=$(DEV_DB) ./bin/itx ingest -inline

dev-serve: build ## serve REST and MCP on :8080 against the local database
	ITX_DATABASE_URL=$(DEV_DB) ./bin/itx serve

dev-web: ## the web UI on :3000 against itx on :8080
	cd web && $(MISE) npm install --no-audit --no-fund && ITX_API=http://localhost:8080 $(MISE) npx next dev -p 3000

dev-verify: build ## verify against the local database
	ITX_DATABASE_URL=$(DEV_DB) ./bin/itx verify

clean:
	rm -rf bin var
