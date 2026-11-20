// [VXG RealForever]
//
// Trash-owned runtime presentation identity.
//
// Trash item domain/custody identity remains owned by the Trash API/client. This file only names how
// one durable Trash record becomes repeated browser presentation. Runtime children inherit from that
// source-owned instance instead of using text, list position or generated VexBoard-only IDs.

import {
  VEX_PRESENTATION_INSTANCE_ENCODING,
  VexPresentationContinuity,
  appendVexPresentationPath,
  dynamicVexPresentationRef,
} from "../../foundation/identity.js";

export const TrashPresentationIdentity=Object.freeze({
  Item:Object.freeze({
    kind:"RUNTIME_INSTANCE",
    baseRef:"element.vexstream.trash.item",
    partSources:Object.freeze(["item.id"]),
    encoding:VEX_PRESENTATION_INSTANCE_ENCODING,
    continuity:VexPresentationContinuity.STABLE_INSTANCE,
  }),
});

export function trashItemPresentationRef(itemID){
  return dynamicVexPresentationRef(
    TrashPresentationIdentity.Item.baseRef,
    [required(itemID,"trash item id")]
  );
}

export function trashItemChildRef(itemID,localPath){
  return appendVexPresentationPath(
    trashItemPresentationRef(itemID),
    localPath
  );
}

function required(value,label){
  const text=String(value??"").trim();
  if(!text)throw new Error(label+" must not be blank");
  return text;
}
