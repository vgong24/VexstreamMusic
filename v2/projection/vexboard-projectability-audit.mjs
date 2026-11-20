// [VXG RealForever]
// Source-wide projectability coverage census for the static V4 shell.

import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const HERE=path.dirname(fileURLToPath(import.meta.url));
const V2=path.resolve(HERE,"..");
const INTERACTIVE=new Set(["button","input","select","textarea","summary","a"]);
const VOID=new Set(["area","base","br","col","embed","hr","img","input","link","meta","param","source","track","wbr"]);

function attrs(raw){
  const result={};
  const re=/([:\w-]+)(?:\s*=\s*(?:"([^"]*)"|'([^']*)'|([^\s"'=<>]+)))?/g;
  for(const match of String(raw).matchAll(re)){
    result[match[1].toLowerCase()]=match[2]??match[3]??match[4]??true;
  }
  return result;
}

function relationOf(a){
  if(a["data-vex-control"])return {kind:"vex-control",identity:String(a["data-vex-control"])};
  if(a["data-vex-action"])return {kind:"vex-action",identity:String(a["data-vex-action"])};
  if(a["data-shell-view"])return {kind:"shell-view",identity:"vexstream.shell.view."+String(a["data-shell-view"])};
  if(a["data-discover-mode"])return {kind:"discover-mode",identity:"vexstream.discover.mode."+String(a["data-discover-mode"])};
  if(a["data-library-mode"])return {kind:"library-mode",identity:"vexstream.library.mode."+String(a["data-library-mode"])};
  if(a["data-phone-action"])return {kind:"phone-action",identity:"vexstream.phone.action."+String(a["data-phone-action"])};
  if(a["data-system-action"])return {kind:"system-action",identity:"vexstream.system.action."+String(a["data-system-action"])};
  if(a["data-track-action"])return {kind:"track-action",identity:"vexstream.track.action."+String(a["data-track-action"])};
  return null;
}

function nearestContext(stack,attribute){
  for(let index=stack.length-1;index>=0;index-=1){
    const value=stack[index].attrs?.[attribute];
    if(typeof value==="string"&&value)return value;
  }
  return null;
}

function nearestExplicitHiddenAncestor(stack){
  for(let index=stack.length-1;index>=0;index-=1){
    const entry=stack[index];
    if(!entry.explicitHidden)continue;
    return {
      tag:entry.tag,
      id:typeof entry.attrs?.id==="string"?entry.attrs.id:null,
      className:typeof entry.attrs?.class==="string"?entry.attrs.class:null,
      shellPanel:typeof entry.attrs?.["data-shell-panel"]==="string"?entry.attrs["data-shell-panel"]:null,
      discoverPanel:typeof entry.attrs?.["data-discover-panel"]==="string"?entry.attrs["data-discover-panel"]:null
    };
  }
  return null;
}

function presentationContext(stack){
  return {
    shellPanel:nearestContext(stack,"data-shell-panel"),
    discoverPanel:nearestContext(stack,"data-discover-panel"),
    hiddenAncestor:nearestExplicitHiddenAncestor(stack)
  };
}

export function scanInteractiveHTML(html){
  const stack=[];
  const rows=[];
  let ordinal=0;
  const tokens=/<!--[\s\S]*?-->|<\/?[A-Za-z][^>]*>/g;
  for(const match of String(html).matchAll(tokens)){
    const token=match[0];
    if(token.startsWith("<!--"))continue;
    const close=token.match(/^<\/\s*([A-Za-z][\w:-]*)/);
    if(close){
      const tag=close[1].toLowerCase();
      while(stack.length){const top=stack.pop();if(top.tag===tag)break;}
      continue;
    }
    const open=token.match(/^<\s*([A-Za-z][\w:-]*)([\s\S]*?)\/?\s*>$/);
    if(!open)continue;
    const tag=open[1].toLowerCase();
    const a=attrs(open[2]);
    const explicitHidden=Object.hasOwn(a,"hidden");
    const parentHidden=stack.some(entry=>entry.hidden);
    const hidden=parentHidden||explicitHidden;
    if(INTERACTIVE.has(tag)&&!(tag==="a"&&!a.href)){
      ordinal+=1;
      rows.push({
        ordinal,
        tag,
        id:typeof a.id==="string"?a.id:null,
        hidden,
        disabled:Object.hasOwn(a,"disabled"),
        relation:relationOf(a),
        textRef:typeof a["data-text-ref"]==="string"?a["data-text-ref"]:null,
        presentation:presentationContext(stack),
        attrs:a
      });
    }
    if(!VOID.has(tag)&&!token.endsWith("/>"))stack.push({tag,hidden,explicitHidden,attrs:a});
  }
  return rows;
}

export function directRuntimeEvents(app){
  const vars=new Map();
  const byId=new Map();
  for(const match of String(app).matchAll(/const\s+([A-Za-z_$][\w$]*)\s*=\s*document\.querySelector\("#([^"]+)"\);/g)){
    vars.set(match[1],match[2]);
  }
  for(const match of String(app).matchAll(/([A-Za-z_$][\w$]*)\.addEventListener\(\s*"([^"]+)"/g)){
    const id=vars.get(match[1]);
    if(!id)continue;
    const events=byId.get(id)??new Set();
    events.add(match[2]);
    byId.set(id,events);
  }
  return byId;
}

function allLensElements(lens){
  return (lens.screens??[]).flatMap(screen=>screen.elements??[]);
}

function lensElementFor(row,lens){
  const elements=allLensElements(lens);
  if(row.relation){
    return elements.find(element=>element.ref==="element."+row.relation.identity)??null;
  }
  if(row.id){
    return elements.find(element=>String(element.declared??"").includes('id="'+row.id+'"'))??null;
  }
  return null;
}

function classify(row,lens,eventsById){
  const element=lensElementFor(row,lens);
  const events=row.id?[...(eventsById.get(row.id)??[])]:[];
  if(element?.pathwayRef)return {status:"MAPPED_PATHWAY",elementRef:element.ref,pathwayRef:element.pathwayRef,events};
  if(element)return {status:"ADDRESSABLE_HELD",elementRef:element.ref,pathwayRef:null,events};
  if(row.relation)return {status:"SOURCE_RELATIONSHIP_UNMAPPED",elementRef:null,pathwayRef:null,events};
  if(row.hidden)return {status:events.length?"LATENT_WIRED_UNIDENTIFIED":"LATENT_UNIDENTIFIED",elementRef:null,pathwayRef:null,events};
  if(row.disabled&&!events.length)return {status:"VISIBLE_DISABLED_UNIDENTIFIED",elementRef:null,pathwayRef:null,events};
  if(events.length)return {status:"VISIBLE_WIRED_UNIDENTIFIED",elementRef:null,pathwayRef:null,events};
  return {status:"VISIBLE_UNIDENTIFIED",elementRef:null,pathwayRef:null,events};
}

function hiddenKey(hidden){
  if(hidden?.id)return "#"+hidden.id;
  if(hidden?.className)return "."+String(hidden.className).trim().split(/\s+/)[0];
  return hidden?.tag??null;
}

function terrainKey(row){
  const shell=row.presentation?.shellPanel;
  const discover=row.presentation?.discoverPanel;
  const hidden=row.presentation?.hiddenAncestor;
  const base=shell==="discover"&&discover
    ?"shell:discover/discover:"+discover
    :shell
      ?"shell:"+shell
      :null;
  const hiddenIsTerrainRoot=
    (shell&&hidden?.shellPanel===shell)||
    (discover&&hidden?.discoverPanel===discover);
  const nestedHidden=hiddenIsTerrainRoot?null:hiddenKey(hidden);
  if(base&&nestedHidden)return base+"/hidden:"+nestedHidden;
  if(base)return base;
  if(nestedHidden)return "hidden:"+nestedHidden;
  return row.hidden?"latent:other":"default";
}

function groupedCoverage(rows){
  const groups={};
  for(const row of rows){
    const key=terrainKey(row);
    const group=groups[key]??{total:0,visible:0,latent:0,statuses:{}};
    group.total+=1;
    if(row.hidden)group.latent+=1;
    else group.visible+=1;
    group.statuses[row.status]=(group.statuses[row.status]??0)+1;
    groups[key]=group;
  }
  return groups;
}

export function auditProjectability({html,app,lens}){
  const eventsById=directRuntimeEvents(app);
  const rows=scanInteractiveHTML(html).map(row=>({...row,...classify(row,lens,eventsById)}));
  const counts={};
  for(const row of rows)counts[row.status]=(counts[row.status]??0)+1;
  const blocking=rows.filter(row=>
    row.status==="SOURCE_RELATIONSHIP_UNMAPPED"||
    row.status==="VISIBLE_WIRED_UNIDENTIFIED"||
    row.status==="VISIBLE_UNIDENTIFIED"
  );
  return {
    schemaVersion:"vexstream.vexboard-projectability-audit/v1",
    generatedFrom:{
      html:"v2/web/index.html",
      runtime:"v2/web/app.js",
      lens:"v2/projection/vexboard-screen-lens.json"
    },
    total:rows.length,
    defaultVisible:rows.filter(row=>!row.hidden).length,
    latent:rows.filter(row=>row.hidden).length,
    counts,
    groupedCoverage:groupedCoverage(rows),
    blocking:blocking.map(row=>row.ordinal),
    rows
  };
}

export function readCurrentAuditInputs(){
  return {
    html:fs.readFileSync(path.join(V2,"web","index.html"),"utf8"),
    app:fs.readFileSync(path.join(V2,"web","app.js"),"utf8"),
    lens:JSON.parse(fs.readFileSync(path.join(HERE,"vexboard-screen-lens.json"),"utf8"))
  };
}

export const stableJSON=value=>JSON.stringify(value,null,2)+"\n";

if(process.argv[1]&&path.resolve(process.argv[1])===fileURLToPath(import.meta.url)){
  const audit=auditProjectability(readCurrentAuditInputs());
  if(process.argv.includes("--stdout"))process.stdout.write(stableJSON(audit));
  else fs.writeFileSync(path.join(HERE,"vexboard-projectability-audit.json"),stableJSON(audit));
  if(process.argv.includes("--strict")&&audit.blocking.length){
    process.stderr.write("VexBoard projectability audit found "+audit.blocking.length+" blocking interactive affordance(s).\n");
    process.exitCode=1;
  }
}
