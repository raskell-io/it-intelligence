"use client";
import cytoscape from "cytoscape";
import { useRouter } from "next/navigation";
import { useEffect, useRef, useState } from "react";
import type { Entity, Relation } from "@/lib/api";
import { RELATIONS } from "@/lib/generated/vocabulary";

const COLOURS: Record<string, string> = {
  Application: "#2f5d8a", Repository: "#6b8f3a", Workload: "#b7791f", Namespace: "#8a6d3b", Team: "#7a5ab8",
  Capability: "#2f7a72", Endpoint: "#b23b3b", Decision: "#555", MarketUnit: "#999",
};

// The neighbourhood of an entity, fetched breadth-first from the REST API up to the given depth.
export function Graph({ id, depth }: { id: string; depth: number }) {
  const el = useRef<HTMLDivElement>(null);
  const router = useRouter();
  const [hidden, setHidden] = useState<Set<Relation>>(new Set(["operated_by", "part_of", "refines", "constrains", "supports"]));
  const [data, setData] = useState<Entity[]>([]);

  useEffect(() => {
    let alive = true;
    (async () => {
      const seen = new Map<string, Entity>();
      let frontier = [id];
      for (let d = 0; d <= depth && frontier.length; d++) {
        const batch = await Promise.all(frontier.filter((f) => !seen.has(f)).slice(0, 60).map((f) =>
          fetch(`/api/v1/entity?id=${encodeURIComponent(f)}`).then((r) => r.json() as Promise<Entity>)));
        batch.forEach((e) => seen.set(e.id, e));
        frontier = d < depth ? batch.flatMap((e) => e.edges.filter((x) => !hidden.has(x.kind)).map((x) => x.other)) : [];
      }
      if (alive) setData([...seen.values()]);
    })();
    return () => { alive = false; };
  }, [id, depth, hidden]);

  useEffect(() => {
    if (!el.current || data.length === 0) return;
    const ids = new Set(data.map((e) => e.id));
    const edges = data.flatMap((e) =>
      e.edges.filter((x) => x.direction === "out" && ids.has(x.other) && !hidden.has(x.kind)).map((x) => ({
        data: { id: x.edge, source: e.id, target: x.other, label: x.kind, basis: x.basis, status: x.status },
      })));
    const cy = cytoscape({
      container: el.current,
      elements: [
        ...data.map((e) => ({ data: { id: e.id, label: e.name, kind: e.kind, root: e.id === id ? 1 : 0 } })),
        ...edges,
      ],
      style: [
        { selector: "node", style: { label: "data(label)", "font-size": 10, "background-color": (n: cytoscape.NodeSingular) => COLOURS[n.data("kind")] ?? "#888", color: "#888", width: 14, height: 14 } },
        { selector: "node[root = 1]", style: { width: 24, height: 24, "font-weight": "bold" } },
        { selector: "edge", style: { width: 1, "line-color": "#aaa", "target-arrow-shape": "triangle", "target-arrow-color": "#aaa", "curve-style": "bezier", "font-size": 8, color: "#999" } },
        { selector: "edge:selected, edge.hover", style: { label: "data(label)", width: 2 } },
        { selector: "edge[basis = 'inferred']", style: { "line-style": "dashed", "line-color": "#7a5ab8" } },
        { selector: "edge[basis = 'derived']", style: { "line-color": "#2f5d8a" } },
      ],
      layout: { name: "cose", animate: false },
    });
    cy.on("mouseover", "edge", (ev) => ev.target.addClass("hover"));
    cy.on("mouseout", "edge", (ev) => ev.target.removeClass("hover"));
    cy.on("tap", "node", (ev) => router.push(`/entity?id=${encodeURIComponent(ev.target.id())}`));
    return () => cy.destroy();
  }, [data, hidden, id, router]);

  return (
    <>
      <p className="muted" style={{ fontSize: ".85rem" }}>
        Hide:{" "}
        {RELATIONS.map((r) => (
          <label key={r} style={{ marginRight: ".7rem" }}>
            <input type="checkbox" checked={hidden.has(r)} onChange={() => {
              const n = new Set(hidden);
              if (n.has(r)) n.delete(r); else n.add(r);
              setHidden(n);
            }} /> {r}
          </label>
        ))}
      </p>
      <div id="graph" ref={el} />
      <p className="muted">{data.length} entities. Grey: asserted; blue: derived; dashed violet: inferred. Hover an edge for its relation; click a node to open it.</p>
    </>
  );
}
