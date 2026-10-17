// [VXG RealForever]
// Read-only projection for the recognizable desktop System status terrain.
//
// This module composes facts already owned by System identity, Library, Sources,
// Player session, Imports, and Trash. It performs no repair, diagnostics,
// lifecycle, provider, or mobile effect.

const HELD_SESSION_KINDS=new Set([
 "held-current-unavailable",
 "held-schema",
 "persistence-held"
]);

export function projectSystemStatus({
 identity=null,
 catalog=[],
 sourceConfig=null,
 sessionRestoration=null,
 importJobs=null,
 trash=null
}={}){
 const warnings=uniqueStrings(identity?.warnings);
 const activeImports=array(importJobs?.active).length;
 const readyImports=array(importJobs?.ready).length;
 const importIssues=array(importJobs?.issues).length;
 const importRecovery=Object.values(
  object(importJobs?.recovery)
 ).filter(value=>String(value??"").trim()!=="").length;
 const sessionKind=String(sessionRestoration?.kind??"fresh").trim()||"fresh";
 const sessionHeld=HELD_SESSION_KINDS.has(sessionKind);
 const pendingPurge=Boolean(trash?.pendingPurge);
 const attentionCount=
  warnings.length+
  readyImports+
  importIssues+
  importRecovery+
  (sessionHeld?1:0)+
  (pendingPurge?1:0);

 return Object.freeze({
  version:String(identity?.version??"").trim(),
  buildRef:String(identity?.buildRef??"").trim(),
  warnings:Object.freeze(warnings),
  trackCount:array(catalog).length,
  sourceCount:array(sourceConfig?.sources).length,
  sessionKind,
  sessionHeld,
  activeImports,
  readyImports,
  importIssues,
  importRecovery,
  trashCount:array(trash?.trash).length,
  pendingPurge,
  attentionCount,
  state:attentionCount>0?"attention":"healthy"
 });
}

function array(value){
 return Array.isArray(value)?value:[];
}

function object(value){
 return value&&typeof value==="object"&&!Array.isArray(value)?value:{};
}

function uniqueStrings(values){
 const seen=new Set();
 const result=[];
 for(const value of array(values)){
  const text=String(value??"").trim();
  if(!text||seen.has(text))continue;
  seen.add(text);
  result.push(text);
 }
 return result;
}
