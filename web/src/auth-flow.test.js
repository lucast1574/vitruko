import test from "node:test";
import assert from "node:assert/strict";
import {authDestination} from "./auth-flow.js";
test("Both login methods preserve a paid plan through signup and login", () => {
 for (const mode of ["signup","login"]) for(const plan of ["launch","scale"]) assert.equal(authDestination(mode,"?plan="+plan),"/app/billing?plan="+plan);
});
test("Unknown, internal and external destinations cannot redirect authentication", () => {
 for(const search of ["?plan=internal","?plan=https://evil.example","?next=https://evil.example","?plan=free"]) assert.equal(authDestination("login",search),"/app");
 assert.equal(authDestination("signup",""),"/app/keys");
});
