// [VXG RealForever]
//
// Portable presentation identity for runtime-rendered VexStream UI.
//
// This is presentation addressability, not domain ownership. Stable source/domain refs remain owned by
// their feature. Runtime presentation derives one deterministic address from a stable base plus exact
// identity parts so reorder, translation and visual position do not become identity by accident.

export const VexPresentationContinuity = Object.freeze({
 STABLE_INSTANCE:"STABLE_INSTANCE",
 CURRENT_OCCURRENCE:"CURRENT_OCCURRENCE"
});

export const VEX_PRESENTATION_INSTANCE_ENCODING="UTF8_HEX_NUL_PARTS";

function required(value,label){
 const text=String(value??"").trim();
 if(!text)throw new Error(label+" must not be blank");
 return text;
}

export function normalizeVexPresentationPath(value){
 const normalized=required(value,"Vex presentation path")
  .replaceAll("/",".")
  .replace(/\s+/g,"-")
  .replace(/^\.+|\.+$/g,"");
 if(!normalized)throw new Error("Vex presentation path must not normalize to blank");
 return normalized;
}

export function appendVexPresentationPath(parentRef,childPath){
 return required(parentRef,"Vex presentation parent ref")+"."+normalizeVexPresentationPath(childPath);
}

export function dynamicVexPresentationRef(baseRef,parts){
 const base=required(baseRef,"Vex presentation base ref");
 if(!Array.isArray(parts)||!parts.length){
  throw new Error("Vex presentation identity parts must be non-empty");
 }
 const values=parts.map((part,index)=>required(part,"Vex presentation identity part "+index));
 const bytes=new TextEncoder().encode(values.join("\u0000"));
 const encoded=[...bytes].map(byte=>byte.toString(16).padStart(2,"0")).join("");
 return base+".dynamic.utf8hex."+encoded;
}
