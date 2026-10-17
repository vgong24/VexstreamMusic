import assert from "node:assert/strict";
import test from "node:test";

import {
  buildImportPlan,
  formImportReview,
} from "../web/features/imports/review.js";

test("review preserves V3 human editing defaults without starting acquisition", () => {
  const review = formImportReview(
    {
      inspection: {
        url: "https://example/song",
        title: "Source",
        channel: "Uploader",
        artist: "Artist",
        album: "Album",
        genre: "",
        categories: ["Dream Pop"],
        hasSections: true,
        sections: [
          { index: 1, title: "One", start: 0, end: 30, selected: true },
          { index: 2, title: "Two", start: 30, end: 60, selected: true },
        ],
      },
      items: [
        {
          kind: "whole",
          matches: [{ trackId: "track-a", score: 90 }],
        },
      ],
      advisory: true,
    },
    [" /music ", "/music", "/archive"],
  );

  assert.equal(review.defaultMode, "split");
  assert.equal(review.collection, "Album");
  assert.equal(review.artistOverride, "Artist");
  assert.equal(review.genre, "Dream Pop");
  assert.deepEqual(review.destinations, ["/music", "/archive"]);
  assert.equal(review.duplicateItemCount, 1);
  assert.equal(review.advisory, true);
});

test("plan uses only admitted destination and explicit section edits", () => {
  const review = formImportReview(
    {
      inspection: {
        url: "https://example/song",
        title: "Source",
        hasSections: true,
        sections: [
          { index: 1, title: "One", start: 0, end: 30, selected: true },
          { index: 2, title: "Two", start: 30, end: 60, selected: true },
        ],
      },
      items: [],
      advisory: true,
    },
    ["/music"],
  );

  const plan = buildImportPlan(review, {
    destination: "/music",
    mode: "both",
    collection: "Collection",
    artistOverride: "Artist",
    genre: "Genre",
    sections: [
      { index: 1, title: "Edited One", artist: "A", selected: true },
      { index: 2, selected: false },
    ],
  });

  assert.equal(plan.destination, "/music");
  assert.equal(plan.mode, "both");
  assert.equal(plan.wholeTitle, "Source");
  assert.equal(plan.sections[0].sourceTitle, "One");
  assert.equal(plan.sections[0].title, "Edited One");
  assert.equal(plan.sections[0].artist, "A");
  assert.equal(plan.sections[1].selected, false);
});

test("source without multiple sections is always a whole-track import", () => {
  const review = formImportReview(
    {
      inspection: {
        url: "https://example/song",
        title: "Source",
        hasSections: false,
        sections: [],
      },
      items: [],
      advisory: true,
    },
    ["/music"],
  );

  const plan = buildImportPlan(review, {
    destination: "/music",
    mode: "split",
  });

  assert.equal(review.splitAvailable, false);
  assert.equal(plan.mode, "whole");
});

test("plan refuses stale destination and empty split selection", () => {
  const review = formImportReview(
    {
      inspection: {
        url: "https://example/song",
        title: "Source",
        hasSections: true,
        sections: [
          { index: 1, title: "One", start: 0, end: 30, selected: true },
          { index: 2, title: "Two", start: 30, end: 60, selected: true },
        ],
      },
      items: [],
      advisory: true,
    },
    ["/music"],
  );

  assert.throws(
    () =>
      buildImportPlan(review, {
        destination: "/removed",
      }),
    /no longer admitted/,
  );

  assert.throws(
    () =>
      buildImportPlan(review, {
        destination: "/music",
        mode: "split",
        sections: [
          { index: 1, selected: false },
          { index: 2, selected: false },
        ],
      }),
    /Select at least one/,
  );
});
