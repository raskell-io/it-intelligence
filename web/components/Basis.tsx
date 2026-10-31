import type { BasisName, Status } from "@/lib/api";

export function BasisPill({ basis, confidence, status }: { basis: BasisName; confidence?: number; status?: Status }) {
  const label = status === "pending" ? "pending" : basis;
  return (
    <span className={`pill ${label}`} title={`${basis}${confidence !== undefined ? `, confidence ${confidence.toFixed(2)}` : ""}${status ? `, ${status}` : ""}`}>
      {label}
      {confidence !== undefined && basis !== "asserted" ? ` ${confidence.toFixed(2)}` : ""}
    </span>
  );
}
