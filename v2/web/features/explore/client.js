// [VXG RealForever]
// Browser transport for the temporary Explore service.
// It owns request/response shape only; Explore Session owns history and Playback owns local audio truth.

export function candidateExploreSeed(candidate) {
  return Object.freeze({
    kind: "provider",
    providerId: stringValue(candidate?.providerId),
    url: stringValue(candidate?.url),
    title: stringValue(candidate?.title),
    channel: stringValue(candidate?.channel),
    duration: finiteNonNegative(candidate?.duration),
  });
}

export function providerPreviewURL(candidate) {
  const providerId = stringValue(candidate?.providerId).trim();
  if (!providerId) return null;
  return `https://www.youtube.com/embed/${encodeURIComponent(providerId)}?autoplay=1&playsinline=1`;
}

export async function requestExploreFromTrack(
  trackRef,
  { request = globalThis.fetch } = {},
) {
  const ref = stringValue(trackRef).trim();
  if (!ref) throw new Error("Explore requires trackRef");
  return requestExplore({ trackId: ref }, request);
}

export async function requestExploreFromCandidate(
  candidate,
  { request = globalThis.fetch } = {},
) {
  const seed = candidateExploreSeed(candidate);
  if (!seed.title) throw new Error("Explore candidate requires title");
  return requestExplore({ seed }, request);
}

async function requestExplore(body, request) {
  if (typeof request !== "function") {
    throw new Error("Explore requires fetch-like request function");
  }
  const response = await request("/api/discovery/radio", {
    method: "POST",
    headers: {
      Accept: "application/json",
      "Content-Type": "application/json",
    },
    body: JSON.stringify(body),
  });
  if (!response?.ok) {
    throw new Error(`Explore request failed: ${response?.status ?? "unknown"}`);
  }
  return validateFrame(await response.json());
}

function validateFrame(frame) {
  if (!frame || frame.schemaVersion !== "vexstream.discovery-candidates/v2") {
    throw new Error("Explore response schema is unsupported");
  }
  if (!frame.seed || !Array.isArray(frame.candidates)) {
    throw new Error("Explore response is incomplete");
  }
  return frame;
}

function stringValue(value) {
  return typeof value === "string" ? value : "";
}

function finiteNonNegative(value) {
  const number = Number(value);
  return Number.isFinite(number) && number > 0 ? number : 0;
}
