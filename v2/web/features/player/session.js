// [VXG RealForever]
// Durable Desktop V4 listening-session browser projection.
//
// Persistence remembers Queue composition truth. It never auto-starts Playback.

import { currentQueueRef } from "../queue/behavior.js";

export const PlayerSessionSchema = "vexstream.desktop-player-session/v1";

export async function loadPlayerSession(
  { request = globalThis.fetch } = {},
) {
  const payload = await requestJSON(
    "/api/player/session",
    {
      method: "GET",
      headers: { Accept: "application/json" },
    },
    request,
    "Player session",
  );

  const found = payload?.found === true;
  const session = normalizeSession(payload?.session);
  return Object.freeze({ found, session });
}

export async function savePlayerSession(
  queueState,
  { request = globalThis.fetch } = {},
) {
  const session = sessionFromQueueState(queueState);
  const payload = await requestJSON(
    "/api/player/session",
    {
      method: "PUT",
      headers: {
        Accept: "application/json",
        "Content-Type": "application/json",
      },
      body: JSON.stringify({
        items: session.items,
        currentIndex: session.currentIndex,
        currentRef: session.currentRef,
        shuffle: session.shuffle,
        repeat: session.repeat,
      }),
    },
    request,
    "Player session save",
  );
  return normalizeSession(payload?.session);
}

export function sessionFromQueueState(state) {
  if (!state || !["empty", "positioned"].includes(state.kind)) {
    throw new Error("Player session requires Queue State");
  }

  if (state.kind === "empty") {
    return Object.freeze({
      schemaVersion: PlayerSessionSchema,
      items: Object.freeze([]),
      currentIndex: -1,
      currentRef: "",
      shuffle: Boolean(state.shuffle),
      repeat: Boolean(state.repeat),
    });
  }

  const items = Object.freeze([...state.items]);
  const currentIndex = Number(state.currentIndex);
  const currentRef = String(currentQueueRef(state) ?? "").trim();
  if (
    !Number.isInteger(currentIndex) ||
    currentIndex < 0 ||
    currentIndex >= items.length ||
    !currentRef ||
    items[currentIndex] !== currentRef
  ) {
    throw new Error("Queue current witness is invalid for persistence");
  }

  return Object.freeze({
    schemaVersion: PlayerSessionSchema,
    items,
    currentIndex,
    currentRef,
    shuffle: Boolean(state.shuffle),
    repeat: Boolean(state.repeat),
  });
}

export function reconcilePlayerSession(
  loaded,
  availableRefs,
) {
  if (!loaded?.found) {
    return freezeRestoration({
      kind: "fresh",
      queue: null,
      droppedRefs: [],
    });
  }

  const session = normalizeSession(loaded.session);
  if (!session.items.length) {
    return freezeRestoration({
      kind: "empty",
      queue: {
        items: [],
        currentIndex: -1,
        shuffle: session.shuffle,
        repeat: session.repeat,
      },
      droppedRefs: [],
    });
  }

  const available = new Set(
    [...availableRefs].map(value => String(value ?? "").trim()),
  );
  const kept = [];
  const droppedRefs = [];
  let restoredCurrentIndex = -1;

  session.items.forEach((ref, originalIndex) => {
    if (!available.has(ref)) {
      droppedRefs.push(ref);
      return;
    }
    if (originalIndex === session.currentIndex) {
      restoredCurrentIndex = kept.length;
    }
    kept.push(ref);
  });

  if (restoredCurrentIndex < 0) {
    return freezeRestoration({
      kind: "held-current-unavailable",
      queue: null,
      droppedRefs,
      currentRef: session.currentRef,
    });
  }

  return freezeRestoration({
    kind: "restored",
    queue: {
      items: kept,
      currentIndex: restoredCurrentIndex,
      shuffle: session.shuffle,
      repeat: session.repeat,
    },
    droppedRefs,
    currentRef: session.currentRef,
  });
}

function normalizeSession(value) {
  if (!value || value.schemaVersion !== PlayerSessionSchema) {
    throw new Error("Player session schema is unsupported");
  }

  const items = Object.freeze(
    Array.isArray(value.items)
      ? value.items.map(ref => String(ref ?? "").trim())
      : [],
  );
  if (items.some(ref => !ref)) {
    throw new Error("Player session contains blank queue ref");
  }

  const currentIndex = Number(value.currentIndex);
  const currentRef = String(value.currentRef ?? "").trim();

  if (!items.length) {
    return Object.freeze({
      schemaVersion: PlayerSessionSchema,
      items,
      currentIndex: -1,
      currentRef: "",
      shuffle: Boolean(value.shuffle),
      repeat: Boolean(value.repeat),
      updatedAt: String(value.updatedAt ?? ""),
    });
  }

  if (
    !Number.isInteger(currentIndex) ||
    currentIndex < 0 ||
    currentIndex >= items.length ||
    !currentRef ||
    items[currentIndex] !== currentRef
  ) {
    throw new Error("Player session current witness is invalid");
  }

  return Object.freeze({
    schemaVersion: PlayerSessionSchema,
    items,
    currentIndex,
    currentRef,
    shuffle: Boolean(value.shuffle),
    repeat: Boolean(value.repeat),
    updatedAt: String(value.updatedAt ?? ""),
  });
}

function freezeRestoration({
  kind,
  queue,
  droppedRefs,
  currentRef = "",
}) {
  return Object.freeze({
    kind,
    queue: queue
      ? Object.freeze({
          ...queue,
          items: Object.freeze([...queue.items]),
        })
      : null,
    droppedRefs: Object.freeze([...droppedRefs]),
    currentRef,
  });
}

async function requestJSON(path, options, request, label) {
  if (typeof request !== "function") {
    throw new Error(label + " requires fetch-like request function");
  }
  const response = await request(path, options);
  if (!response?.ok) {
    let detail = "";
    try {
      const body = await response.json();
      detail = String(body?.error ?? "").trim();
    } catch {
      // Status remains useful when no JSON error is available.
    }
    throw new Error(
      detail || `${label} request failed: ${response?.status ?? "unknown"}`,
    );
  }
  return response.json();
}
