// [VXG RealForever]
import assert from "node:assert/strict";
import test from "node:test";

import {
  PlaybackAct,
  PlaybackControl,
  PlaybackEffect,
  PlaybackState,
  VexStreamPlaybackFamily,
  resolvePlaybackControl,
  vexStreamPlaybackBehavior,
} from "../web/features/playback/behavior.js";
import { presentPlaybackControl } from "../web/features/playback/presentation.js";
import { resolveText } from "../web/foundation/text.js";
import { en } from "../web/locales/en.js";
import { stateOf } from "../web/architecture/behavior.js";

test("PlayPause means Play from paused/current media", () => {
  const state = PlaybackState.media({ playRequested: false });

  assert.equal(
    resolvePlaybackControl(state, PlaybackControl.PlayPause),
    PlaybackAct.Play,
  );

  const relationship =
    vexStreamPlaybackBehavior.resolve(state, PlaybackAct.Play);

  assert.equal(relationship.effect, PlaybackEffect.RequestPlay);
  assert.equal(relationship.admittedStateKind, "media");
});

test("PlayPause means Pause from play-requested media", () => {
  const state = PlaybackState.media({ playRequested: true });

  assert.equal(resolvePlaybackControl(state), PlaybackAct.Pause);
  assert.equal(
    vexStreamPlaybackBehavior.resolve(state, PlaybackAct.Pause).effect,
    PlaybackEffect.RequestPause,
  );
});

test("Play from ended media requests restart rather than generic play", () => {
  const state = PlaybackState.ended();

  assert.equal(resolvePlaybackControl(state), PlaybackAct.Play);
  assert.equal(
    vexStreamPlaybackBehavior.resolve(state, PlaybackAct.Play).effect,
    PlaybackEffect.RestartCurrent,
  );
});

test("no-media does not silently make Playback own Queue selection", () => {
  const state = PlaybackState.noMedia();

  assert.equal(resolvePlaybackControl(state), null);
  assert.equal(presentPlaybackControl(state).enabled, false);
});

test("presentation emits stable text identity before localization", () => {
  const state = PlaybackState.media({ playRequested: true });
  const presentation = presentPlaybackControl(state);

  assert.equal(presentation.textRef, "playback.pause");
  assert.equal(resolveText(en, presentation.textRef), "Pause");
});

test("family boundary rejects a State from another semantic family", () => {
  const foreign = stateOf("vexstream.queue", "ready");

  assert.throws(
    () => vexStreamPlaybackBehavior.resolve(foreign, PlaybackAct.Play),
    /State belongs to another family/,
  );
});

test("playback family remains explicitly addressable", () => {
  assert.equal(vexStreamPlaybackBehavior.ref, VexStreamPlaybackFamily);
});
