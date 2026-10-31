import Link from "next/link";
import { notFound } from "next/navigation";
import { BasisPill } from "@/components/Basis";
import { api, entityHref, type Explanation } from "@/lib/api";

export const dynamic = "force-dynamic";

export default async function Explain({ searchParams }: { searchParams: Promise<{ edge?: string }> }) {
  const { edge } = await searchParams;
  if (!edge) notFound();
  const x = await api<Explanation>(`/edges/${edge}/explain`).catch(() => null);
  if (!x) notFound();
  const [from, rest] = x.fact.split(" - ");
  const [kind, to] = rest.split(" -> ");
  return (
    <>
      <p className="muted mono">{x.edge}</p>
      <h1>Why does the model think this?</h1>
      <p style={{ fontSize: "1.1rem" }}>
        <Link href={entityHref(from)} className="mono">{from}</Link> <strong>{kind}</strong> <Link href={entityHref(to)} className="mono">{to}</Link>
      </p>
      <p>
        <BasisPill basis={x.basis} confidence={x.confidence} status={x.status} /> source <code>{x.source}</code>
        {x.confirmed_by && <> · confirmed by <strong>{x.confirmed_by}</strong> on {x.confirmed_at?.slice(0, 10)}</>}
      </p>
      {x.rule && (
        <>
          <h2>Derived by rule <code>{x.rule}</code></h2>
          <ol className="path">{x.chain?.map((c) => <li key={c}>{c}</li>)}</ol>
        </>
      )}
      {x.judgement && (
        <>
          <h2>Proposed by a model</h2>
          <p className="muted">candidate <code>{x.judgement.candidate}</code>, {x.judgement.provider}/{x.judgement.model}, prompt {x.judgement.prompt_version}</p>
          <ul>{x.judgement.reasons.map((r) => <li key={r}>{r}</li>)}</ul>
        </>
      )}
      {x.observations && x.observations.length > 0 && (
        <>
          <h2>What the sources said</h2>
          <table>
            <thead><tr><th>source</th><th>data set</th><th>record</th><th>observed</th></tr></thead>
            <tbody>
              {x.observations.map((o) => (
                <tr key={o.id}><td>{o.source}</td><td>{o.dataset}</td><td className="mono">{o.external_id}</td><td>{o.observed_at.slice(0, 10)}</td></tr>
              ))}
            </tbody>
          </table>
        </>
      )}
    </>
  );
}
