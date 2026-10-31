// Proxies /api/v1/* to itx, so the browser talks to one origin. ITX_API is read at request time, so the
// same image works under compose, Helm or anywhere else.
import type { NextRequest } from "next/server";

export const dynamic = "force-dynamic";

async function forward(req: NextRequest, { params }: { params: Promise<{ path: string[] }> }) {
  const { path } = await params;
  const base = process.env.ITX_API ?? "http://localhost:8080";
  const url = `${base}/api/v1/${path.map(encodeURIComponent).join("/")}${req.nextUrl.search}`;
  const headers: Record<string, string> = { "Content-Type": "application/json" };
  const actor = req.headers.get("x-actor");
  if (actor) headers["X-Actor"] = actor;
  const res = await fetch(url, {
    method: req.method,
    headers,
    body: req.method === "POST" ? await req.text() : undefined,
    cache: "no-store",
  });
  return new Response(res.body, { status: res.status, headers: { "Content-Type": res.headers.get("Content-Type") ?? "application/json" } });
}

export { forward as GET, forward as POST };
