"use client";
// The reference has no authentication. Writes still carry a person's name, because confirmations and
// primary declarations do, so this picker sets who you act as. Replace with real identity.
import { useEffect, useState } from "react";
import type { Person } from "@/lib/api";

export function getActor(): string {
  if (typeof document === "undefined") return "";
  return document.cookie.split("; ").find((c) => c.startsWith("actor="))?.slice(6) ?? "";
}

export function ActAs() {
  const [people, setPeople] = useState<Person[]>([]);
  const [actor, setActor] = useState("");
  useEffect(() => {
    setActor(getActor());
    fetch("/api/v1/people").then((r) => r.json()).then((d) => setPeople(d.people ?? []));
  }, []);
  return (
    <label className="muted" title="No authentication in the reference: pick who you act as">
      act as{" "}
      <select
        value={actor}
        onChange={(e) => {
          document.cookie = `actor=${e.target.value}; path=/; samesite=lax`;
          setActor(e.target.value);
        }}
      >
        <option value="">nobody (read only)</option>
        {people.map((p) => (
          <option key={p.id} value={p.id}>
            {p.name} ({p.team})
          </option>
        ))}
      </select>
    </label>
  );
}
