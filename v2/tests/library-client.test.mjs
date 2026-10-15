import assert from "node:assert/strict";
import test from "node:test";

import {
  loadLibraryCatalog,
  mediaURLFor,
  projectLibraryCatalog,
} from "../web/features/library/client.js";

test("catalog projects opaque media refs without filesystem custody", () => {
  const catalog = projectLibraryCatalog({
    tracks: [{
      id: "track / one",
      path: "C:/private/music/one.mp3",
      source: "C:/private/music",
      filename: "one.mp3",
      title: "Human title",
      artist: "Human artist",
      sourceTitle: "Source title",
      playbackStart: 1.25,
      playbackEnd: 8.5,
    }],
  });

  assert.deepEqual(catalog, [{
    ref: "track / one",
    url: "/media/track%20%2F%20one",
    title: "Human title",
    artist: "Human artist",
    album: "",
    genre: "",
    year: "",
    providerId: "",
    sourceURL: "",
    channel: "",
    tags: [],
    categories: [],
    description: "",
    addedAt: "",
    durationSeconds: 0,
    playbackStartSeconds: 1.25,
    playbackEndSeconds: 8.5,
    thumbnailURL: "",
    sourceTitle: "Source title",
    sourceArtist: "",
    sourceAlbum: "",
    sourceGenre: "",
  }]);
  assert.equal("path" in catalog[0], false);
  assert.equal("source" in catalog[0], false);
  assert.equal("filename" in catalog[0], false);
});

test("invalid playback end does not become a playback-window claim", () => {
  const [track] = projectLibraryCatalog({
    tracks: [{ id: "a", playbackStart: 5, playbackEnd: 2 }],
  });
  assert.equal(track.playbackStartSeconds, 5);
  assert.equal(track.playbackEndSeconds, null);
});

test("catalog identity is required", () => {
  assert.throws(() => projectLibraryCatalog({ tracks: [{}] }), /missing id/);
  assert.throws(() => projectLibraryCatalog({}), /tracks\[\]/);
  assert.throws(() => mediaURLFor(""), /trackRef/);
});

test("load uses the read-only local catalog route", async () => {
  let request;
  const catalog = await loadLibraryCatalog({
    request: async (url, options) => {
      request = { url, options };
      return {
        ok: true,
        status: 200,
        json: async () => ({ tracks: [{ id: "a", title: "A" }] }),
      };
    },
  });
  assert.equal(request.url, "/api/library");
  assert.equal(request.options.method, "GET");
  assert.equal(request.options.headers.Accept, "application/json");
  assert.equal(catalog[0].ref, "a");
  assert.equal(catalog[0].url, "/media/a");
});

test("failed HTTP response remains a load failure", async () => {
  await assert.rejects(
    () => loadLibraryCatalog({ request: async () => ({ ok: false, status: 503 }) }),
    /503/,
  );
});
