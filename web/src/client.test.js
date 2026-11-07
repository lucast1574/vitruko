import test from "node:test";
import assert from "node:assert/strict";
import { api } from "./client.js";

test("API requests preserve credentials, JSON and authorization headers", async (t) => {
  const signal = new AbortController().signal;
  t.mock.method(globalThis, "fetch", async (url, options) => {
    assert.equal(url, "/console/keys");
    assert.equal(options.credentials, "same-origin");
    assert.equal(options.headers["Content-Type"], "application/json");
    assert.equal(options.headers["X-API-Key"], "test-only");
    assert.equal(options.body, '{"name":"QA"}');
    assert.equal(options.signal, signal);
    return Response.json({ id: "created" });
  });
  assert.deepEqual(await api("/console/keys", "POST", {name: "QA"}, {"X-API-Key": "test-only"}, signal), {id: "created"});
});

test("API rejection preserves user-facing error, status and code", async (t) => {
  t.mock.method(globalThis, "fetch", async () => Response.json({error: "Sesión caducada", code: "unauthorized"}, {status: 401}));
  await assert.rejects(api("/console/session"), e => e.message === "Sesión caducada" && e.status === 401 && e.code === "unauthorized");
});

test("HTML proxy errors become actionable errors instead of JSON syntax failures", async (t) => {
  t.mock.method(globalThis, "fetch", async () => new Response("<html>Bad gateway</html>", {status: 502}));
  await assert.rejects(api("/console/usage"), e => e.status === 502 && e.message.includes("Intenta de nuevo"));
});

test("Empty successful responses are accepted", async (t) => {
  t.mock.method(globalThis, "fetch", async () => new Response(null, {status: 204}));
  assert.equal(await api("/console/logout", "POST", {}), null);
});
