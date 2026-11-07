import test from "node:test";
import assert from "node:assert/strict";
import { JSDOM } from "jsdom";
import React, { act } from "react";
import { createRoot } from "react-dom/client";
import { useData } from "./useData.js";

function harness(t) {
  const dom = new JSDOM('<div id="root"></div>', {url: "https://app.example.test/app"});
  globalThis.window = dom.window;
  globalThis.document = dom.window.document;
  globalThis.IS_REACT_ACT_ENVIRONMENT = true;
  let redirected = "", state;
  globalThis.location = {assign: path => { redirected = path; }};
  const root = createRoot(document.getElementById("root"));
  function Probe({path}) { state = useData(path); return null; }
  t.after(async () => { await act(async () => root.unmount()); dom.window.close(); });
  return {get state() {return state;}, get redirected() {return redirected;}, render: path => act(async () => root.render(React.createElement(Probe, {path})))};
}
const deferred = () => { let resolve; const promise = new Promise(r => {resolve = r;}); return {promise, resolve}; };

test("Failed reload rejects so a mutation cannot falsely announce refresh success", async (t) => {
  const h = harness(t);
  let fail = false;
  t.mock.method(globalThis, "fetch", async () => fail ? Response.json({error: "No se pudo cargar"}, {status: 503}) : Response.json({name: "Original"}));
  await h.render("/console/admin/users");
  fail = true;
  await act(async () => { await assert.rejects(h.state.reload(), /No se pudo cargar/); });
  assert.equal(h.state.error, "No se pudo cargar");
  assert.equal(h.state.loading, false);
  fail = false;
  await act(async () => { await h.state.reload(); });
  assert.equal(h.state.error, "");
});

test("New search wins even when the old request finishes last", async (t) => {
  const h = harness(t), first = deferred(), second = deferred();
  let firstSignal;
  t.mock.method(globalThis, "fetch", (path, options) => { if(path.endsWith("old")){ firstSignal = options.signal; return first.promise; } return second.promise; });
  await h.render("/console/admin/users?q=old");
  await h.render("/console/admin/users?q=new");
  assert.equal(firstSignal.aborted, true);
  await act(async () => { second.resolve(Response.json({users: ["new"]})); });
  await act(async () => { first.resolve(Response.json({users: ["old"]})); });
  assert.deepEqual(h.state.data, {users: ["new"]});
  assert.equal(h.state.loading, false);
});

test("Changing query clears stale accounts until the new query is ready", async (t) => {
  const h = harness(t), pending = deferred();
  t.mock.method(globalThis, "fetch", path => path.endsWith("old") ? Promise.resolve(Response.json({users: ["old"]})) : pending.promise);
  await h.render("/console/admin/users?q=old");
  assert.deepEqual(h.state.data, {users: ["old"]});
  await h.render("/console/admin/users?q=new");
  assert.equal(h.state.data, null);
  assert.equal(h.state.loading, true);
  await act(async () => {pending.resolve(Response.json({users: []}));});
});

test("Expired initial session redirects to login", async (t) => {
  const h = harness(t);
  t.mock.method(globalThis, "fetch", async () => Response.json({error: "Inicia sesión"}, {status: 401}));
  await h.render("/console/session");
  assert.equal(h.redirected, "/login");
  assert.equal(h.state.error, "Inicia sesión");
});
