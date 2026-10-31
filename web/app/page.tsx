import Link from "next/link";
import { api, entityHref, shortId, type EntitySummary, type Finding, type Stats } from "@/lib/api";
import { KIND_DESCRIPTIONS, KINDS } from "@/lib/generated/vocabulary";

export const dynamic = "force-dynamic";

export default async function Home({ searchParams }: { searchParams: Promise<{ q?: string; kind?: string }> }) {
  const { q = "", kind = "" } = await searchParams;
  const [stats, { findings }] = await Promise.all([api<Stats>("/stats"), api<{ findings: Finding[] }>("/findings")]);
  const hits = q || kind ? (await api<{ entities: EntitySummary[] }>(`/entities?q=${encodeURIComponent(q)}&kind=${kind}&limit=50`)).entities : [];
  const order = { error: 0, warning: 1, info: 2 };
  findings.sort((a, b) => order[a.severity] - order[b.severity] || a.insight.localeCompare(b.insight));
  const edges = Object.values(stats.edges).reduce((a, b) => a + b, 0);

  return (
    <>
      <h1>The Kestrel Group</h1>
      <p className="muted">
        A fictional financial group, as its IT tools describe it: {Object.values(stats.entities).reduce((a, b) => a + b, 0)} entities and {edges} relationships,
        each with a basis, a confidence and a source. {stats.pending_review > 0 && <Link href="/review">{stats.pending_review} inferred {stats.pending_review === 1 ? "link waits" : "links wait"} for review.</Link>}
      </p>

      <form className="search" action="/">
        <input name="q" defaultValue={q} placeholder="Search: policy, claims-prod, pol-adm-ui, Payments" style={{ flex: 1 }} />
        <select name="kind" defaultValue={kind}>
          <option value="">all kinds</option>
          {KINDS.map((k) => (
            <option key={k} value={k}>{k}</option>
          ))}
        </select>
        <button>Search</button>
      </form>

      {(q || kind) && (
        <table>
          <tbody>
            {hits.map((h) => (
              <tr key={h.id}>
                <td><span className="pill">{h.kind}</span></td>
                <td><Link href={entityHref(h.id)}>{h.name}</Link> <span className="muted mono">{shortId(h.id)}</span></td>
                <td className="muted">{h.summary}</td>
              </tr>
            ))}
            {hits.length === 0 && <tr><td className="muted">Nothing matches.</td></tr>}
          </tbody>
        </table>
      )}

      <h2>What the model found</h2>
      <p className="muted">Insights are named queries run on every snapshot. Each finding links to the entities involved: a starting point for a conversation, not a report to file.</p>
      <table>
        <tbody>
          {findings.map((f, i) => (
            <tr key={i}>
              <td><span className={`pill ${f.severity}`}>{f.severity}</span></td>
              <td className="mono">{f.insight}</td>
              <td>
                {f.message}
                <div>
                  {f.entities.map((e) => (
                    <Link key={e} href={entityHref(e)} className="mono" style={{ marginRight: ".8rem" }}>{shortId(e)}</Link>
                  ))}
                </div>
              </td>
            </tr>
          ))}
        </tbody>
      </table>

      <h2>Entity kinds</h2>
      <div className="grid">
        {KINDS.map((k) => (
          <Link key={k} href={`/?kind=${k}`} className="card" style={{ color: "inherit" }}>
            <strong>{k}</strong> <span className="muted">{stats.entities[k] ?? 0}</span>
            <div className="muted" style={{ fontSize: ".85rem" }}>{KIND_DESCRIPTIONS[k]}</div>
          </Link>
        ))}
      </div>
    </>
  );
}
