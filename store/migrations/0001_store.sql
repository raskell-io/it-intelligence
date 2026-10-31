-- The store holds five things: observations, entities, edges, scorecards and artifacts, plus the small
-- tables those need to be correct. The vocabulary (entity_kind, relation_kind and their domain and
-- range) is generated from the ontology in store/vocabulary.sql and applied before this file.

-- ---------------------------------------------------------------- observations
-- Append-only: what a source said. A new row is written only when a record's content changed; the
-- model can always be rebuilt from these rows plus the decision log.
create table observation (
  id            bigserial primary key,
  source        text not null,
  dataset       text not null,
  external_id   text not null,
  payload       jsonb not null,
  content_hash  text not null,
  observed_at   timestamptz not null,
  first_run     bigint not null,
  last_seen_run bigint not null            -- the last extract that still returned this content
);
create index observation_key on observation (source, dataset, external_id, id desc);
create unique index observation_content on observation (source, dataset, external_id, content_hash);

-- Every data set a source provides, with its standing. Secondary: the source wins. Primary: the Context
-- Model is authoritative, an explicit decision with an owner and a date.
create table dataset (
  source          text not null,
  dataset         text not null,
  standing        text not null default 'secondary' check (standing in ('secondary', 'primary')),
  owner           text,
  declared_by     text,
  declared_at     timestamptz,
  expected_cycle  interval not null default '180 days',
  last_run        bigint,
  last_extract_at timestamptz,
  records         integer not null default 0,
  primary key (source, dataset)
);

-- ---------------------------------------------------------------- entities
create table entity (
  id          text primary key,              -- 'ctx:Application/policy-admin'
  kind        text not null references entity_kind(kind),
  name        text not null,
  standing    text not null default 'secondary',
  attributes  jsonb not null default '{}',
  sources     text[] not null default '{}',
  last_seen   timestamptz
);
create index entity_kind_name on entity (kind, name);
create index entity_lower_name on entity (lower(name));

-- Every id an entity is known by in a source. Identity links found by inference carry a basis, a
-- confidence and a status like edges do.
create table alias (
  id            text primary key,
  entity_id     text not null references entity(id) on delete cascade,
  source        text not null,
  dataset       text not null,
  external_id   text not null,
  basis         text not null check (basis in ('asserted', 'derived', 'inferred')),
  confidence    double precision not null,
  status        text not null check (status in ('active', 'pending', 'rejected')),
  evidence      jsonb not null default '{}',
  confirmed_by  text,
  confirmed_at  timestamptz,
  last_seen     timestamptz
);
create index alias_lookup on alias (source, dataset, external_id);
create index alias_entity on alias (entity_id);

-- What became of each latest observation: its entity and whether it resolved (for the scorecards).
create table record_status (
  observation_id bigint primary key references observation(id),
  entity_id      text,
  resolved       boolean not null
);

-- ---------------------------------------------------------------- edges
-- One row per relationship per source. Several sources asserting the same link are several rows; that
-- is what the agreement score counts. Ids are stable across rebuilds.
create table edge (
  id            text primary key,
  kind          text not null references relation_kind(kind),
  from_id       text not null references entity(id) on delete cascade,
  to_id         text not null references entity(id) on delete cascade,
  basis         text not null check (basis in ('asserted', 'derived', 'inferred')),
  confidence    double precision not null,
  status        text not null check (status in ('active', 'pending', 'rejected')),
  source        text not null,
  evidence      jsonb not null default '{}',
  confirmed_by  text,
  confirmed_at  timestamptz,
  last_seen     timestamptz,
  unique (kind, from_id, to_id, source)
);
create index edge_from on edge (from_id, kind);
create index edge_to on edge (to_id, kind);
create trigger edge_vocabulary before insert or update on edge
  for each row execute function edge_check_vocabulary();

-- Every confirmation, rejection and primary declaration, append-only. The model is rebuilt from
-- observations plus this log.
create table decision_log (
  id            bigserial primary key,
  target_type   text not null check (target_type in ('edge', 'alias', 'dataset')),
  target_id     text not null,
  action        text not null check (action in ('confirm', 'reject', 'request_primary', 'approve_primary')),
  actor         text not null,
  note          text,
  evidence_hash text,
  at            timestamptz not null default now()
);
create index decision_target on decision_log (target_id, id);

-- Model judgements, cached so a rebuild never calls a model.
create table judgement (
  id             bigserial primary key,
  candidate      text not null,
  input_hash     text not null,
  provider       text not null,
  model          text not null,
  prompt_version text not null,
  result         jsonb not null,
  created_at     timestamptz not null default now()
);
create index judgement_candidate on judgement (candidate, id desc);

-- What the resolver could not use: dangling references, judgements below the threshold.
create table dropped (
  build_id bigint not null,
  reason   text not null,
  what     text not null
);

-- ---------------------------------------------------------------- scorecards
create table scorecard (
  id            bigserial primary key,
  source        text not null,
  dataset       text not null,
  run_id        bigint,
  records       integer not null,
  freshness     double precision,
  completeness  double precision,
  agreement     double precision,
  resolvability double precision,
  overall       double precision not null,
  failing       jsonb not null default '{}',   -- signal -> external ids of the records that fail it
  computed_at   timestamptz not null
);
create index scorecard_latest on scorecard (source, dataset, id desc);

-- ---------------------------------------------------------------- artifacts
create table snapshot (
  id          text primary key,              -- content hash prefix
  created_at  timestamptz not null,
  entities    integer not null,
  edges       integer not null,
  path        text not null
);

create table artifact (
  id          bigserial primary key,
  kind        text not null,                 -- diagram | insight | draft
  name        text not null,
  subject     text,                          -- entity id, when the artifact is about one
  snapshot_id text not null references snapshot(id),
  format      text not null,                 -- mermaid | markdown | json
  body        text not null,
  meta        jsonb not null default '{}',
  created_at  timestamptz not null
);
create index artifact_lookup on artifact (kind, subject, id desc);

create table finding (
  id          bigserial primary key,
  snapshot_id text not null references snapshot(id),
  insight     text not null,
  severity    text not null,
  entities    text[] not null,
  message     text not null
);
create index finding_snapshot on finding (snapshot_id, insight);

-- ---------------------------------------------------------------- workflow runs
-- State of the workflow runner (workflows/runner). A run is a list of steps; each step is idempotent
-- and records every attempt.
create table run (
  id           bigserial primary key,
  workflow     text not null,
  input        jsonb not null default '{}',
  status       text not null default 'queued' check (status in ('queued', 'running', 'succeeded', 'failed')),
  actor        text,
  claimed_by   text,
  heartbeat_at timestamptz,
  not_before   timestamptz not null default now(),
  created_at   timestamptz not null default now(),
  started_at   timestamptz,
  finished_at  timestamptz,
  error        text
);
create index run_queue on run (status, not_before, id);

create table run_step (
  run_id      bigint not null references run(id) on delete cascade,
  step        text not null,
  position    integer not null,
  attempt     integer not null,
  status      text not null check (status in ('running', 'succeeded', 'failed')),
  error       text,
  output      jsonb not null default '{}',
  started_at  timestamptz not null,
  finished_at timestamptz,
  primary key (run_id, step, attempt)
);
