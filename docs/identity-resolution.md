# Identity resolution

The same application has a different name in every system. The EA repository says "Policy
Administration", the git host has `pol-adm-svc` and `pol-adm-ui`, the cluster runs `policy-admin` in
`claims-prod`, and the ITSM tool has a business service called "PAS". Resolution decides which of these
are one thing, and how sure it is. It is most of the work.

## Basis: how a fact was established

Every relationship (`edge`) and every identity link (`alias`) carries a basis:

| Basis | Meaning | Confidence | Status |
|---|---|---|---|
| **asserted** | straight from a source's own key, or confirmed by a person | `0.7 + 0.3 x` the source's last overall score; 1.0 once a person confirms | active |
| **derived** | follows from other facts by a named rule | the rule's precision x the weakest fact it rests on | active |
| **inferred** | proposed by a model from names, descriptions, owners and labels | the model's own confidence | active at 0.9 or above, pending review from 0.6, dropped below |

## The passes

```text
 assertions from every latest observation
        │
        ▼
 1. keys          natural keys and shared ids  ─────────────▶ asserted
        │         (an EA id in a deployment label, a catalog-info key in a repository)
        ▼
 2. rules         named heuristics with a measured precision ─▶ derived
        │
        ▼
 3. inference     a shortlist per leftover candidate, a recorded model judgement ─▶ inferred
        │
        ▼
 4. decisions     confirmations and rejections from the decision log
        │
        ▼
 rules again      facts that follow from inferred links (an endpoint of a newly linked workload)
```

### 1. Keys

A mapper says what a record is and what it points at, by natural key (`ctx:Repository/kestrel/pol-adm-svc`)
or by an alias in another source (`earepo:applications/APP-0142`). Entities with the same natural key are
one entity; a record that carries another source's id joins that entity. The cheapest win in the whole
system is a shared key: a deployment label carrying the EA application id turns an inference into an
assertion, and every team that adds one makes the model more certain.

### 2. Rules

Rules fill gaps; they never override what a source asserted. Each has a name, a precision in
`config/resolver.yaml`, and evidence that lists the edges it rests on, so `explain_fact` can show the
chain.

| Rule | Derives | Precision (configured) | Measured on the seed |
|---|---|---|---|
| `realised_via_build` | app `realised_by` repo, from `deployed_as` + `built_from` | 0.95 | 3 of 3 |
| `same_normalised_name` | app `realised_by` repo, app `deployed_as` workload, when the names match after stripping `-svc` | 0.9 | 1 of 1 |
| `team_owns_single_app` | app `realised_by` repo, when the repo's team owns exactly one application | 0.7 | 0 of 1 |
| `owner_via_workloads` | namespace `owned_by` team, when every application in it has the same owner | 0.95 | 1 of 1 |
| `owner_via_namespace` | workload `owned_by` the owner of its namespace | 0.9 | 40 of 40 |
| `app_owner_via_namespace` | app `owned_by` team, when all its workloads run in namespaces of one team | 0.75 | 2 of 2 |
| `exposed_via_route` | app `exposed_via` endpoint, from `routes_to` + `deployed_as` | 0.95 | 9 of 9 |

`team_owns_single_app` is the honest exception: on the seed it links an experiments repository to the
only application its team owns, which is wrong. Its confidence (0.68) keeps it below the threshold the
insights use. `itx verify` prints these measurements on every run.

### 3. Inference

What keys and rules leave open becomes a **candidate**:

- a repository that realises no application,
- a workload no application is deployed as,
- a record that is some application, but nobody says which (an ITSM business service without an EA
  reference).

Each candidate gets a **shortlist** of at most five applications: those of the owning team first, then
the most similar names (trigram similarity). The model sees the candidate's name, README excerpt or
description, owner and namespace, and the shortlist, and answers in a fixed schema:

```json
{
  "match": "ctx:Application/policy-admin",
  "confidence": 0.82,
  "reasons": [
    "README describes the policy administration frontend",
    "same owning team as pol-adm-svc, already linked by key",
    "deploys to claims-prod next to policy-admin"
  ]
}
```

The prompt is in `resolver/provider/prompt.md`; its version goes into every inferred link's evidence.
Every judgement is stored with a hash of its inputs; a build only uses a judgement whose inputs still
match, and the Judge step asks again when they change.

### 4. Decisions

The review queue shows pending links to the team that owns the application. A **confirmation** makes the
link asserted, at confidence 1.0, with the person's name and the date on it. A **rejection** sticks: the
next build proposes the link again only if its evidence changed, and the queue then shows the earlier
rejection next to it. Both live in the append-only decision log, so the model can be rebuilt from scratch
and keep every human decision.

## Providers

| Provider | When | Notes |
|---|---|---|
| `fixture` | default, CI, the demo | replays `seed/judgements.jsonl`; deterministic and offline |
| `openai-compatible` | a local model (Ollama, llama.cpp, vLLM) or any compatible endpoint | JSON-schema response format |
| `anthropic` | Claude | structured outputs; left out of the default Docker image (`-tags noanthropic`) |

The inputs are metadata and the prompts are small, so a local model does the job. `itx resolver record
-provider anthropic` re-records the fixtures with a real model.

## Measuring it

`seed/truth.yaml` is the answer key: every true entity, alias and relationship of the Kestrel Group.
`itx verify` compares the active model to it: edge precision 0.996 and recall 0.994, identity precision
and recall 1.0 on the seed, with a baseline in `config/verify.yaml` that CI enforces. The three true
relationships it misses are the two links still waiting for review and one fact no source holds.
