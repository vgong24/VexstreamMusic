import assert from "node:assert/strict";
import test from "node:test";

import {
  candidateExploreSeed,
  providerPreviewURL,
  requestExploreFromCandidate,
  requestExploreFromTrack,
} from "../web/features/explore/client.js";

const frame = Object.freeze({
  schemaVersion: "vexstream.discovery-candidates/v2",
  seed: Object.freeze({ title: "Seed" }),
  candidates: Object.freeze([]),
});

test("track request preserves opaque trackRef and read-only Explore route", async () => {
  let seen;
  const result = await requestExploreFromTrack(" track-a ", {
    request: async (url, options) => {
      seen = { url, options };
      return {
        ok: true,
        status: 200,
        json: async () => frame,
      };
    },
  });

  assert.equal(seen.url, "/api/discovery/radio");
  assert.equal(seen.options.method, "POST");
  assert.deepEqual(JSON.parse(seen.options.body), { trackId: "track-a" });
  assert.equal(result, frame);
});

test("candidate request keeps provider payload feature-local", async () => {
  let body;
  await requestExploreFromCandidate(
    {
      providerId: "abc",
      url: "https://example/video",
      title: "Candidate",
      channel: "Creator",
      duration: 123,
    },
    {
      request: async (_url, options) => {
        body = JSON.parse(options.body);
        return {
          ok: true,
          status: 200,
          json: async () => frame,
        };
      },
    },
  );

  assert.deepEqual(body, {
    seed: {
      kind: "provider",
      providerId: "abc",
      url: "https://example/video",
      title: "Candidate",
      channel: "Creator",
      duration: 123,
    },
  });
});

test("candidate seed normalization does not manufacture invalid duration", () => {
  assert.deepEqual(
    candidateExploreSeed({
      providerId: "abc",
      title: "Candidate",
      duration: -5,
    }),
    {
      kind: "provider",
      providerId: "abc",
      url: "",
      title: "Candidate",
      channel: "",
      duration: 0,
    },
  );
});

test("provider preview URL requires provider identity and escapes it", () => {
  assert.equal(providerPreviewURL({}), null);
  assert.equal(
    providerPreviewURL({ providerId: "a/b" }),
    "https://www.youtube.com/embed/a%2Fb?autoplay=1&playsinline=1",
  );
});

test("unsupported or incomplete Explore response fails closed", async () => {
  await assert.rejects(
    () =>
      requestExploreFromTrack("track-a", {
        request: async () => ({
          ok: true,
          status: 200,
          json: async () => ({
            schemaVersion: "other/v1",
            seed: {},
            candidates: [],
          }),
        }),
      }),
    /schema is unsupported/,
  );

  await assert.rejects(
    () =>
      requestExploreFromTrack("track-a", {
        request: async () => ({ ok: false, status: 503 }),
      }),
    /503/,
  );
});
