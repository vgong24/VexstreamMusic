import assert from "node:assert/strict";
import test from "node:test";
import { createHTMLMediaPlaybackConnection } from "../web/features/playback/connection.js";
import { QueueState, currentQueueRef } from "../web/features/queue/behavior.js";
import { createQueueConnection } from "../web/features/queue/connection.js";
import { createPlayerComposition } from "../web/features/player/composition.js";

class FakeMedia extends EventTarget {
  constructor() {
    super();
    this.paused=true;
    this.ended=false;
    this.readyState=4;
    this.currentTime=0;
    this.src="";
    this.dataset={};
    this.playCalls=0;
    this.pauseCalls=0;
  }
  get currentSrc(){return this.src;}
  async play(){this.playCalls+=1;this.paused=false;this.ended=false;this.dispatchEvent(new Event("play"));}
  pause(){this.pauseCalls+=1;this.paused=true;this.dispatchEvent(new Event("pause"));}
}

function harness({repeat=false}={}){
  const media=new FakeMedia();
  const queue=createQueueConnection({initialState:QueueState.empty({repeat})});
  const catalog=new Map([
    ["a",{ref:"a",url:"media:a",playbackStartSeconds:0}],
    ["b",{ref:"b",url:"media:b",playbackStartSeconds:0}],
    ["c",{ref:"c",url:"media:c",playbackStartSeconds:0}],
  ]);
  const playback=createHTMLMediaPlaybackConnection({
    media,
    hasCurrentMedia:()=>currentQueueRef(queue.currentProjection.state)!==null,
    restartCurrent:async()=>{media.currentTime=0;}
  });
  const current=[];
  const player=createPlayerComposition({
    queue,playback,media,
    resolveMedia:ref=>catalog.get(ref),
    defaultContextRefs:()=>["a","b","c"],
    onCurrentMedia:item=>current.push(item.ref),
  });
  return {media,queue,playback,player,current};
}

test("top-level Play with no media explicitly seeds default Queue context then starts Playback", async()=>{
  const h=harness();
  assert.equal(h.playback.currentProjection.state.kind,"no-media");
  assert.equal(h.player.playPausePresentation.enabled,true);
  assert.equal(h.player.playPausePresentation.compositionRef,"player.seed-default-context");
  await h.player.activatePlayPause();
  assert.deepEqual(h.queue.currentProjection.state.items,["a","b","c"]);
  assert.equal(currentQueueRef(h.queue.currentProjection.state),"a");
  assert.equal(h.media.dataset.mediaRef,"a");
  assert.equal(h.media.playCalls,1);
  assert.equal(h.playback.currentProjection.control.textRef,"playback.pause");
});

test("Next changes Queue selection then realizes Playback for the selected media", async()=>{
  const h=harness(); await h.player.activatePlayPause();
  await h.player.next();
  assert.equal(currentQueueRef(h.queue.currentProjection.state),"b");
  assert.equal(h.media.dataset.mediaRef,"b");
  assert.equal(h.media.playCalls,2);
});

test("Previous after threshold restarts current media without moving Queue", async()=>{
  const h=harness(); await h.player.activatePlayPause(); await h.player.next();
  h.media.currentTime=5.2;
  const result=await h.player.previous();
  assert.equal(result.compositionRef,"player.restart-current");
  assert.equal(currentQueueRef(h.queue.currentProjection.state),"b");
  assert.equal(h.media.currentTime,0);
});

test("Previous near start moves Queue to previous item and plays it", async()=>{
  const h=harness(); await h.player.activatePlayPause(); await h.player.next();
  h.media.currentTime=1;
  await h.player.previous();
  assert.equal(currentQueueRef(h.queue.currentProjection.state),"a");
  assert.equal(h.media.dataset.mediaRef,"a");
});

test("Repeat makes Next wrap from final item through Queue semantics", async()=>{
  const h=harness({repeat:true});
  await h.player.playContext(["a","b"],1);
  await h.player.next();
  assert.equal(currentQueueRef(h.queue.currentProjection.state),"a");
  assert.equal(h.media.dataset.mediaRef,"a");
});

test("Shuffle and Repeat remain Queue controls rather than Playback state", async()=>{
  const h=harness(); await h.player.activatePlayPause();
  h.player.toggleShuffle(); h.player.toggleRepeat();
  assert.equal(h.queue.currentProjection.state.shuffle,true);
  assert.equal(h.queue.currentProjection.state.repeat,true);
  assert.equal(h.playback.currentProjection.state.kind,"media");
  assert.equal(h.playback.currentProjection.state.shuffle,undefined);
});

test("Play Track composes context selection with Playback instead of hiding it in Playback", async()=>{
  const h=harness();
  await h.player.playTrack("b",["a","b","c"]);
  assert.equal(currentQueueRef(h.queue.currentProjection.state),"b");
  assert.equal(h.media.dataset.mediaRef,"b");
  assert.equal(h.media.paused,false);
});
