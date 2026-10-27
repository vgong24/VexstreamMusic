// [VXG RealForever]
// Human Phone Connection walk for Desktop V4.
//
// The walk intentionally mirrors the stable VexLife consumer:
// open -> find/advertise -> compare code -> approve/deny -> remembered pairing -> forget.
//
// Network contract remains in client.js. Modal HTML remains in presentation.js.

export const PhoneConnectionAction = Object.freeze({
  StartPairing: "start",
  Approve: "approve",
  Deny: "deny",
  Forget: "forget",
});

const ACTIONS = new Set(Object.values(PhoneConnectionAction));

export function createPhoneConnection({
  loadPairing,
  startPairing,
  decidePairing,
  forgetPairedPhone,
  confirmForget = () => true,
  schedule = globalThis.setTimeout,
  cancel = globalThis.clearTimeout,
  pollMs = 1000,
} = {}) {
  for (const [name, value] of Object.entries({
    loadPairing,
    startPairing,
    decidePairing,
    forgetPairedPhone,
    confirmForget,
    schedule,
    cancel,
  })) {
    if (typeof value !== "function") {
      throw new Error(`Phone Connection requires ${name}()`);
    }
  }
  if (!Number.isInteger(pollMs) || pollMs < 100) {
    throw new Error("Phone Connection pollMs must be at least 100");
  }

  let projection = freezeProjection({
    open: false,
    phase: "closed",
    pairing: null,
    error: "",
  });
  let pollTimer = null;
  let revision = 0;
  const listeners = new Set();

  function publish(next = projection) {
    projection = freezeProjection(next);
    for (const listener of listeners) listener(projection);
    return projection;
  }

  function stopPolling() {
    if (pollTimer !== null) cancel(pollTimer);
    pollTimer = null;
  }

  function scheduleRefresh() {
    stopPolling();
    if (!projection.open || projection.phase !== "ready") return;
    pollTimer = schedule(() => {
      pollTimer = null;
      void refresh();
    }, pollMs);
  }

  async function observe({ autoStart = false, showLoading = false } = {}) {
    if (!projection.open) return projection;
    const observedRevision = ++revision;
    stopPolling();
    if (showLoading) {
      publish({
        ...projection,
        phase: "loading",
        error: "",
      });
    }

    try {
      let pairing = await loadPairing();
      if (!projection.open || observedRevision !== revision) return projection;

      if (
        autoStart &&
        !pairing.pairingWindowOpen &&
        pairing.paired.length === 0
      ) {
        await startPairing();
        if (!projection.open || observedRevision !== revision) return projection;
        pairing = await loadPairing();
        if (!projection.open || observedRevision !== revision) return projection;
      }

      publish({
        open: true,
        phase: "ready",
        pairing,
        error: "",
      });
      scheduleRefresh();
      return projection;
    } catch (failure) {
      if (!projection.open || observedRevision !== revision) return projection;
      publish({
        ...projection,
        phase: "failed",
        error: failure?.message ?? String(failure),
      });
      return projection;
    }
  }

  async function open() {
    if (projection.open) return refresh();
    publish({
      open: true,
      phase: "loading",
      pairing: null,
      error: "",
    });
    return observe({ autoStart: true });
  }

  async function refresh() {
    return observe({ autoStart: false });
  }

  async function activate(action) {
    if (!projection.open) {
      throw new Error("Phone Connection action requires an open presentation");
    }
    const kind = String(action?.kind ?? "").trim();
    if (!ACTIONS.has(kind)) {
      throw new Error("Phone Connection action is unsupported");
    }
    stopPolling();

    try {
      if (kind === PhoneConnectionAction.StartPairing) {
        await startPairing();
      } else if (
        kind === PhoneConnectionAction.Approve ||
        kind === PhoneConnectionAction.Deny
      ) {
        const requestRef = String(action?.requestRef ?? "").trim();
        if (!requestRef) {
          throw new Error("Phone pairing decision requires requestRef");
        }
        await decidePairing(
          requestRef,
          kind === PhoneConnectionAction.Approve ? "APPROVE" : "DENY",
        );
      } else if (kind === PhoneConnectionAction.Forget) {
        const deviceRef = String(action?.deviceRef ?? "").trim();
        if (!deviceRef) throw new Error("Forget phone requires deviceRef");
        const confirmed = await confirmForget(deviceRef);
        if (!confirmed) {
          scheduleRefresh();
          return projection;
        }
        await forgetPairedPhone(deviceRef);
      }
      return observe({ autoStart: false });
    } catch (failure) {
      publish({
        ...projection,
        phase: "failed",
        error: failure?.message ?? String(failure),
      });
      return projection;
    }
  }

  function close() {
    revision += 1;
    stopPolling();
    return publish({
      open: false,
      phase: "closed",
      pairing: null,
      error: "",
    });
  }

  return Object.freeze({
    get currentProjection() {
      return projection;
    },
    open,
    refresh,
    activate,
    close,
    subscribe(listener) {
      if (typeof listener !== "function") {
        throw new Error("Phone Connection subscriber must be a function");
      }
      listeners.add(listener);
      listener(projection);
      return () => listeners.delete(listener);
    },
  });
}

function freezeProjection(value) {
  return Object.freeze({
    open: value.open === true,
    phase: String(value.phase ?? ""),
    pairing: value.pairing ?? null,
    error: String(value.error ?? ""),
  });
}
