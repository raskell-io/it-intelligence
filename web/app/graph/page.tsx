import Link from "next/link";
import { Graph } from "@/components/Graph";
import { entityHref } from "@/lib/api";

export default async function GraphPage({ searchParams }: { searchParams: Promise<{ id?: string; depth?: string }> }) {
  const { id = "ctx:Application/policy-admin", depth = "1" } = await searchParams;
  const d = Math.min(Math.max(parseInt(depth, 10) || 1, 1), 3);
  return (
    <>
      <h1>Graph</h1>
      <p>
        <Link href={entityHref(id)} className="mono">{id}</Link>, depth{" "}
        {[1, 2, 3].map((n) => <Link key={n} href={`/graph?id=${encodeURIComponent(id)}&depth=${n}`} style={{ marginRight: ".4rem", fontWeight: n === d ? 700 : 400 }}>{n}</Link>)}
      </p>
      <Graph id={id} depth={d} />
    </>
  );
}
