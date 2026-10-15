import assert from "node:assert/strict";
import test from "node:test";

import {
  loadPlaylists,
  projectPlaylists,
  resolvePlaylistAvailability,
} from "../web/features/playlists/client.js";

test("playlist projection preserves durable order including unresolved refs", () => {
  const [playlist] = projectPlaylists({
    playlists: [{
      id: "road-trip",
      name: "Road Trip",
      trackIds: ["track-a", "missing", "track-b"],
      createdAt: "c",
      updatedAt: "u",
    }],
  });

  assert.deepEqual(playlist.trackRefs, ["track-a", "missing", "track-b"]);
  assert.equal(playlist.name, "Road Trip");
});

test("availability derives from current Library without rewriting Playlist truth", () => {
  const [playlist] = projectPlaylists({
    playlists: [{
      id: "road-trip",
      name: "Road Trip",
      trackIds: ["track-a", "missing", "track-b"],
    }],
  });
  const catalog = [
    Object.freeze({ ref: "track-a", title: "A" }),
    Object.freeze({ ref: "track-b", title: "B" }),
  ];

  const resolved = resolvePlaylistAvailability(playlist, catalog);
  assert.deepEqual(resolved.playableRefs, ["track-a", "track-b"]);
  assert.deepEqual(resolved.missingRefs, ["missing"]);
  assert.deepEqual(playlist.trackRefs, ["track-a", "missing", "track-b"]);
  assert.equal(resolved.entries[1].available, false);
  assert.equal(resolved.entries[1].track, null);
});

test("playlist identity, name and refs are required", () => {
  assert.throws(() => projectPlaylists({ playlists: [{}] }), /missing id/);
  assert.throws(
    () => projectPlaylists({ playlists: [{ id: "p", name: " ", trackIds: [] }] }),
    /missing name/,
  );
  assert.throws(
    () => projectPlaylists({ playlists: [{ id: "p", name: "Mix" }] }),
    /trackIds/,
  );
  assert.throws(
    () => projectPlaylists({ playlists: [{ id: "p", name: "Mix", trackIds: [""] }] }),
    /is empty/,
  );
});

test("load uses read-only playlist route", async () => {
  let seen;
  const rows = await loadPlaylists({
    request: async (url, options) => {
      seen = { url, options };
      return {
        ok: true,
        status: 200,
        json: async () => ({
          playlists: [{ id: "p", name: "Mix", trackIds: ["a"] }],
        }),
      };
    },
  });

  assert.equal(seen.url, "/api/playlists");
  assert.equal(seen.options.method, "GET");
  assert.equal(seen.options.headers.Accept, "application/json");
  assert.equal(rows[0].id, "p");
});

test("failed playlist HTTP response remains a load failure", async () => {
  await assert.rejects(
    () => loadPlaylists({ request: async () => ({ ok: false, status: 503 }) }),
    /503/,
  );
});
