import assert from "node:assert/strict";
import test from "node:test";

import {
  PlaybackAct,
  PlaybackControl,
} from "../web/features/playback/behavior.js";
import { createExploreComposition } from "../web/features/explore/composition.js";
import { createExploreSession } from "../web/features/explore/session.js";

function frame(title, candidates = []) {
  return {
    schemaVersion: "vexstream.discovery-candidates/v2",
    seed: { title },
    candidates,
  };
}

function playbackFixture(initialAct) {
  let act = initialAct;
  const calls = [];

  return {
    get currentProjection() {
      return {
        control: {
          enabled: true,
          act,
        },
      };
    },

    async activate(control) {
      calls.push(control);
      act =
        act === PlaybackAct.Pause
          ? PlaybackAct.Play
          : PlaybackAct.Pause;
    },

    calls,
  };
}

test("preview pauses active local Playback and Clear resumes through Playback Control", async () => {
  const session = createExploreSession();
  session.replace(
    frame("Seed", [
      { providerId: "abc", title: "Candidate" },
    ]),
  );
  const playback = playbackFixture(PlaybackAct.Pause);
  const explore = createExploreComposition({ session, playback });

  const preview = await explore.preview(0);

  assert.match(preview.url, /youtube\.com\/embed\/abc/);
  assert.equal(session.currentProjection.previewIndex, 0);
  assert.deepEqual(playback.calls, [PlaybackControl.PlayPause]);

  await explore.clear();

  assert.equal(session.currentProjection.kind, "empty");
  assert.deepEqual(
    playback.calls,
    [
      PlaybackControl.PlayPause,
      PlaybackControl.PlayPause,
    ],
  );
});

test("branch and Back stop preview without spuriously resuming local Playback", async () => {
  const session = createExploreSession();
  session.replace(
    frame("A", [{ providerId: "one", title: "One" }]),
  );
  const playback = playbackFixture(PlaybackAct.Pause);
  const explore = createExploreComposition({
    session,
    playback,
    requestCandidate: async candidate =>
      frame(candidate.title, [
        { providerId: "two", title: "Two" },
      ]),
  });

  await explore.preview(0);
  assert.equal(playback.calls.length, 1);

  await explore.branchFromCandidate(0);
  assert.equal(session.currentProjection.frame.seed.title, "One");
  assert.equal(playback.calls.length, 1);

  await explore.back();
  assert.equal(session.currentProjection.frame.seed.title, "A");
  assert.equal(playback.calls.length, 1);
});

test("preview does not manufacture pause/resume when local Playback was paused", async () => {
  const session = createExploreSession();
  session.replace(
    frame("A", [{ providerId: "one", title: "One" }]),
  );
  const playback = playbackFixture(PlaybackAct.Play);
  const explore = createExploreComposition({ session, playback });

  await explore.preview(0);
  await explore.clear();

  assert.deepEqual(playback.calls, []);
});

test("replaceFromTrack keeps request payload outside Explore Session semantics", async () => {
  const session = createExploreSession();
  const playback = playbackFixture(PlaybackAct.Play);
  let seen;

  const explore = createExploreComposition({
    session,
    playback,
    requestTrack: async ref => {
      seen = ref;
      return frame("Library Seed", []);
    },
  });

  await explore.replaceFromTrack("track-a");

  assert.equal(seen, "track-a");
  assert.equal(
    session.currentProjection.frame.seed.title,
    "Library Seed",
  );
});
