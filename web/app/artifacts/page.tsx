import Link from "next/link";
import { Mermaid } from "@/components/Mermaid";
import { api, entityHref, type Artifact } from "@/lib/api";

export const dynamic = "force-dynamic";

export default async function Artifacts({ searchParams }: { searchParams: Promise<{ id?: string }> }) {
  const { id } = await searchParams;
  const { artifacts } = await api<{ artifacts: Artifact[] }>("/artifacts?body=true");
  const one = id ? artifacts.find((a) => String(a.id) === id) : undefined;
  if (one) {
    return (
      <>
        <p className="muted">{one.kind} · snapshot <code>{one.snapshot}</code> · {one.created_at.slice(0, 10)}</p>
        <h1>{one.name}</h1>
        {one.subject && <p><Link href={entityHref(one.subject)}>{one.subject}</Link></p>}
        {one.format === "mermaid" ? <Mermaid text={one.body ?? ""} /> : null}
        <pre>{one.body}</pre>
      </>
    );
  }
  const insights = artifacts.filter((a) => a.kind === "insight");
  const diagrams = artifacts.filter((a) => a.kind === "diagram");
  return (
    <>
      <h1>Artifacts</h1>
      <p className="muted">Built by BuildArtifacts from snapshot <code>{artifacts[0]?.snapshot}</code>. Each artifact records the snapshot it came from, so it can be reproduced.</p>
      <h2>Insights</h2>
      <table>
        <tbody>
          {insights.map((a) => (
            <tr key={a.id}>
              <td><Link href={`/artifacts?id=${a.id}`} className="mono">{a.name}</Link></td>
              <td className="muted">{String(a.meta?.description ?? "")}</td>
              <td><span className={`pill ${String(a.meta?.severity)}`}>{String(a.meta?.findings)} found</span></td>
            </tr>
          ))}
        </tbody>
      </table>
      <h2>Diagrams</h2>
      <ul>{diagrams.map((a) => <li key={a.id}><Link href={`/artifacts?id=${a.id}`}>{a.name}</Link></li>)}</ul>
    </>
  );
}
