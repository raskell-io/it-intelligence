@AGENTS.md

## For Claude Code

- `.mcp.json` connects this session to the running itx (`make dev-serve` or `make up`). Use the
  `it-intelligence` MCP tools to answer questions about the Kestrel Group; that is also how to check that a
  change to the tools still reads well for an agent.
- Verify the UI visually with headless Chrome (playwright-core in the session scratchpad), light, dark and
  phone width, not just a successful build.
