import assert from "node:assert/strict";
import test from "node:test";

import { createExploreSession } from "../web/features/explore/session.js";

function frame(title, candidates = []) {
  return {
    seed: { title },
    basis: { method: "SEARCH_DERIVED_MULTI_AXIS" },
    candidates,
  };
}

test("replace forms a temporary current Explore frame without mutating input", () => {
  const source = frame("Track A", [{ providerId: "one", title: "One" }]);
  const session = createExploreSession();
  const projected = session.replace(source);

  assert.equal(projected.kind, "active");
  assert.equal(projected.frame.seed.title, "Track A");
  assert.equal(projected.canGoBack, false);
  assert.equal(projected.previewCandidate, null);

  source.seed.title = "mutated";
  source.candidates[0].title = "mutated";
  assert.equal(session.currentProjection.frame.seed.title, "Track A");
  assert.equal(session.currentProjection.frame.candidates[0].title, "One");
});

test("branch appends from current frame and Back returns without rewriting history", () => {
  const session = createExploreSession();
  session.replace(frame("A"));
  session.branch(frame("B"));
  session.branch(frame("C"));

  assert.equal(session.currentProjection.frame.seed.title, "C");
  assert.equal(session.currentProjection.frames.length, 3);
  session.back();
  assert.equal(session.currentProjection.frame.seed.title, "B");
  assert.equal(session.currentProjection.frames.length, 3);
  assert.equal(session.currentProjection.canGoBack, true);
});

test("branch after Back discards abandoned forward history", () => {
  const session = createExploreSession();
  session.replace(frame("A"));
  session.branch(frame("B"));
  session.branch(frame("C"));
  session.back();
  session.branch(frame("D"));

  assert.deepEqual(
    session.currentProjection.frames.map(value => value.seed.title),
    ["A", "B", "D"],
  );
  assert.equal(session.currentProjection.frameIndex, 2);
});

test("preview selection belongs to current frame and resets on navigation", () => {
  const session = createExploreSession();
  session.replace(frame("A", [
    { providerId: "one", title: "One" },
    { providerId: "two", title: "Two" },
  ]));
  session.selectPreview(1);
  assert.equal(session.currentProjection.previewCandidate.providerId, "two");

  session.branch(frame("B", [{ providerId: "three", title: "Three" }]));
  assert.equal(session.currentProjection.previewCandidate, null);
  session.selectPreview(0);
  session.back();
  assert.equal(session.currentProjection.previewCandidate, null);
});

test("invalid preview selection fails instead of silently selecting another candidate", () => {
  const session = createExploreSession();
  session.replace(frame("A", [{ providerId: "one" }]));
  assert.throws(() => session.selectPreview(2), /must address current candidates/);
  assert.equal(session.currentProjection.previewIndex, -1);
});

test("clear destroys only the temporary Explore session", () => {
  const session = createExploreSession();
  session.replace(frame("A", [{ providerId: "one" }]));
  session.selectPreview(0);
  const cleared = session.clear();

  assert.equal(cleared.kind, "empty");
  assert.deepEqual(cleared.frames, []);
  assert.equal(cleared.frameIndex, -1);
  assert.equal(cleared.previewIndex, -1);
});

test("subscribers observe session projections, not provider side effects", () => {
  const session = createExploreSession();
  const seen = [];
  const unsubscribe = session.subscribe(value => {
    seen.push([value.kind, value.frameIndex]);
  });
  session.replace(frame("A"));
  session.branch(frame("B"));
  session.back();
  session.clear();
  unsubscribe();

  assert.deepEqual(seen, [
    ["empty", -1],
    ["active", 0],
    ["active", 1],
    ["active", 0],
    ["empty", -1],
  ]);
});
