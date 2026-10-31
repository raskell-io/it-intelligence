import Link from "next/link";
import { api, type Run } from "@/lib/api";

export const dynamic = "force-dynamic";

export default async function Runs({ searchParams }: { searchParams: Promise<{ id?: string }> }) {
  const { id } = await searchParams;
  if (id) {
    const r = await api<Run>(`/runs/${id}`);
    return (
      <>
        <p className="muted"><Link href="/runs">runs</Link></p>
        <h1>Run {r.id}: {r.workflow}</h1>
        <p><span className={`pill ${r.status === "failed" ? "error" : r.status === "succeeded" ? "asserted" : "pending"}`}>{r.status}</span> <code>{JSON.stringify(r.input)}</code> {r.actor && <>by {r.actor}</>}</p>
        {r.error && <pre>{r.error}</pre>}
        <table>
          <thead><tr><th>step</th><th>attempt</th><th>status</th><th>took</th><th>output</th></tr></thead>
          <tbody>
            {r.steps?.map((s) => (
              <tr key={`${s.step}-${s.attempt}`}>
                <td>{s.step}</td><td>{s.attempt}</td><td>{s.status}</td><td className="mono">{s.duration_ms ?? ""} ms</td>
                <td className="mono" style={{ fontSize: ".78rem" }}>{s.error ?? JSON.stringify(s.output)}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </>
    );
  }
  const { runs } = await api<{ runs: Run[] }>("/runs");
  return (
    <>
      <h1>Workflow runs</h1>
      <p className="muted">Every run with its steps, attempts and errors. When a scorecard drops, the first question is what happened during the last ingest; the answer is here.</p>
      <table>
        <tbody>
          {runs.map((r) => (
            <tr key={r.id}>
              <td><Link href={`/runs?id=${r.id}`}>{r.id}</Link></td>
              <td>{r.workflow}</td>
              <td className="mono">{JSON.stringify(r.input)}</td>
              <td>{r.status}</td>
              <td className="muted">{r.actor}</td>
              <td className="muted">{r.created_at.slice(0, 19).replace("T", " ")}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </>
  );
}
