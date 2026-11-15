import test from "node:test";
import assert from "node:assert/strict";
import { installPrompt } from "./mcp-config.js";
test("personal MCP instructions embed the supplied credential and preserve approvals", () => {
  const connection = {
    url: "https://api.vitruko.lat/mcp",
    key: "test-private-key",
  };
  for (const client of ["codex", "claude"]) {
    const prompt = installPrompt(client, connection);
    assert.ok(prompt.includes(connection.key));
    assert.ok(prompt.includes(connection.url));
    assert.ok(prompt.includes("Mantén las confirmaciones"));
    assert.ok(prompt.includes("No hagas consultas de datos"));
    assert.ok(prompt.includes("vitruko_me"));
  }
  assert.match(installPrompt("codex", connection), /mcp_servers\.vitruko/);
  const json = installPrompt("claude", connection).split("\n\n").at(-1);
  assert.equal(
    JSON.parse(json).mcpServers.vitruko.headers.Authorization,
    "Bearer test-private-key",
  );
});
