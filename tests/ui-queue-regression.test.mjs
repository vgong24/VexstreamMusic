import assert from 'node:assert/strict';
import fs from 'node:fs';
import test from 'node:test';
import vm from 'node:vm';

const html = fs.readFileSync(new URL('../src/ui/index.html', import.meta.url), 'utf8');

function extractFunction(name) {
  const start = html.indexOf(`function ${name}(`);
  assert.notEqual(start, -1, `${name} should exist in recovered UI`);
  const brace = html.indexOf('{', start);
  let depth = 0;
  let quote = null;
  let escaped = false;
  for (let i = brace; i < html.length; i += 1) {
    const ch = html[i];
    if (quote) {
      if (escaped) escaped = false;
      else if (ch === '\\') escaped = true;
      else if (ch === quote) quote = null;
      continue;
    }
    if (ch === '"' || ch === "'" || ch === '`') { quote = ch; continue; }
    if (ch === '{') depth += 1;
    else if (ch === '}') {
      depth -= 1;
      if (depth === 0) return html.slice(start, i + 1);
    }
  }
  throw new Error(`unterminated ${name}`);
}

const setQueueSource = extractFunction('setQueue');
const playLibraryTrackSource = extractFunction('playLibraryTrack');
const queueTrackFromButtonSource = extractFunction('queueTrackFromButton');
const togglePlaybackSource = extractFunction('togglePlayback');

function makePlaybackHarness({ shuffle = false, view = 'songs' } = {}) {
  const tracks = ['a', 'b', 'c', 'd'].map(id => ({ id }));
  const calls = [];
  const context = {
    state: { tracks, queue: [], queueIndex: -1, shuffle },
    currentView: view,
    recentTracks: () => [tracks[2], tracks[0]],
    // Deliberately represents an active Songs filter. The fixed Songs playback
    // path must not use this as its playback universe.
    sortedSongTracks: () => [tracks[1], tracks[3]],
    sortTrackList: rows => [...rows],
    shuf: rows => [...rows].reverse(),
    saveQueue: () => calls.push(['saveQueue']),
    playIndex: i => calls.push(['playIndex', i]),
    renderQueue: () => calls.push(['renderQueue']),
    renderPlayer: () => calls.push(['renderPlayer']),
    playTrack: id => calls.push(['playTrack', id]),
  };
  vm.createContext(context);
  vm.runInContext(`${setQueueSource}\n${playLibraryTrackSource}`, context);
  return { context, tracks, calls };
}

test('Songs filtering remains a display lens: non-shuffle playback receives the full library', () => {
  const { context, calls } = makePlaybackHarness({ shuffle: false });
  context.playLibraryTrack('b');
  assert.deepEqual(Array.from(context.state.queue), ['a', 'b', 'c', 'd']);
  assert.equal(context.state.queueIndex, 1);
  assert.deepEqual(calls.at(-1), ['playIndex', 1]);
});

test('Songs shuffle starts on the chosen result and includes every library song exactly once', () => {
  const { context, calls } = makePlaybackHarness({ shuffle: true });
  context.playLibraryTrack('b');
  assert.equal(context.state.queueIndex, 0);
  assert.equal(context.state.queue[0], 'b');
  assert.deepEqual(new Set(context.state.queue), new Set(['a', 'b', 'c', 'd']));
  assert.equal(context.state.queue.length, 4);
  // Deterministic reverse shuffle proves that only the future portion is shuffled.
  assert.deepEqual(Array.from(context.state.queue), ['b', 'd', 'c', 'a']);
  assert.deepEqual(calls.at(-1), ['playIndex', 0]);
});

test('Home recent-track playback context remains unchanged', () => {
  const { context } = makePlaybackHarness({ shuffle: false, view: 'home' });
  context.playLibraryTrack('c');
  assert.deepEqual(Array.from(context.state.queue), ['c', 'a']);
  assert.equal(context.state.queueIndex, 0);
});



test('idle player Play in Songs also starts from the full library, not the filtered projection', () => {
  const tracks = ['a', 'b', 'c', 'd'].map(id => ({ id }));
  const calls = [];
  const context = {
    audio: { paused: true, pause() {}, dataset: {}, currentSrc: '', play: () => Promise.resolve() },
    state: { tracks, queue: [], queueIndex: -1, shuffle: false },
    currentView: 'songs',
    currentTrack: () => null,
    sortedSongTracks: () => [tracks[1], tracks[3]],
    sortTrackList: rows => [...rows],
    setQueue: (rows, start, autoplay) => calls.push([rows.map(t => t.id), start, autoplay]),
    playIndex: () => { throw new Error('playIndex should not be used without a current track'); },
  };
  vm.createContext(context);
  vm.runInContext(togglePlaybackSource, context);
  context.togglePlayback();
  assert.deepEqual(calls, [[['a', 'b', 'c', 'd'], 0, true]]);
});

test('Play all and Shuffle all are explicitly full-library actions', () => {
  assert.match(html, /bindClick\('playAllBtn',\(\)=>setQueue\(sortTrackList\(state\.tracks\),0,true\)\);/);
  assert.match(html, /bindClick\('shuffleAllBtn',\(\)=>setQueue\(shuf\(sortTrackList\(state\.tracks\)\),0,true\)\);/);
  assert.doesNotMatch(html, /bindClick\('playAllBtn',\(\)=>setQueue\(sortedSongTracks\(\),0,true\)\);/);
  assert.doesNotMatch(html, /bindClick\('shuffleAllBtn',\(\)=>setQueue\(shuf\(sortedSongTracks\(\)\),0,true\)\);/);
});

test('row Add to queue remains a one-track action even when the Songs view is filtered', () => {
  const calls = [];
  const context = {
    trackById: id => ({ id }),
    appendQueue: track => calls.push(['appendQueue', track.id]),
    flashAction: button => calls.push(['flashAction', button]),
  };
  vm.createContext(context);
  vm.runInContext(queueTrackFromButtonSource, context);
  context.queueTrackFromButton('d', 'button-ref');
  assert.deepEqual(calls, [['appendQueue', 'd'], ['flashAction', 'button-ref']]);
});

test('1.1.7 UI keeps filtering code but no longer reports 1.1.5 or 1.1.6', () => {
  assert.match(html, /function baseFilteredTracks\(\)/);
  assert.match(html, /function sortedSongTracks\(\)\{return sortTrackList\(baseFilteredTracks\(\)\)\}/);
  assert.match(html, /<span class="version">1\.1\.7<\/span>/);
  assert.doesNotMatch(html, /<span class="version">1\.1\.5<\/span>/);
  assert.doesNotMatch(html, /<span class="version">1\.1\.6<\/span>/);
});