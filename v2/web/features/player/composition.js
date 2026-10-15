// [VXG RealForever]
//
// Higher composition for human listening actions that legitimately span Queue + Playback.
// This is feature-local composition, not a fifth semantic family and not a generic command bus.

import { PlaybackControl } from "../playback/behavior.js";
import { QueueControl, currentQueueRef } from "../queue/behavior.js";

export function createPlayerComposition({
  queue,
  playback,
  media,
  resolveMedia,
  defaultContextRefs,
  onCurrentMedia = () => {},
  previousRestartThresholdSeconds = 4,
}) {
  if (!queue || !playback || !media) throw new Error("Player composition requires Queue, Playback, and media provider");
  if (typeof resolveMedia !== "function") throw new Error("Player composition requires resolveMedia(ref)");
  if (typeof defaultContextRefs !== "function") throw new Error("Player composition requires defaultContextRefs()");

  function queueState() {
    return queue.currentProjection.state;
  }

  function currentRef() {
    return currentQueueRef(queueState());
  }

  function currentItem() {
    const ref = currentRef();
    return ref ? resolveMedia(ref) : null;
  }

  function startSeconds(item) {
    return Math.max(0, Number(item?.playbackStartSeconds) || 0);
  }

  function playPausePresentation() {
    const playbackControl = playback.currentProjection.control;
    if (currentRef()) return playbackControl;

    const context = defaultContextRefs();
    if (context.length) {
      return Object.freeze({
        ...playbackControl,
        enabled: true,
        textRef: "playback.play",
        iconRef: "play",
        compositionRef: "player.seed-default-context",
      });
    }

    return playbackControl;
  }

  async function loadCurrent({ autoplay = true } = {}) {
    const item = currentItem();
    if (!item) return null;

    if (media.dataset.mediaRef !== item.ref) {
      media.pause();
      media.dataset.mediaRef = item.ref;
      media.src = item.url;
      try {
        media.currentTime = startSeconds(item);
      } catch {}
    }

    onCurrentMedia(item, queue.currentProjection);
    playback.refresh();

    if (autoplay) {
      const control = playback.currentProjection.control;
      if (control.enabled) {
        await playback.activate(PlaybackControl.PlayPause);
      }
    }

    return item;
  }

  async function activatePlayPause() {
    if (!currentRef()) {
      const context = defaultContextRefs();
      if (!context.length) return null;
      queue.replaceContext(context, 0);
      await loadCurrent({ autoplay: true });
      return Object.freeze({ compositionRef: "player.seed-default-context" });
    }

    return playback.activate(PlaybackControl.PlayPause);
  }

  async function playContext(refs, startIndex = 0) {
    queue.replaceContext(refs, startIndex);
    return loadCurrent({ autoplay: true });
  }

  async function playTrack(ref, contextRefs = [ref]) {
    const refs = [...contextRefs];
    const index = refs.indexOf(ref);
    return playContext(index >= 0 ? refs : [ref], index >= 0 ? index : 0);
  }

  async function next() {
    const relationship = queue.activate(QueueControl.Next);
    if (!relationship) return null;
    await loadCurrent({ autoplay: true });
    return relationship;
  }

  async function previous() {
    const item = currentItem();
    if (!item) return null;

    const start = startSeconds(item);
    if (Number(media.currentTime) > start + previousRestartThresholdSeconds) {
      media.currentTime = start;
      return Object.freeze({ compositionRef: "player.restart-current" });
    }

    const relationship = queue.activate(QueueControl.Previous);
    if (relationship) {
      await loadCurrent({ autoplay: true });
      return relationship;
    }

    media.currentTime = start;
    return Object.freeze({ compositionRef: "player.restart-current-at-queue-start" });
  }

  function toggleShuffle() {
    return queue.activate(QueueControl.Shuffle);
  }

  function toggleRepeat() {
    return queue.activate(QueueControl.Repeat);
  }

  function clearAfterCurrent() {
    return queue.activate(QueueControl.ClearAfterCurrent);
  }

  const onEnded = () => {
    void next();
  };
  media.addEventListener("ended", onEnded);

  return Object.freeze({
    get currentItem() {
      return currentItem();
    },
    get playPausePresentation() {
      return playPausePresentation();
    },
    activatePlayPause,
    playContext,
    playTrack,
    next,
    previous,
    toggleShuffle,
    toggleRepeat,
    clearAfterCurrent,
    append: refs => queue.append(refs),
    playNext: refs => queue.playNext(refs),
    moveQueue: (from, to) => queue.move(from, to),
    removeQueue: index => queue.remove(index),
    close() {
      media.removeEventListener("ended", onEnded);
    },
  });
}
