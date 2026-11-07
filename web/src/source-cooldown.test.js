import test from "node:test";
import assert from "node:assert/strict";
import {sourceRetryDeadline,sourceHost} from "./source-cooldown.js";
test("Store retry deadlines respect seconds and HTTP dates with bounded fallback",()=>{
 const now=Date.UTC(2026,10,6,12,0,0);
 assert.equal(sourceRetryDeadline("60",now),now+60000);
 assert.equal(sourceRetryDeadline(new Date(now+120000).toUTCString(),now),now+120000);
 for(const value of [null,"invalid","-50","99999999999999999999999"]) assert.equal(sourceRetryDeadline(value,now),now+60000);
 assert.equal(sourceRetryDeadline("99999999",now),now+86400000);
});
test("Cooldown identity is per store, independent of its product path",()=>{
 assert.equal(sourceHost("https://shop.example/es/products/cup?variant=42"),"shop.example");
 assert.notEqual(sourceHost("https://other.example/p"),"shop.example");
 assert.equal(sourceHost("broken"),"");
});
