// [VXG RealForever]
// Browser client for the established VexStream <-> VexLife local pairing contract.
//
// This module owns only localhost administration request/response shape. The separate
// desktop/mobile producer owns TLS, mDNS, bearer authorization, catalog, and media bytes.

export async function loadPhonePairing(
  { request = globalThis.fetch } = {},
) {
  const payload = await requestJSON(
    "/api/mobile/pairing",
    {
      method: "GET",
      headers: { Accept: "application/json" },
    },
    request,
    "Phone pairing status",
  );
  return projectPhonePairing(payload);
}

export async function startPhonePairing(
  { request = globalThis.fetch } = {},
) {
  const payload = await requestJSON(
    "/api/mobile/pairing/start",
    {
      method: "POST",
      headers: jsonHeaders(),
      body: JSON.stringify({}),
    },
    request,
    "Start phone pairing",
  );
  const pairingCode = pairingCodeValue(payload?.pairingCode);
  const expiresAt = stringValue(payload?.expiresAt);
  if (payload?.ok !== true || !pairingCode || !expiresAt) {
    throw new Error("Start phone pairing response is incomplete");
  }
  return Object.freeze({ ok: true, pairingCode, expiresAt });
}

export async function decidePhonePairing(
  requestRef,
  decision,
  { request = globalThis.fetch } = {},
) {
  const ref = stringValue(requestRef);
  const resolvedDecision = stringValue(decision).toUpperCase();
  if (!ref) throw new Error("Phone pairing decision requires requestRef");
  if (!["APPROVE", "DENY"].includes(resolvedDecision)) {
    throw new Error("Phone pairing decision must be APPROVE or DENY");
  }

  const payload = await requestJSON(
    "/api/mobile/pairing/decision",
    {
      method: "POST",
      headers: jsonHeaders(),
      body: JSON.stringify({
        requestRef: ref,
        decision: resolvedDecision,
      }),
    },
    request,
    "Phone pairing decision",
  );
  if (payload?.ok !== true || payload?.state !== resolvedDecision + "ED") {
    // APPROVE -> APPROVED; DENY -> DENIED
    const expected = resolvedDecision === "APPROVE" ? "APPROVED" : "DENIED";
    if (payload?.ok !== true || payload?.state !== expected) {
      throw new Error("Phone pairing decision response is incomplete");
    }
  }
  return Object.freeze({
    ok: true,
    state: String(payload.state),
  });
}

export async function forgetPairedPhone(
  deviceRef,
  { request = globalThis.fetch } = {},
) {
  const ref = stringValue(deviceRef);
  if (!ref) throw new Error("Forget phone requires deviceRef");
  const payload = await requestJSON(
    "/api/mobile/pairing/revoke",
    {
      method: "POST",
      headers: jsonHeaders(),
      body: JSON.stringify({ deviceRef: ref }),
    },
    request,
    "Forget phone",
  );
  if (payload?.ok !== true) {
    throw new Error("Forget phone response is incomplete");
  }
  return Object.freeze({ ok: true });
}

export function projectPhonePairing(payload) {
  if (!payload || typeof payload !== "object") {
    throw new Error("Phone pairing status response is incomplete");
  }
  const displayName = stringValue(payload.displayName);
  const pairingCode = pairingCodeValue(payload.pairingCode);
  if (!displayName || !pairingCode) {
    throw new Error("Phone pairing status response is incomplete");
  }

  return Object.freeze({
    running: payload.running === true,
    displayName,
    pairingCode,
    pairingWindowOpen: payload.pairingWindowOpen === true,
    pairingWindowUntil: stringValue(payload.pairingWindowUntil),
    pending: freezePending(payload.pending),
    paired: freezePaired(payload.paired),
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
      detail = stringValue(body?.error);
    } catch {}
    throw new Error(
      detail || `${label} request failed: ${response?.status ?? "unknown"}`,
    );
  }
  return response.json();
}

function freezePending(values) {
  return Object.freeze(
    array(values).map(value => {
      const requestRef = stringValue(value?.requestRef);
      const displayName = stringValue(value?.displayName);
      if (!requestRef || !displayName) {
        throw new Error("Phone pairing pending entry is incomplete");
      }
      return Object.freeze({
        requestRef,
        displayName,
        requestedAt: stringValue(value?.requestedAt),
      });
    }),
  );
}

function freezePaired(values) {
  return Object.freeze(
    array(values).map(value => {
      const deviceRef = stringValue(value?.deviceRef);
      const displayName = stringValue(value?.displayName);
      if (!deviceRef || !displayName) {
        throw new Error("Remembered phone entry is incomplete");
      }
      return Object.freeze({
        deviceRef,
        displayName,
        approvedAt: stringValue(value?.approvedAt),
      });
    }),
  );
}

function pairingCodeValue(value) {
  const code = stringValue(value);
  return /^[0-9]{6}$/.test(code) ? code : "";
}

function jsonHeaders() {
  return {
    Accept: "application/json",
    "Content-Type": "application/json",
  };
}

function array(value) {
  return Array.isArray(value) ? value : [];
}

function stringValue(value) {
  return typeof value === "string" ? value.trim() : "";
}
