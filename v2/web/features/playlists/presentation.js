// [VXG RealForever]
//
// Playlist-owned runtime presentation identity.
//
// Durable Playlist truth stays in the Playlist model/client. This file owns only how that truth is
// addressed when it becomes repeated browser presentation. The same contract is consumed by the
// live renderer and the VexBoard exporter.

import {
 VEX_PRESENTATION_INSTANCE_ENCODING,
 VexPresentationContinuity,
 appendVexPresentationPath,
 dynamicVexPresentationRef,
} from "../../foundation/identity.js";

export const PlaylistPresentationIdentity=Object.freeze({
 Card:Object.freeze({
  kind:"RUNTIME_INSTANCE",
  baseRef:"element.vexstream.playlists.card",
  partSources:Object.freeze(["playlist.id"]),
  encoding:VEX_PRESENTATION_INSTANCE_ENCODING,
  continuity:VexPresentationContinuity.STABLE_INSTANCE,
 }),
 DetailEntry:Object.freeze({
  kind:"RUNTIME_INSTANCE",
  baseRef:"element.vexstream.playlists.entry",
  partSources:Object.freeze([
   "playlist.id",
   "entry.ref",
   "occurrenceWithinPlaylist",
  ]),
  encoding:VEX_PRESENTATION_INSTANCE_ENCODING,
  continuity:VexPresentationContinuity.CURRENT_OCCURRENCE,
 }),
});

export function playlistCardPresentationRef(playlistID){
 return dynamicVexPresentationRef(
  PlaylistPresentationIdentity.Card.baseRef,
  [required(playlistID,"playlist id")]
 );
}

export function playlistCardChildRef(playlistID,localPath){
 return appendVexPresentationPath(
  playlistCardPresentationRef(playlistID),
  localPath
 );
}

export function playlistEntryPresentationRef({
 playlistID,
 trackRef,
 occurrence,
}){
 const index=Number(occurrence);
 if(!Number.isInteger(index)||index<1){
  throw new Error("playlist entry occurrence must be >= 1");
 }
 return dynamicVexPresentationRef(
  PlaylistPresentationIdentity.DetailEntry.baseRef,
  [
   required(playlistID,"playlist id"),
   required(trackRef,"playlist entry track ref"),
   String(index),
  ]
 );
}

export function playlistEntryChildRef(input,localPath){
 return appendVexPresentationPath(
  playlistEntryPresentationRef(input),
  localPath
 );
}

function required(value,label){
 const text=String(value??"").trim();
 if(!text)throw new Error(label+" must not be blank");
 return text;
}
