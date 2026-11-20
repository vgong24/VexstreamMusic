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

test("admitted Discover modes have addressable immediate controls while Import review remains nested debt",()=>{
 const audit=auditProjectability(readCurrentAuditInputs());
 assert.deepEqual(audit.groupedCoverage["shell:discover"].statuses,{ADDRESSABLE_HELD:3});
 assert.deepEqual(audit.groupedCoverage["shell:discover/discover:pc"].statuses,{ADDRESSABLE_HELD:5});
 assert.deepEqual(audit.groupedCoverage["shell:discover/discover:youtube"].statuses,{ADDRESSABLE_HELD:3});
 assert.deepEqual(audit.groupedCoverage["shell:discover/discover:explore"].statuses,{ADDRESSABLE_HELD:2});
 assert.ok(audit.groupedCoverage["shell:discover/discover:youtube/hidden:#importReview"]);
});


test("admitted Playlists terrain has addressable create controls while selected-playlist detail remains nested debt",()=>{
 const audit=auditProjectability(readCurrentAuditInputs());
 assert.deepEqual(audit.groupedCoverage["shell:playlists"].statuses,{ADDRESSABLE_HELD:2});
 assert.ok(audit.groupedCoverage["shell:playlists/hidden:#playlistDetail"]);
 assert.equal(audit.groupedCoverage["shell:playlists/hidden:#playlistDetail"].total,7);
});


test("selected Playlist detail has no unresolved fixed affordances once its presentation is admitted",()=>{
  const audit=auditProjectability(readCurrentAuditInputs());
  assert.deepEqual(
    audit.groupedCoverage["shell:playlists/hidden:#playlistDetail"].statuses,
    {MAPPED_PATHWAY:7}
  );
});


test("admitted Deleted terrain makes its immediate stable control addressable while nested Trash states remain separate",()=>{
  const audit=auditProjectability(readCurrentAuditInputs());
  assert.deepEqual(
    audit.groupedCoverage["shell:deleted"].statuses,
    {ADDRESSABLE_HELD:1}
  );
  const refresh=audit.rows.find(row=>row.id==="trashRefresh");
  assert.equal(refresh.relation?.identity,"vexstream.trash.refresh");
  assert.equal(refresh.elementRef,"element.vexstream.trash.refresh");
  assert.equal(refresh.status,"ADDRESSABLE_HELD");

  assert.ok(audit.groupedCoverage["shell:deleted/hidden:#trashSelectionBar"]);
  assert.ok(audit.groupedCoverage["shell:deleted/hidden:#trashPurgeConfirm"]);
  assert.ok(audit.groupedCoverage["shell:deleted/hidden:#purgeRecovery"]);
});
