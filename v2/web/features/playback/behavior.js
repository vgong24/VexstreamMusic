// [VXG RealForever]
import {
  defineBehaviorFamily,
  semanticToken,
  stateOf,
} from "../../architecture/behavior.js";

export const VexStreamPlaybackFamily = "vexstream.playback";

export const PlaybackControl = Object.freeze({
  PlayPause: semanticToken(
    VexStreamPlaybackFamily,
    "control",
    "play-pause",
  ),
});

export const PlaybackAct = Object.freeze({
  Play: semanticToken(VexStreamPlaybackFamily, "act", "play"),
  Pause: semanticToken(VexStreamPlaybackFamily, "act", "pause"),
});

export const PlaybackEffect = Object.freeze({
  RequestPlay: semanticToken(
    VexStreamPlaybackFamily,
    "effect",
    "request-play",
  ),
  RequestPause: semanticToken(
    VexStreamPlaybackFamily,
    "effect",
    "request-pause",
  ),
  RestartCurrent: semanticToken(
    VexStreamPlaybackFamily,
    "effect",
    "restart-current",
  ),
});

export const PlaybackState = Object.freeze({
  noMedia() {
    return stateOf(VexStreamPlaybackFamily, "no-media");
  },

  media({
    playRequested,
    playPauseAvailable = true,
    requestPending = false,
    phase = "ready",
  }) {
    return stateOf(VexStreamPlaybackFamily, "media", {
      playRequested: Boolean(playRequested),
      playPauseAvailable: Boolean(playPauseAvailable),
      requestPending: Boolean(requestPending),
      phase,
    });
  },

  ended({
    restartAvailable = true,
    restartPending = false,
  } = {}) {
    return stateOf(VexStreamPlaybackFamily, "ended", {
      restartAvailable: Boolean(restartAvailable),
      restartPending: Boolean(restartPending),
    });
  },
});

export const vexStreamPlaybackBehavior = defineBehaviorFamily({
  ref: VexStreamPlaybackFamily,
  relationships: [
    {
      stateKind: "media",
      act: PlaybackAct.Play,
      effect: PlaybackEffect.RequestPlay,
      when: current =>
        !current.playRequested &&
        current.playPauseAvailable,
    },
    {
      stateKind: "media",
      act: PlaybackAct.Pause,
      effect: PlaybackEffect.RequestPause,
      when: current =>
        current.playRequested &&
        current.playPauseAvailable,
    },
    {
      stateKind: "ended",
      act: PlaybackAct.Play,
      effect: PlaybackEffect.RestartCurrent,
      when: current =>
        current.restartAvailable &&
        !current.restartPending,
    },
  ],
});

export function resolvePlaybackControl(
  currentState,
  control = PlaybackControl.PlayPause,
) {
  if (
    control?.familyRef !== VexStreamPlaybackFamily ||
    control?.role !== "control"
  ) {
    throw new Error("vexstream.playback: Control belongs to another family or role");
  }

  switch (currentState.kind) {
    case "media":
      if (!currentState.playPauseAvailable) return null;
      return currentState.playRequested
        ? PlaybackAct.Pause
        : PlaybackAct.Play;

    case "ended":
      return currentState.restartAvailable && !currentState.restartPending
        ? PlaybackAct.Play
        : null;

    case "no-media":
      // V1 currently seeds Queue/library context here. V2 keeps that composition
      // outside the Playback family instead of hiding Queue meaning in Playback.
      return null;

    default:
      throw new Error(
        `vexstream.playback: unknown State kind ${currentState.kind}`,
      );
  }
}
