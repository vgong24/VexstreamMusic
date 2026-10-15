import assert from "node:assert/strict";
import test from "node:test";

import { escapeHTML } from "../web/foundation/html.js";

test("dynamic product data is escaped for HTML text and quoted attributes", () => {
  assert.equal(
    escapeHTML(`A&B <tag data-x="1">'quoted'</tag>`),
    "A&amp;B &lt;tag data-x=&quot;1&quot;&gt;&#39;quoted&#39;&lt;/tag&gt;",
  );
});

test("nullish values become empty text", () => {
  assert.equal(escapeHTML(null), "");
  assert.equal(escapeHTML(undefined), "");
});
