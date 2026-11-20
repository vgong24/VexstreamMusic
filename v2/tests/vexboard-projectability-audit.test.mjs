// [VXG RealForever]
import assert from "node:assert/strict";
import fs from "node:fs";
import path from "node:path";
import test from "node:test";
import { fileURLToPath } from "node:url";

import {
  auditProjectability,
  readCurrentAuditInputs,
  stableJSON
} from "../projection/vexboard-projectability-audit.mjs";

const HERE=path.dirname(fileURLToPath(import.meta.url));
const V2=path.resolve(HERE,"..");

test("committed projectability audit regenerates from current shell/runtime/Lens",()=>{
  const committed=fs.readFileSync(path.join(V2,"projection","vexboard-projectability-audit.json"),"utf8");
  assert.equal(committed,stableJSON(auditProjectability(readCurrentAuditInputs())));
});

test("every source-stable relationship is at least addressable",()=>{
  const audit=auditProjectability(readCurrentAuditInputs());
  const stable=audit.rows.filter(row=>row.relation);
  assert.ok(stable.length>0);
  assert.equal(stable.some(row=>row.status==="SOURCE_RELATIONSHIP_UNMAPPED"),false);
});

test("audit mechanically exposes visible runtime-wired affordances still missing truthful projection identity",()=>{
  const audit=auditProjectability(readCurrentAuditInputs());
  const unresolved=audit.rows.filter(row=>row.status==="VISIBLE_WIRED_UNIDENTIFIED");
  assert.ok(unresolved.every(row=>row.id&&row.events.length));
  assert.equal(Array.isArray(audit.blocking),true);
});
