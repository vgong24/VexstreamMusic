import assert from "node:assert/strict";
import test from "node:test";

import {
  cancelImport,
  discardImport,
  inspectImport,
  loadImportJobs,
  loadImportRuntimeStatus,
  projectImportRuntimeStatus,
  projectJobSnapshot,
  recoverImportCommit,
  retryImportCommit,
  setupImportRuntime,
  startImport,
} from "../web/features/imports/client.js";

function okJSON(payload) {
  return {
    ok: true,
    status: 200,
    json: async () => payload,
  };
}

test("inspection posts one provider URL without starting acquisition", async () => {
  let seen;
  const result = await inspectImport(" https://example/song ", {
    request: async (url, options) => {
      seen = { url, options };
      return okJSON({
        inspection: { id: "abc", title: "Song" },
        items: [],
        matchCount: 0,
        advisory: true,
      });
    },
  });

  assert.equal(seen.url, "/api/import/inspect");
  assert.equal(seen.options.method, "POST");
  assert.deepEqual(JSON.parse(seen.options.body), {
    url: "https://example/song",
  });
  assert.equal(result.advisory, true);
});

test("runtime status and one-time setup preserve V3 import readiness semantics", async () => {
  const seen = [];
  const request = async (url, options) => {
    seen.push([url, options.method]);
    if (url === "/api/import/status") {
      return okJSON({
        searchReady: true,
        ffmpegReady: false,
        importReady: false,
        detail: "FFmpeg is still required.",
      });
    }
    return okJSON({
      message: "ready",
      status: {
        searchReady: true,
        ffmpegReady: true,
        importReady: true,
        detail: "YouTube import is ready.",
      },
    });
  };

  const before = await loadImportRuntimeStatus({ request });
  assert.equal(before.searchReady, true);
  assert.equal(before.importReady, false);
  assert.equal(before.detail, "FFmpeg is still required.");

  const after = await setupImportRuntime({ request });
  assert.equal(after.importReady, true);
  assert.equal(after.ffmpegReady, true);
  assert.deepEqual(seen, [
    ["/api/import/status", "GET"],
    ["/api/import/setup", "POST"],
  ]);

  const legacy = projectImportRuntimeStatus({
    ready: true,
    ffmpegReady: true,
  });
  assert.equal(legacy.importReady, true);
});

test("start maps the reviewed plan without inventing selected=false defaults", async () => {
  let body;
  await startImport(
    {
      sourceTitle: " Song ",
      sourceCreator: " Uploader ",
      thumbnail: " https://example/song.jpg ",
      sourceDuration: 123.5,
      destination: " /music ",
      url: " https://example/song ",
      mode: "both",
      sections: [
        { index: 1, title: "One", start: 0, end: 30 },
        { index: 2, title: "Two", start: 30, end: 60, selected: false },
      ],
    },
    {
      request: async (_url, options) => {
        body = JSON.parse(options.body);
        return okJSON({ jobId: "job-a", status: "queued" });
      },
    },
  );

  assert.equal(body.sourceTitle, "Song");
  assert.equal(body.sourceCreator, "Uploader");
  assert.equal(body.thumbnail, "https://example/song.jpg");
  assert.equal(body.sourceDuration, 123.5);
  assert.equal(body.destination, "/music");
  assert.equal(body.url, "https://example/song");
  assert.equal(body.sections[0].selected, true);
  assert.equal(body.sections[1].selected, false);
});

test("job snapshot keeps recovery separate from normal status groups", async () => {
  const projected = projectJobSnapshot({
    groups: {
      active: [{ id: "a", status: "running" }],
      ready: [{ id: "r", status: "ready" }],
      completed: [{ id: "d", status: "done", trackIds: ["track-a"] }],
    },
    recovery: { uncertain: "COMMIT_UNCERTAIN" },
    issues: [{ code: "COMMIT_RECONCILIATION_REQUIRED" }],
  });

  assert.equal(projected.active[0].id, "a");
  assert.equal(projected.ready[0].id, "r");
  assert.deepEqual(projected.completed[0].trackIds, ["track-a"]);
  assert.equal(projected.recovery.uncertain, "COMMIT_UNCERTAIN");
  assert.equal(
    projected.issues[0].code,
    "COMMIT_RECONCILIATION_REQUIRED",
  );

  const loaded = await loadImportJobs({
    request: async () =>
      okJSON({
        groups: { active: [], ready: [], completed: [] },
      }),
  });
  assert.deepEqual(loaded.active, []);
});

test("job actions remain distinct endpoints", async () => {
  const seen = [];
  const request = async (url, options) => {
    seen.push([url, JSON.parse(options.body).id]);
    return okJSON({ ok: true });
  };

  await cancelImport("a", { request });
  await retryImportCommit("b", { request });
  await discardImport("c", { request });
  await recoverImportCommit("d", { request });

  assert.deepEqual(seen, [
    ["/api/import/cancel", "a"],
    ["/api/import/commit", "b"],
    ["/api/import/discard", "c"],
    ["/api/import/recover-commit", "d"],
  ]);
});

test("failed HTTP response preserves server error when available", async () => {
  await assert.rejects(
    () =>
      discardImport("job", {
        request: async () => ({
          ok: false,
          status: 409,
          json: async () => ({ error: "Import job is not discardable" }),
        }),
      }),
    /not discardable/,
  );
});

test("invalid browser inputs fail before request", async () => {
  let calls = 0;
  const request = async () => {
    calls++;
    return okJSON({});
  };

  await assert.rejects(
    () => inspectImport(" ", { request }),
    /requires url/,
  );
  await assert.rejects(
    () => startImport({ url: "x" }, { request }),
    /url and destination/,
  );
  await assert.rejects(
    () => cancelImport(" ", { request }),
    /job id/,
  );
  assert.equal(calls, 0);
});
