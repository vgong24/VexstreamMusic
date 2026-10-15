// [VXG RealForever]
import assert from "node:assert/strict";
import test from "node:test";

import {
  PlaybackAct,
  PlaybackControl,
  PlaybackEffect,
} from "../web/features/playback/behavior.js";
import {
  createHTMLMediaPlaybackConnection,
} from "../web/features/playback/connection.js";

class FakeMedia extends EventTarget {
  constructor() {
    super();
    this.paused = true;
    this.ended = false;
    this.readyState = 4;
    this.playCalls = 0;
    this.pauseCalls = 0;
    this.rejectPlay = null;
  }

  async play() {
    this.playCalls += 1;

    if (this.rejectPlay) {
      throw this.rejectPlay;
    }

    this.paused = false;
    this.ended = false;
    this.dispatchEvent(new Event("play"));
  }

  pause() {
    this.pauseCalls += 1;
    this.paused = true;
    this.dispatchEvent(new Event("pause"));
  }
}

function connectionFor(media, options = {}) {
  return createHTMLMediaPlaybackConnection({
    media,
    hasCurrentMedia:
      options.hasCurrentMedia ??
      (() => true),
    restartCurrent:
      options.restartCurrent ??
      (async currentMedia => {
        currentMedia.ended = false;
      }),
  });
}

test("Connection projects real paused provider truth into Play", () => {
  const media = new FakeMedia();
  const connection = connectionFor(media);

  assert.equal(
    connection.currentProjection.control.act,
    PlaybackAct.Play,
  );
  assert.equal(
    connection.currentProjection.control.textRef,
    "playback.play",
  );

  connection.close();
});

test("activation re-resolves stale PlayPause meaning from current provider truth", async () => {
  const media = new FakeMedia();
  const connection = connectionFor(media);

  // Render-time projection says Play.
  assert.equal(
    connection.currentProjection.control.act,
    PlaybackAct.Play,
  );

  // Provider changes before the human activates the same stable Control.
  media.paused = false;
  media.dispatchEvent(new Event("play"));

  const relationship =
    await connection.activate(
      PlaybackControl.PlayPause,
    );

  assert.equal(relationship.act, PlaybackAct.Pause);
  assert.equal(
    relationship.effect,
    PlaybackEffect.RequestPause,
  );
  assert.equal(media.pauseCalls, 1);
  assert.equal(
    connection.currentProjection.control.act,
    PlaybackAct.Play,
  );

  connection.close();
});

test("Play effect observes provider acknowledgement instead of claiming it early", async () => {
  const media = new FakeMedia();
  const connection = connectionFor(media);

  const relationship =
    await connection.activate(
      PlaybackControl.PlayPause,
    );

  assert.equal(
    relationship.effect,
    PlaybackEffect.RequestPlay,
  );
  assert.equal(media.playCalls, 1);
  assert.equal(
    connection.currentProjection.state.playRequested,
    true,
  );
  assert.equal(
    connection.currentProjection.state.requestPending,
    false,
  );

  connection.close();
});

test("provider play failure returns to observed paused truth", async () => {
  const media = new FakeMedia();
  media.rejectPlay = new Error("provider rejected play");
  const connection = connectionFor(media);

  await assert.rejects(
    () => connection.activate(PlaybackControl.PlayPause),
    /provider rejected play/,
  );

  assert.equal(
    connection.currentProjection.state.playRequested,
    false,
  );
  assert.equal(
    connection.currentProjection.state.requestPending,
    false,
  );

  connection.close();
});

test("Ended Play delegates restart position to the concrete provider owner", async () => {
  const media = new FakeMedia();
  media.ended = true;
  media.paused = true;
  let restartCalls = 0;

  const connection = connectionFor(media, {
    restartCurrent: async currentMedia => {
      restartCalls += 1;
      currentMedia.ended = false;
    },
  });

  const relationship =
    await connection.activate(
      PlaybackControl.PlayPause,
    );

  assert.equal(
    relationship.effect,
    PlaybackEffect.RestartCurrent,
  );
  assert.equal(restartCalls, 1);
  assert.equal(media.playCalls, 1);

  connection.close();
});

test("No current media remains outside Playback queue-selection ownership", async () => {
  const media = new FakeMedia();
  const connection = connectionFor(media, {
    hasCurrentMedia: () => false,
  });

  assert.equal(
    connection.currentProjection.state.kind,
    "no-media",
  );

  const relationship =
    await connection.activate(
      PlaybackControl.PlayPause,
    );

  assert.equal(relationship, null);
  assert.equal(media.playCalls, 0);

  connection.close();
});
