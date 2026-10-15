// [VXG RealForever]
// Browser consumer for durable desktop Playlist truth.
//
// Playlist owns durable ordered references. Library owns whether a ref resolves today.
// Queue owns the playable listening context formed from those truths right now.

export function projectPlaylists(payload) {
  if (!payload || !Array.isArray(payload.playlists)) {
    throw new Error("playlist response requires playlists[]");
  }

  return Object.freeze(
    payload.playlists.map((raw, index) => {
      const id = String(raw?.id ?? "").trim();
      const name = String(raw?.name ?? "").trim();
      if (!id) throw new Error(`playlist ${index} is missing id`);
      if (!name) throw new Error(`playlist ${index} is missing name`);
      if (!Array.isArray(raw.trackIds)) {
        throw new Error(`playlist ${index} requires trackIds[]`);
      }

      const trackRefs = Object.freeze(
        raw.trackIds.map((value, refIndex) => {
          const ref = String(value ?? "").trim();
          if (!ref) {
            throw new Error(`playlist ${index} trackIds[${refIndex}] is empty`);
          }
          return ref;
        }),
      );

      return Object.freeze({
        id,
        name,
        trackRefs,
        createdAt: stringValue(raw.createdAt),
        updatedAt: stringValue(raw.updatedAt),
      });
    }),
  );
}

export function resolvePlaylistAvailability(playlist, catalog) {
  const byRef = new Map(catalog.map(track => [track.ref, track]));
  const entries = Object.freeze(
    playlist.trackRefs.map(ref => {
      const track = byRef.get(ref) ?? null;
      return Object.freeze({
        ref,
        available: track !== null,
        track,
      });
    }),
  );

  return Object.freeze({
    playlist,
    entries,
    playableRefs: Object.freeze(
      entries.filter(entry => entry.available).map(entry => entry.ref),
    ),
    missingRefs: Object.freeze(
      entries.filter(entry => !entry.available).map(entry => entry.ref),
    ),
  });
}

export async function loadPlaylists({ request = globalThis.fetch } = {}) {
  if (typeof request !== "function") {
    throw new Error("playlists require fetch-like request function");
  }
  const response = await request("/api/playlists", {
    method: "GET",
    headers: { Accept: "application/json" },
  });
  if (!response?.ok) {
    throw new Error(`playlist request failed: ${response?.status ?? "unknown"}`);
  }
  return projectPlaylists(await response.json());
}

function stringValue(value) {
  return typeof value === "string" ? value : "";
}
