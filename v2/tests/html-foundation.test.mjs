import assert from "node:assert/strict";
import fs from "node:fs";
import path from "node:path";
import test from "node:test";
import { fileURLToPath } from "node:url";

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


test("desktop HTML preserves recognizable V3 shell landmarks around V4 owners",()=>{
  const here=path.dirname(fileURLToPath(import.meta.url));
  const html=fs.readFileSync(path.join(here,"../web/index.html"),"utf8");

  for(const landmark of [
    'id="brandHome"',
    '>Library</button>',
    '>Discover</button>',
    'data-shell-view="home"',
    'data-shell-view="songs"',
    'data-shell-view="albums"',
    'data-shell-view="artists"',
    'data-shell-view="playlists"',
    'data-shell-view="deleted"',
    'id="recentSongs"',
    'id="discoverTitle"',
    'data-discover-mode="pc"',
    'data-discover-mode="youtube"',
    'data-discover-mode="explore"',
    'id="youtubeImportURL"',
    'id="libraryGroupDetail"',
    'id="libraryGroupBack"',
    'id="libraryGroupTracks"',
    'class="queue queue-panel"',
    'id="trackTitle"',
    'id="playPause"',
    'id="audio"'
  ]){
    assert.equal(html.includes(landmark),true,landmark);
  }

  assert.equal(html.includes("Desktop V4 player practicum"),false);
});
