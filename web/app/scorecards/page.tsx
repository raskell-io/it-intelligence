import { api, pct, type Scorecard } from "@/lib/api";

export const dynamic = "force-dynamic";

function Signal({ v }: { v: number | null }) {
  if (v === null) return <span className="muted">n/a</span>;
  return (
    <span title={pct(v)} style={{ display: "flex", alignItems: "center", gap: ".4rem" }}>
      <span className="bar" style={{ flex: 1 }}><span style={{ width: `${v * 100}%` }} /></span>
      <span className="mono">{pct(v)}</span>
    </span>
  );
}

export default async function Scorecards() {
  const { scorecards } = await api<{ scorecards: Scorecard[] }>("/scorecards?failing=true");
  return (
    <>
      <h1>Source scorecards</h1>
      <p className="muted">
        Every data set from every source, worst first. The score is computed, not declared, and every number links to the records behind it. New
        relationships start with a confidence that reflects their source&apos;s score.
      </p>
      <table>
        <thead>
          <tr><th>data set</th><th>records</th><th>freshness</th><th>completeness</th><th>agreement</th><th>resolvability</th><th>overall</th></tr>
        </thead>
        <tbody>
          {scorecards.map((s) => (
            <tr key={`${s.source}/${s.dataset}`}>
              <td>
                <strong>{s.source}</strong>/{s.dataset} {s.standing === "primary" && <span className="pill primary">primary</span>}
                {s.failing && Object.keys(s.failing).length > 0 && (
                  <details>
                    <summary className="muted">failing records</summary>
                    {Object.entries(s.failing).map(([signal, ids]) => (
                      <div key={signal}><span className="muted">{signal}:</span> <span className="mono">{ids.slice(0, 20).join(", ")}{ids.length > 20 ? ` and ${ids.length - 20} more` : ""}</span></div>
                    ))}
                  </details>
                )}
              </td>
              <td>{s.records}</td>
              <td><Signal v={s.freshness} /></td>
              <td><Signal v={s.completeness} /></td>
              <td><Signal v={s.agreement} /></td>
              <td><Signal v={s.resolvability} /></td>
              <td><strong className="mono">{pct(s.overall)}</strong></td>
            </tr>
          ))}
        </tbody>
      </table>
    </>
  );
}
