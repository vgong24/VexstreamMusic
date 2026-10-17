// [VXG RealForever]
//
// Explore owns a temporary navigation session over provider result frames.
//
// This is deliberately not a Vex Behavior family yet:
// - branch operations carry provider candidate payload;
// - provider search/result grammar is feature-local;
// - preview realization crosses provider + Playback boundaries.
//
// Keep the session question small:
// "What Explore frame/history/preview selection is current right now?"

function freezeFrame(frame) {
  if (!frame || typeof frame !== "object") {
    throw new Error("Explore frame must be an object");
  }
  const candidates = Array.isArray(frame.candidates)
    ? Object.freeze(frame.candidates.map(candidate => Object.freeze({ ...candidate })))
    : Object.freeze([]);
  const basis = frame.basis && typeof frame.basis === "object"
    ? Object.freeze({ ...frame.basis })
    : Object.freeze({});
  const seed = frame.seed && typeof frame.seed === "object"
    ? Object.freeze({ ...frame.seed })
    : Object.freeze({});
  return Object.freeze({
    ...frame,
    seed,
    basis,
    candidates,
  });
}

export function createExploreSession() {
  let frames = Object.freeze([]);
  let frameIndex = -1;
  let previewIndex = -1;
  const listeners = new Set();

  function projection() {
    const frame = frameIndex >= 0 ? frames[frameIndex] ?? null : null;
    return Object.freeze({
      kind: frame ? "active" : "empty",
      frames,
      frameIndex,
      frame,
      canGoBack: frameIndex > 0,
      previewIndex,
      previewCandidate:
        frame && previewIndex >= 0
          ? frame.candidates[previewIndex] ?? null
          : null,
    });
  }

  function publish() {
    const current = projection();
    for (const listener of listeners) listener(current);
    return current;
  }

  function replace(frame) {
    frames = Object.freeze([freezeFrame(frame)]);
    frameIndex = 0;
    previewIndex = -1;
    return publish();
  }

  function branch(frame) {
    const next = [
      ...frames.slice(0, Math.max(0, frameIndex + 1)),
      freezeFrame(frame),
    ];
    frames = Object.freeze(next);
    frameIndex = next.length - 1;
    previewIndex = -1;
    return publish();
  }

  function back() {
    if (frameIndex <= 0) return projection();
    frameIndex -= 1;
    previewIndex = -1;
    return publish();
  }

  function selectPreview(index) {
    const frame = projection().frame;
    if (!frame) return projection();
    const normalized = Number(index);
    if (
      !Number.isInteger(normalized) ||
      normalized < 0 ||
      normalized >= frame.candidates.length
    ) {
      throw new Error("Explore preview index must address current candidates");
    }
    previewIndex = normalized;
    return publish();
  }

  function stopPreview() {
    if (previewIndex < 0) return projection();
    previewIndex = -1;
    return publish();
  }

  function clear() {
    frames = Object.freeze([]);
    frameIndex = -1;
    previewIndex = -1;
    return publish();
  }

  return Object.freeze({
    get currentProjection() {
      return projection();
    },
    replace,
    branch,
    back,
    selectPreview,
    stopPreview,
    clear,
    subscribe(listener) {
      if (typeof listener !== "function") {
        throw new Error("Explore subscriber must be a function");
      }
      listeners.add(listener);
      listener(projection());
      return () => listeners.delete(listener);
    },
  });
}
