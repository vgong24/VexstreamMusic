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


test("shell presentation state is not registered as a shell navigation control",()=>{
  const here=path.dirname(fileURLToPath(import.meta.url));
  const html=fs.readFileSync(path.join(here,"../web/index.html"),"utf8");
  const app=fs.readFileSync(path.join(here,"../web/app.js"),"utf8");

  assert.equal(/<body[^>]*data-shell-view=/.test(html),false);
  assert.equal(/<body[^>]*data-shell-current-view="home"/.test(html),true);
  assert.equal(app.includes('document.querySelectorAll("button[data-shell-view]")'),true);
  assert.equal(app.includes("document.body.dataset.shellCurrentView=projection.view"),true);
});

test("desktop HTML preserves recognizable V3 shell landmarks around V4 owners",()=>{
  const here=path.dirname(fileURLToPath(import.meta.url));
  const html=fs.readFileSync(path.join(here,"../web/index.html"),"utf8");

  for(const landmark of [
    'id="brandHome"',
    'id="systemStatusButton"',
    'id="systemStatusBadge"',
    'id="systemDrawer"',
    'id="systemSummary"',
    'id="exitVexStream"',
    '>Library</button>',
    '>Discover</button>',
    'data-shell-view="home"',
    'data-shell-view="songs"',
    'data-shell-view="albums"',
    'data-shell-view="artists"',
    'data-shell-view="playlists"',
    'data-shell-view="deleted"',
    'id="playlistNavCount"',
    'id="trashNavCount"',
    'id="rescanLibraryBtn"',
    'id="shuffleAllBtn"',
    'id="libraryUtilityBar"',
    'id="librarySearch"',
    'id="librarySearchExecute"',
    'id="librarySearchClear"',
    'id="libraryGenre"',
    'id="goDiscoverAdd"',
    'id="recentSongs"',
    'id="songSortSummary"',
    'id="songLayoutButton"',
    'id="songLayoutPanel"',
    'id="songTitleWidth"',
    'id="selectSongs"',
    'id="librarySelectionBar"',
    'id="selectVisibleSongs"',
    'id="bulkPlaySelected"',
    'id="bulkPlayNextSelected"',
    'id="bulkQueueSelected"',
    'id="bulkEditSelected"',
    'id="bulkDisplayEditor"',
    'id="applyBulkDisplay"',
    'id="bulkPlaylistSelected"',
    'id="bulkTrashSelected"',
    'id="playlistOverview"',
    'id="playlistDetail"',
    'id="playlistDetailBack"',
    'id="playlistDetailAddSongs"',
    'id="playlistDetailEntries"',
    'id="trashTitle"',
    'id="trashSelectionBar"',
    'id="trashSelectionCount"',
    'id="restoreSelectedTrash"',
    'id="purgeSelectedTrash"',
    'id="trashCount"',
    'id="deletedHistoryCount"',
    'id="trashPurgeConfirm"',
    'id="preserveDeletedMetadata"',
    'id="trashPurgeCommit"',
    'id="discoverTitle"',
    'data-discover-mode="pc"',
    'data-discover-mode="youtube"',
    'data-discover-mode="explore"',
    'id="youtubeImportURL"',
    'id="libraryGroupDetail"',
    'id="libraryGroupArtwork"',
    'id="libraryGroupBack"',
    'id="libraryGroupTracks"',
    'id="queueDrawer"',
    'id="queueDrawerHandle"',
    'id="queueDrawerCount"',
    'id="queueDrawerNextCount"',
    'class="queue queue-drawer-body"',
    'id="trackTitle"',
    'id="curTime"',
    'id="seek"',
    'id="durTime"',
    'id="volume"',
    'id="queueMiniBtn"',
    'id="queueMiniCount"',
    'id="playPause"',
    'id="audio"'
  ]){
    assert.equal(html.includes(landmark),true,landmark);
  }

  assert.equal(html.includes('class="queue queue-panel"'),false);
  assert.equal(html.includes("Desktop V4 player practicum"),false);
  assert.equal(html.includes('id="phoneConnectBtn"'),false);
  assert.equal(html.includes('id="runDiagnosticsBtn"'),false);
  assert.equal(html.includes('id="loadRepairPlanBtn"'),false);
});
