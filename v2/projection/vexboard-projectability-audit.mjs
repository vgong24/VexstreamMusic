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
    const hidden=stack.some(entry=>entry.hidden)||Object.hasOwn(a,"hidden");
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
        attrs:a
      });
    }
    if(!VOID.has(tag)&&!token.endsWith("/>"))stack.push({tag,hidden});
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
    blocking: blocking.map(row=>row.ordinal),
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
