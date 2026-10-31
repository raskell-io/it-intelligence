"use client";
import { useRouter } from "next/navigation";
import { useState } from "react";
import { getActor } from "./ActAs";

export function ReviewActions({ type, id }: { type: "edge" | "alias"; id: string }) {
  const router = useRouter();
  const [msg, setMsg] = useState("");
  async function act(action: "confirm" | "reject") {
    const actor = getActor();
    if (!actor) {
      setMsg("Pick who you act as first (top right).");
      return;
    }
    const res = await fetch(`/api/v1/${type === "edge" ? "edges" : "aliases"}/${id}/${action}`, {
      method: "POST",
      headers: { "X-Actor": actor, "Content-Type": "application/json" },
      body: JSON.stringify({ note: "" }),
    });
    const body = await res.json();
    if (!res.ok) {
      setMsg(body.error);
      return;
    }
    setMsg(`${action === "confirm" ? "Confirmed" : "Rejected"}; run ${body.run} rebuilds the model.`);
    setTimeout(() => router.refresh(), 1500);
  }
  return (
    <span>
      <button className="ok" onClick={() => act("confirm")}>Confirm</button>{" "}
      <button className="bad" onClick={() => act("reject")}>Reject</button>
      {msg && <div className="muted">{msg}</div>}
    </span>
  );
}
