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

test("audit groups latent debt by source presentation terrain",()=>{
  const audit=auditProjectability(readCurrentAuditInputs());
  const groups=audit.groupedCoverage;
  assert.ok(groups.default?.total>0,"default shell group exists");
  assert.ok(Object.keys(groups).some(key=>key.startsWith("shell:")),"shell terrain groups exist");
  assert.ok(Object.keys(groups).some(key=>key.startsWith("shell:discover/discover:")),"Discover sub-terrain groups exist");
  const groupedTotal=Object.values(groups).reduce((sum,group)=>sum+group.total,0);
  const groupedLatent=Object.values(groups).reduce((sum,group)=>sum+group.latent,0);
  assert.equal(groupedTotal,audit.total);
  assert.equal(groupedLatent,audit.latent);
});


test("explicit hidden owners distinguish Library terrain from nested Library states",()=>{
  const audit=auditProjectability(readCurrentAuditInputs());
  const layout=audit.rows.find(row=>row.id==="songLayoutButton");
  const selection=audit.rows.find(row=>row.id==="bulkPlaySelected");
  const detail=audit.rows.find(row=>row.id==="libraryGroupPlay");
  assert.equal(layout.presentation.hiddenAncestor?.shellPanel,"library",
    "main Library controls are latent because the Library terrain itself is hidden");
  assert.equal(selection.presentation.hiddenAncestor?.id,"librarySelectionBar",
    "selection actions keep their nested presentation owner");
  assert.equal(detail.presentation.hiddenAncestor?.id,"libraryGroupDetail",
    "group-detail actions keep their nested presentation owner");
  assert.ok(audit.groupedCoverage["shell:library"]);
  assert.ok(audit.groupedCoverage["shell:library/hidden:#librarySelectionBar"]);
  assert.ok(audit.groupedCoverage["shell:library/hidden:#libraryGroupDetail"]);
});


test("admitted expressions promote scenario-visible unresolved controls into blockers",()=>{
  const audit=auditProjectability(readCurrentAuditInputs());
  const scenario=audit.rows.filter(row=>row.status==="SCENARIO_WIRED_UNIDENTIFIED"||row.status==="SCENARIO_UNIDENTIFIED");
  assert.deepEqual(
    scenario.filter(row=>row.status==="SCENARIO_WIRED_UNIDENTIFIED").map(row=>row.id).sort(),
    ["librarySort","playAllSongs","selectSongs","songLayoutButton"].sort()
  );
  assert.equal(scenario.some(row=>row.presentation?.hiddenAncestor?.id==="librarySelectionBar"),false,
    "nested Library selection bar remains latent until that presentation state is admitted");
  assert.ok(audit.admittedScenarioStructuralRefs.includes("vexstream.desktop.library.panel.library"));
  assert.ok(audit.blocking.length>=4);
});
