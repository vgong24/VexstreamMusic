// [VXG RealForever]
// Browser consumer for the local read-only V4 Library projection.
// Filesystem paths/source-folder custody remain server-side; song metadata remains data, not TextRef.

export function mediaURLFor(trackRef) {
  const ref = String(trackRef ?? "").trim();
  if (!ref) throw new Error("media URL requires trackRef");
  return `/media/${encodeURIComponent(ref)}`;
}

export function projectLibraryCatalog(payload) {
  if (!payload || !Array.isArray(payload.tracks)) {
    throw new Error("library catalog response requires tracks[]");
  }

  return Object.freeze(
    payload.tracks.map((raw, index) => {
      const ref = String(raw?.id ?? "").trim();
      if (!ref) throw new Error(`library track ${index} is missing id`);

      const start = finiteNonNegative(raw.playbackStart);
      const end = finitePositive(raw.playbackEnd);

      return Object.freeze({
        ref,
        url: mediaURLFor(ref),
        title: stringValue(raw.title),
        artist: stringValue(raw.artist),
        album: stringValue(raw.album),
        genre: stringValue(raw.genre),
        year: stringValue(raw.year),
        providerId: stringValue(raw.providerId),
        sourceURL: stringValue(raw.sourceUrl),
        channel: stringValue(raw.channel),
        tags: Object.freeze(stringArray(raw.tags)),
        categories: Object.freeze(stringArray(raw.categories)),
        description: stringValue(raw.description),
        addedAt: stringValue(raw.addedAt),
        durationSeconds: finiteNonNegative(raw.duration),
        playbackStartSeconds: start,
        playbackEndSeconds: end != null && end > start ? end : null,
        thumbnailURL: stringValue(raw.thumbnailUrl),
        sourceTitle: stringValue(raw.sourceTitle),
        sourceArtist: stringValue(raw.sourceArtist),
        sourceAlbum: stringValue(raw.sourceAlbum),
        sourceGenre: stringValue(raw.sourceGenre),
      });
    }),
  );
}

export async function rescanLibrary({ request = globalThis.fetch } = {}) {
  if (typeof request !== "function") throw new Error("library rescan requires fetch-like request function");
  const response = await request("/api/library/rescan", {
    method: "POST",
    headers: { Accept: "application/json", "Content-Type": "application/json" },
    body: "{}",
  });
  if (!response?.ok) {
    let detail = "";
    try { detail = String((await response.json())?.error ?? "").trim(); } catch {}
    throw new Error(detail || `library rescan request failed: ${response?.status ?? "unknown"}`);
  }
  const payload = await response.json();
  return Object.freeze({ ok: payload?.ok === true, tracks: finiteNonNegative(payload?.tracks) });
}

export async function loadLibraryCatalog({ request = globalThis.fetch } = {}) {
  if (typeof request !== "function") {
    throw new Error("library catalog requires fetch-like request function");
  }
  const response = await request("/api/library", {
    method: "GET",
    headers: { Accept: "application/json" },
  });
  if (!response?.ok) {
    throw new Error(`library catalog request failed: ${response?.status ?? "unknown"}`);
  }
  return projectLibraryCatalog(await response.json());
}

function stringArray(value) {
  return Array.isArray(value)
    ? value
        .map(item => String(item ?? "").trim())
        .filter(Boolean)
    : [];
}

function stringValue(value) {
  return typeof value === "string" ? value : "";
}

function finiteNonNegative(value) {
  const number = Number(value);
  return Number.isFinite(number) && number > 0 ? number : 0;
}

function finitePositive(value) {
  if (value == null || value === "") return null;
  const number = Number(value);
  return Number.isFinite(number) && number > 0 ? number : null;
}
