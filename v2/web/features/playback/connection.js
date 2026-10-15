// [VXG RealForever]
//
// Browser provider realization for the V2 Playback family.
//
// HTMLMediaElement remains the provider truth owner. This Connection translates provider facts into
// semantic Playback State, re-resolves Control meaning at activation time, realizes Effects, and
// keeps pending intent distinct from provider acknowledgement.

import {
  PlaybackEffect,
  PlaybackState,
  resolvePlaybackControl,
  vexStreamPlaybackBehavior,
} from "./behavior.js";
import { presentPlaybackControl } from "./presentation.js";

function phaseFromMedia(media) {
  if (media.readyState >= 3) return "ready";
  if (!media.paused) return "buffering";
  return "idle";
}

export function createHTMLMediaPlaybackConnection({
  media,
  hasCurrentMedia,
  restartCurrent,
}) {
  if (!media?.addEventListener || !media?.removeEventListener) {
    throw new Error("Playback Connection requires an HTMLMediaElement-like provider");
  }
  if (typeof media.play !== "function" || typeof media.pause !== "function") {
    throw new Error("Playback Connection requires play/pause provider methods");
  }
  if (typeof hasCurrentMedia !== "function") {
    throw new Error("Playback Connection requires hasCurrentMedia()");
  }
  if (typeof restartCurrent !== "function") {
    throw new Error("Playback Connection requires restartCurrent()");
  }

  const listeners = new Set();
  let pendingEffect = null;
  let lastProjection = null;

  function observedState() {
    if (!hasCurrentMedia()) {
      return PlaybackState.noMedia();
    }

    if (media.ended) {
      return PlaybackState.ended({
        restartAvailable: true,
      });
    }

    return PlaybackState.media({
      playRequested: !media.paused,
      playPauseAvailable: true,
      requestPending: false,
      phase: phaseFromMedia(media),
    });
  }

  function reconciledState() {
    const observed = observedState();

    if (!pendingEffect) {
      return observed;
    }

    if (pendingEffect === PlaybackEffect.RequestPlay) {
      if (
        observed.kind === "media" &&
        observed.playRequested
      ) {
        pendingEffect = null;
        return observed;
      }

      if (observed.kind === "media") {
        return PlaybackState.media({
          ...observed,
          playRequested: true,
          requestPending: true,
        });
      }
    }

    if (pendingEffect === PlaybackEffect.RequestPause) {
      if (
        observed.kind === "media" &&
        !observed.playRequested
      ) {
        pendingEffect = null;
        return observed;
      }

      if (observed.kind === "media") {
        return PlaybackState.media({
          ...observed,
          playRequested: false,
          requestPending: true,
        });
      }
    }

    if (pendingEffect === PlaybackEffect.RestartCurrent) {
      if (observed.kind !== "ended") {
        pendingEffect = null;
        return observed;
      }

      return PlaybackState.ended({
        ...observed,
        restartPending: true,
      });
    }

    pendingEffect = null;
    return observed;
  }

  function formProjection() {
    const state = reconciledState();

    return Object.freeze({
      state,
      control: presentPlaybackControl(state),
    });
  }

  function publish() {
    const projection = formProjection();
    lastProjection = projection;

    for (const listener of listeners) {
      listener(projection);
    }

    return projection;
  }

  function currentProjection() {
    return publish();
  }

  async function realize(effect) {
    if (effect === PlaybackEffect.RequestPlay) {
      await media.play();
      return;
    }

    if (effect === PlaybackEffect.RequestPause) {
      media.pause();
      return;
    }

    if (effect === PlaybackEffect.RestartCurrent) {
      await restartCurrent(media);
      await media.play();
      return;
    }

    throw new Error(
      `vexstream.playback: unknown Effect ${effect?.ref ?? effect}`,
    );
  }

  async function activate(control) {
    // Re-read provider truth here. The Control meaning rendered earlier may now be stale.
    const current = reconciledState();
    const act = resolvePlaybackControl(current, control);

    if (!act) {
      publish();
      return null;
    }

    const relationship =
      vexStreamPlaybackBehavior.resolve(current, act);

    if (!relationship) {
      publish();
      return null;
    }

    pendingEffect = relationship.effect;
    publish();

    try {
      await realize(relationship.effect);
    } catch (failure) {
      pendingEffect = null;
      publish();
      throw failure;
    }

    publish();
    return relationship;
  }

  const providerEvents = [
    "play",
    "playing",
    "pause",
    "ended",
    "emptied",
    "loadedmetadata",
    "canplay",
    "waiting",
  ];

  const onProviderEvent = () => {
    publish();
  };

  for (const event of providerEvents) {
    media.addEventListener(event, onProviderEvent);
  }

  publish();

  return Object.freeze({
    get currentProjection() {
      return lastProjection ?? currentProjection();
    },

    activate,

    subscribe(listener) {
      if (typeof listener !== "function") {
        throw new Error("Playback Connection subscriber must be a function");
      }
      listeners.add(listener);
      listener(lastProjection ?? currentProjection());
      return () => listeners.delete(listener);
    },

    close() {
      for (const event of providerEvents) {
        media.removeEventListener(event, onProviderEvent);
      }
      listeners.clear();
    },
  });
}
