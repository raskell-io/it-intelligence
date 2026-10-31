// Types of the REST API (api/openapi.yaml) and a fetch helper for server components.
// Entity kinds and relation kinds come from the ontology: when it changes, this file stops compiling
// until the frontend catches up.
import type { Kind, Relation } from "./generated/vocabulary";
import type { Basis, Standing } from "./generated/context_model";

export type { Kind, Relation };
export type BasisName = `${Basis}`;
export type StandingName = `${Standing}`;
export type Status = "active" | "pending" | "rejected";

export interface EntitySummary {
  id: string;
  kind: Kind;
  name: string;
  summary?: string;
}

export interface EdgeRef {
  edge: string;
  kind: Relation;
  direction: "out" | "in";
  other: string;
  other_name: string;
  basis: BasisName;
  confidence: number;
  status: Status;
  source: string;
}

export interface Entity {
  id: string;
  kind: Kind;
  name: string;
  standing: StandingName;
  attributes: Record<string, unknown>;
  sources: string[];
  last_seen?: string;
  edges: EdgeRef[];
}

export interface Owner {
  entity: string;
  owner?: string;
  owner_name?: string;
  basis?: BasisName;
  confidence?: number;
  edge?: string;
  source?: string;
  path?: string[];
  last_seen?: string;
  note?: string;
}

export interface Explanation {
  edge: string;
  fact: string;
  basis: BasisName;
  confidence: number;
  status: Status;
  source: string;
  rule?: string;
  chain?: string[];
  observations?: { id: number; source: string; dataset: string; external_id: string; observed_at: string }[];
  judgement?: { match: string; confidence: number; reasons: string[]; candidate: string; provider: string; model: string; prompt_version: string };
  confirmed_by?: string;
  confirmed_at?: string;
  previously_rejected?: unknown;
}

export interface ReviewItem {
  type: "edge" | "alias";
  id: string;
  relation: Relation | "same_as";
  from: string;
  from_name: string;
  to: string;
  to_name: string;
  confidence: number;
  reasons: string[];
  team: string;
  previously_rejected?: { by: string; at: string; note?: string };
}

export interface Scorecard {
  source: string;
  dataset: string;
  standing: StandingName;
  records: number;
  freshness: number | null;
  completeness: number | null;
  agreement: number | null;
  resolvability: number | null;
  overall: number;
  computed_at: string;
  failing?: Record<string, string[]>;
}

export interface Finding {
  insight: string;
  severity: "error" | "warning" | "info";
  entities: string[];
  message: string;
  snapshot: string;
}

export interface Artifact {
  id: number;
  kind: "diagram" | "insight" | "draft";
  name: string;
  subject?: string;
  snapshot: string;
  format: string;
  body?: string;
  meta?: Record<string, unknown>;
  created_at: string;
}

export interface Run {
  id: number;
  workflow: string;
  input: unknown;
  status: "queued" | "running" | "succeeded" | "failed";
  actor?: string;
  created_at: string;
  finished_at?: string;
  error?: string;
  steps?: { step: string; attempt: number; status: string; error?: string; duration_ms?: number; output?: unknown }[];
}

export interface Person {
  id: string;
  name: string;
  team: string;
}

export interface Stats {
  entities: Record<string, number>;
  edges: Record<string, number>;
  pending_review: number;
  now?: string;
}

const base = process.env.ITX_API ?? "http://localhost:8080";

/** Fetch from itx on the server. Never cached: the model changes with every ingest. */
export async function api<T>(path: string): Promise<T> {
  const res = await fetch(`${base}/api/v1${path}`, { cache: "no-store" });
  if (!res.ok) {
    const body = await res.text();
    throw new Error(`${res.status} ${path}: ${body}`);
  }
  return res.json() as Promise<T>;
}

export function entityHref(id: string): string {
  return `/entity?id=${encodeURIComponent(id)}`;
}

export function kindOf(id: string): string {
  return id.replace(/^ctx:/, "").split("/")[0];
}

export function shortId(id: string): string {
  return id.replace(/^ctx:[A-Za-z]+\//, "");
}

export function pct(v: number | null | undefined): string {
  return v === null || v === undefined ? "n/a" : `${Math.round(v * 100)}%`;
}
