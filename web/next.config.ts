import type { NextConfig } from "next";

// /api/v1 is proxied to itx by app/api/v1/[...path]/route.ts, at request time.
const config: NextConfig = { output: "standalone" };

export default config;
