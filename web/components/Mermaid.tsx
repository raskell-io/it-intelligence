"use client";
import { useEffect, useId, useState } from "react";

export function Mermaid({ text }: { text: string }) {
  const id = useId().replace(/:/g, "");
  const [svg, setSvg] = useState("");
  const [err, setErr] = useState("");
  useEffect(() => {
    let alive = true;
    import("mermaid").then(async ({ default: mermaid }) => {
      const dark = window.matchMedia("(prefers-color-scheme: dark)").matches;
      mermaid.initialize({ startOnLoad: false, theme: dark ? "dark" : "neutral", securityLevel: "strict" });
      try {
        const out = await mermaid.render(`m${id}`, text);
        if (alive) setSvg(out.svg);
      } catch (e) {
        if (alive) setErr(String(e));
      }
    });
    return () => {
      alive = false;
    };
  }, [id, text]);
  if (err) return <pre>{text}</pre>;
  return <div className="diagram" dangerouslySetInnerHTML={{ __html: svg }} />;
}
