import Link from "next/link";
import { notFound } from "next/navigation";
import { BasisPill } from "@/components/Basis";
import { Mermaid } from "@/components/Mermaid";
import { api, entityHref, shortId, type Entity, type Owner, type Stats } from "@/lib/api";

export const dynamic = "force-dynamic";

interface Decision { id: string; title: string; status: string; decided_on: string; url: string; edge: string }

export default async function EntityPage({ searchParams }: { searchParams: Promise<{ id?: string }> }) {
  const { id } = await searchParams;
  if (!id) notFound();
  let e: Entity;
  try {
    e = await api<Entity>(`/entity?id=${encodeURIComponent(id)}`);
  } catch {
    notFound();
  }
  const [owner, decisions, diagram, stats] = await Promise.all([
    api<Owner>(`/owner?id=${encodeURIComponent(id)}`).catch(() => null),
    api<{ decisions: Decision[] }>(`/decisions?id=${encodeURIComponent(id)}`),
    e.kind === "Application" ? api<{ mermaid: string }>(`/diagram?id=${encodeURIComponent(id)}`) : Promise.resolve(null),
    api<Stats>("/stats"),
  ]);
  // A fact not seen for 30 days turns amber.
  const now = stats.now ? new Date(stats.now).getTime() : Date.now();
  const stale = (t?: string) => !!t && now - new Date(t).getTime() > 30 * 24 * 3600 * 1000;
  const groups = new Map<string, typeof e.edges>();
  for (const r of e.edges) {
    const key = `${r.direction === "out" ? "" : "← "}${r.kind}`;
    groups.set(key, [...(groups.get(key) ?? []), r]);
  }

  return (
    <>
      <p className="muted mono">{e.id}</p>
      <h1>{e.name}</h1>
      <p>
        <span className="pill">{e.kind}</span>{" "}
        <span className={`pill ${e.standing}`} title="secondary: mirrored from a source, the source wins; primary: the Context Model is authoritative">{e.standing}</span>{" "}
        <span className="muted">known to {e.sources.join(", ")}</span>{" "}
        {e.last_seen && <span className={`pill ${stale(e.last_seen) ? "stale" : ""}`}>last seen {e.last_seen.slice(0, 10)}</span>}{" "}
        <Link href={`/graph?id=${encodeURIComponent(e.id)}`}>graph view</Link>
      </p>

      {owner && (
        <div className="card">
          {owner.owner ? (
            <>
              <strong>Owner:</strong> <Link href={entityHref(owner.owner)}>{owner.owner_name}</Link>{" "}
              <BasisPill basis={owner.basis!} confidence={owner.confidence} />{" "}
              <Link href={`/explain?edge=${owner.edge}`} className="muted">why?</Link>
              {owner.path && owner.path.length > 1 && (
                <ol className="path">{owner.path.map((p) => <li key={p}>{p}</li>)}</ol>
              )}
            </>
          ) : (
            <span className="muted">{owner.note}</span>
          )}
        </div>
      )}

      <h2>Attributes</h2>
      <dl className="kv">
        {Object.entries(e.attributes).sort().map(([k, v]) => (
          <div key={k} style={{ display: "contents" }}>
            <dt>{k}</dt>
            <dd className="mono">{String(v)}</dd>
          </div>
        ))}
      </dl>

      <h2>Relationships</h2>
      <table>
        <tbody>
          {[...groups.entries()].map(([kind, rs]) =>
            rs.map((r, i) => (
              <tr key={r.edge}>
                <td className="mono">{i === 0 ? kind : ""}</td>
                <td><Link href={entityHref(r.other)}>{r.other_name}</Link> <span className="muted mono">{shortId(r.other)}</span></td>
                <td title={`source ${r.source}, confidence ${r.confidence.toFixed(2)}, edge ${r.edge}`}>
                  <BasisPill basis={r.basis} confidence={r.confidence} status={r.status} /> <span className="muted">{r.source}</span>
                </td>
                <td><Link href={`/explain?edge=${r.edge}`} className="muted">why?</Link></td>
              </tr>
            )),
          )}
        </tbody>
      </table>

      {decisions.decisions.length > 0 && (
        <>
          <h2>Architecture decisions</h2>
          <ul>
            {decisions.decisions.map((d) => (
              <li key={d.id}>
                <Link href={entityHref(d.id)}>{d.title}</Link> <span className={`pill ${d.status === "superseded" ? "stale" : ""}`}>{d.status}</span>{" "}
                <span className="muted">{d.decided_on}</span>
              </li>
            ))}
          </ul>
        </>
      )}

      {diagram && (
        <>
          <h2>Neighbourhood</h2>
          <p className="muted">Generated as Mermaid text; inferred links are dashed and carry their confidence.</p>
          <Mermaid text={diagram.mermaid} />
          <details><summary className="muted">Mermaid source</summary><pre>{diagram.mermaid}</pre></details>
        </>
      )}
    </>
  );
}
