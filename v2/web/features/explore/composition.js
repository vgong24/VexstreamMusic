// [VXG RealForever]
// Higher composition for Explore provider preview crossing into local Playback.
//
// Explore never realizes Playback Effects directly. It carries the stable Playback Control back
// through the Playback connection, which re-resolves current meaning from current provider truth.

import { PlaybackAct, PlaybackControl } from "../playback/behavior.js";
import {
  providerPreviewURL,
  requestExploreFromCandidate,
  requestExploreFromTrack,
} from "./client.js";

export function createExploreComposition({
  session,
  playback,
  requestTrack = requestExploreFromTrack,
  requestCandidate = requestExploreFromCandidate,
}) {
  if (!session || !playback) {
    throw new Error("Explore composition requires session and Playback");
  }

  let resumeLocalAfterPreview = false;

  async function stopPreview({ resumeLocal = false } = {}) {
    session.stopPreview();

    if (resumeLocal && resumeLocalAfterPreview) {
      const control = playback.currentProjection.control;
      if (
        control?.enabled &&
        control.act === PlaybackAct.Play
      ) {
        await playback.activate(PlaybackControl.PlayPause);
      }
    }

    resumeLocalAfterPreview = false;
    return session.currentProjection;
  }

  async function replaceFromTrack(trackRef) {
    await stopPreview();
    const frame = await requestTrack(trackRef);
    return session.replace(frame);
  }

  async function branchFromCandidate(index) {
    const candidate = candidateAt(session.currentProjection, index);
    await stopPreview();
    const frame = await requestCandidate(candidate);
    return session.branch(frame);
  }

  async function preview(index) {
    const candidate = candidateAt(session.currentProjection, index);
    const url = providerPreviewURL(candidate);
    if (!url) {
      throw new Error("Explore candidate has no preview provider id");
    }

    if (session.currentProjection.previewIndex < 0) {
      const control = playback.currentProjection.control;
      resumeLocalAfterPreview =
        control?.enabled === true &&
        control.act === PlaybackAct.Pause;

      if (resumeLocalAfterPreview) {
        await playback.activate(PlaybackControl.PlayPause);
      }
    }

    session.selectPreview(index);
    return Object.freeze({ candidate, url });
  }

  async function back() {
    await stopPreview();
    return session.back();
  }

  async function clear() {
    await stopPreview({ resumeLocal: true });
    return session.clear();
  }

  return Object.freeze({
    replaceFromTrack,
    branchFromCandidate,
    preview,
    stopPreview,
    back,
    clear,
  });
}

function candidateAt(projection, index) {
  const rows = projection?.frame?.candidates;
  const normalized = Number(index);
  if (
    !Array.isArray(rows) ||
    !Number.isInteger(normalized) ||
    normalized < 0 ||
    normalized >= rows.length
  ) {
    throw new Error("Explore candidate index is unavailable");
  }
  return rows[normalized];
}
