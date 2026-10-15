// [VXG RealForever]
// Browser consumer for V4 Import inspection and job lifecycle.
//
// This module owns request/response shape only. It does not decide duplicate blocking, provider
// execution, staging custody, or Library commit/recovery semantics.

export async function inspectImport(
  url,
  { request = globalThis.fetch } = {},
) {
  const value = String(url ?? "").trim();
  if (!value) throw new Error("Import inspection requires url");
  return requestJSON(
    "/api/import/inspect",
    {
      method: "POST",
      headers: jsonHeaders(),
      body: JSON.stringify({ url: value }),
    },
    request,
    "Import inspection",
  );
}

export async function loadImportJobs(
  { request = globalThis.fetch } = {},
) {
  const payload = await requestJSON(
    "/api/import/jobs",
    {
      method: "GET",
      headers: { Accept: "application/json" },
    },
    request,
    "Import jobs",
  );
  return projectJobSnapshot(payload);
}

export async function startImport(
  plan,
  { request = globalThis.fetch } = {},
) {
  if (!plan || typeof plan !== "object") {
    throw new Error("Import start requires plan");
  }
  const url = String(plan.url ?? "").trim();
  const destination = String(plan.destination ?? "").trim();
  if (!url || !destination) {
    throw new Error("Import start requires url and destination");
  }

  const payload = {
    sourceTitle: stringValue(plan.sourceTitle),
    destination,
    url,
    mode: stringValue(plan.mode),
    wholeTitle: stringValue(plan.wholeTitle),
    artistOverride: stringValue(plan.artistOverride),
    collection: stringValue(plan.collection),
    genre: stringValue(plan.genre),
    sections: Array.isArray(plan.sections)
      ? plan.sections.map(projectSection)
      : [],
  };

  return requestJSON(
    "/api/import/start",
    {
      method: "POST",
      headers: jsonHeaders(),
      body: JSON.stringify(payload),
    },
    request,
    "Import start",
  );
}

export function cancelImport(id, options) {
  return jobAction("/api/import/cancel", id, options, "Import cancel");
}

export function retryImportCommit(id, options) {
  return jobAction("/api/import/commit", id, options, "Import commit retry");
}

export function discardImport(id, options) {
  return jobAction("/api/import/discard", id, options, "Import discard");
}

export function recoverImportCommit(id, options) {
  return jobAction(
    "/api/import/recover-commit",
    id,
    options,
    "Import commit recovery",
  );
}

export function projectJobSnapshot(payload) {
  const groups = payload?.groups;
  if (!groups || typeof groups !== "object") {
    throw new Error("Import job response requires groups");
  }
  return Object.freeze({
    active: freezeJobs(groups.active),
    ready: freezeJobs(groups.ready),
    completed: freezeJobs(groups.completed),
    recovery: Object.freeze({ ...(payload.recovery ?? {}) }),
    issues: Object.freeze(
      Array.isArray(payload.issues)
        ? payload.issues.map(issue => Object.freeze({ ...issue }))
        : [],
    ),
  });
}

async function jobAction(path, id, options = {}, label) {
  const ref = String(id ?? "").trim();
  if (!ref) throw new Error(label + " requires job id");
  return requestJSON(
    path,
    {
      method: "POST",
      headers: jsonHeaders(),
      body: JSON.stringify({ id: ref }),
    },
    options.request ?? globalThis.fetch,
    label,
  );
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
      // HTTP status remains sufficient when no structured body is available.
    }
    throw new Error(
      detail || `${label} request failed: ${response?.status ?? "unknown"}`,
    );
  }
  return response.json();
}

function jsonHeaders() {
  return {
    Accept: "application/json",
    "Content-Type": "application/json",
  };
}

function freezeJobs(input) {
  return Object.freeze(
    Array.isArray(input)
      ? input.map(job => Object.freeze({
          ...job,
          trackIds: Object.freeze([...(job.trackIds ?? [])]),
          problem:
            job.problem && typeof job.problem === "object"
              ? Object.freeze({ ...job.problem })
              : null,
        }))
      : [],
  );
}

function projectSection(section) {
  return {
    index: Number(section?.index ?? 0),
    sourceTitle: stringValue(section?.sourceTitle),
    title: stringValue(section?.title),
    artist: stringValue(section?.artist),
    start: finiteNonNegative(section?.start),
    end: finiteNonNegative(section?.end),
    selected: section?.selected !== false,
  };
}

function stringValue(value) {
  return typeof value === "string" ? value.trim() : "";
}

function finiteNonNegative(value) {
  const number = Number(value);
  return Number.isFinite(number) && number > 0 ? number : 0;
}
