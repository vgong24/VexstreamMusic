// [VXG RealForever]
import {
  PlaybackAct,
  PlaybackControl,
  resolvePlaybackControl,
} from "./behavior.js";

export function presentPlaybackControl(
  currentState,
  control = PlaybackControl.PlayPause,
) {
  const act = resolvePlaybackControl(currentState, control);

  if (act === PlaybackAct.Play) {
    const restart = currentState.kind === "ended";
    return Object.freeze({
      control,
      act,
      enabled: true,
      textRef: restart ? "playback.restart" : "playback.play",
      iconRef: "play",
    });
  }

  if (act === PlaybackAct.Pause) {
    return Object.freeze({
      control,
      act,
      enabled: true,
      textRef: "playback.pause",
      iconRef: "pause",
    });
  }

  return Object.freeze({
    control,
    act: null,
    enabled: false,
    textRef: "playback.unavailable",
    iconRef: "play",
  });
}
