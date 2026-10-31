import Link from "next/link";
import { ReviewActions } from "@/components/ReviewActions";
import { api, entityHref, shortId, type EntitySummary, type ReviewItem } from "@/lib/api";

export const dynamic = "force-dynamic";

export default async function Review({ searchParams }: { searchParams: Promise<{ team?: string }> }) {
  const { team = "" } = await searchParams;
  const [{ items }, { entities: teams }] = await Promise.all([
    api<{ items: ReviewItem[] }>(`/review?team=${encodeURIComponent(team)}`),
    api<{ entities: EntitySummary[] }>("/entities?kind=Team&limit=50"),
  ]);
  return (
    <>
      <h1>Review queue</h1>
      <p className="muted">
        Links the resolver inferred with a confidence between 0.6 and 0.9, waiting for the team that owns the application. A confirmation makes the link
        asserted, with your name on it; a rejection sticks until the evidence changes.
      </p>
      <form className="search">
        <select name="team" defaultValue={team}>
          <option value="">every team</option>
          {teams.map((t) => <option key={t.id} value={shortId(t.id)}>{t.name}</option>)}
        </select>
        <button>Show</button>
      </form>
      {items.length === 0 && <p className="muted">Nothing waits for review.</p>}
      {items.map((it) => (
        <div className="card" key={it.id} style={{ marginBottom: "1rem" }}>
          <div>
            <Link href={entityHref(it.from)}>{it.from_name}</Link> <strong className="mono">{it.relation}</strong>{" "}
            {it.type === "edge" ? <Link href={entityHref(it.to)}>{it.to_name}</Link> : <span>{it.to_name}</span>}{" "}
            <span className="pill pending">{it.confidence.toFixed(2)}</span>{" "}
            <span className="muted">for {it.team ? shortId(it.team) : "no owning team"}</span>
          </div>
          <ul>{it.reasons.map((r) => <li key={r}>{r}</li>)}</ul>
          {it.previously_rejected && (
            <p className="muted">Rejected before by {it.previously_rejected.by}; new evidence brought it back.</p>
          )}
          <ReviewActions type={it.type} id={it.id} />{" "}
          {it.type === "edge" && <Link href={`/explain?edge=${it.id}`} className="muted">evidence</Link>}
        </div>
      ))}
    </>
  );
}
