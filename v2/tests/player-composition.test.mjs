import assert from "node:assert/strict";
import test from "node:test";
import { createHTMLMediaPlaybackConnection } from "../web/features/playback/connection.js";
import { createHTMLMediaPlaybackTimelineConnection } from "../web/features/playback/timeline.js";
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
    this.duration=120;
    this.src="";
    this.dataset={};
    this.playCalls=0;
    this.pauseCalls=0;
  }
  get currentSrc(){return this.src;}
  async play(){this.playCalls+=1;this.paused=false;this.ended=false;this.dispatchEvent(new Event("play"));}
  pause(){this.pauseCalls+=1;this.paused=true;this.dispatchEvent(new Event("pause"));}
}

function harness({repeat=false,catalogRows=null}={}){
  const media=new FakeMedia();
  const queue=createQueueConnection({initialState:QueueState.empty({repeat})});
  const catalog=new Map(catalogRows??[
    ["a",{ref:"a",url:"media:a",playbackStartSeconds:0,playbackEndSeconds:null}],
    ["b",{ref:"b",url:"media:b",playbackStartSeconds:0,playbackEndSeconds:null}],
    ["c",{ref:"c",url:"media:c",playbackStartSeconds:0,playbackEndSeconds:null}],
  ]);
  const playback=createHTMLMediaPlaybackConnection({
    media,
    hasCurrentMedia:()=>currentQueueRef(queue.currentProjection.state)!==null,
    restartCurrent:async()=>{media.currentTime=0;}
  });
  const timeline=createHTMLMediaPlaybackTimelineConnection({
    media,
    currentWindow:()=>{
      const item=catalog.get(currentQueueRef(queue.currentProjection.state));
      return item?{
        startSeconds:item.playbackStartSeconds,
        endSeconds:item.playbackEndSeconds
      }:null;
    }
  });
  const current=[];
  const player=createPlayerComposition({
    queue,playback,timeline,media,
    resolveMedia:ref=>catalog.get(ref),
    defaultContextRefs:()=>["a","b","c"],
    onCurrentMedia:item=>current.push(item.ref),
  });
  return {media,queue,playback,timeline,player,current};
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

test("Play now relocates selected media after current without replacing shuffled Queue", async()=>{
  const h=harness();
  await h.player.activatePlayPause();
  h.player.toggleShuffle();
  assert.deepEqual(h.queue.currentProjection.state.items,["a","b","c"]);
  await h.player.playNow("c",["a","b","c"]);
  assert.deepEqual(h.queue.currentProjection.state.items,["a","c","b"]);
  assert.equal(h.queue.currentProjection.state.shuffle,true);
  assert.equal(currentQueueRef(h.queue.currentProjection.state),"c");
  assert.equal(h.media.dataset.mediaRef,"c");

  h.media.currentTime=0;
  await h.player.previous();
  assert.equal(currentQueueRef(h.queue.currentProjection.state),"a");
  await h.player.next();
  assert.equal(currentQueueRef(h.queue.currentProjection.state),"c");
});

test("Play now on the already-playing item does not toggle it to Pause", async()=>{
  const h=harness();
  await h.player.activatePlayPause();
  const before=h.media.playCalls;
  await h.player.playNow("a",["a","b","c"]);
  assert.equal(h.media.paused,false);
  assert.equal(h.media.playCalls,before);
  assert.equal(currentQueueRef(h.queue.currentProjection.state),"a");
});

test("Play now moves a history occurrence instead of creating a duplicate", async()=>{
  const h=harness();
  await h.player.playContext(["a","b","c"],2);
  await h.player.playNow("a",["a","b","c"]);
  assert.deepEqual(h.queue.currentProjection.state.items,["b","c","a"]);
  assert.equal(currentQueueRef(h.queue.currentProjection.state),"a");
  assert.equal(h.queue.currentProjection.state.currentIndex,2);
  h.media.currentTime=0;
  await h.player.previous();
  assert.equal(currentQueueRef(h.queue.currentProjection.state),"c");
});

test("Play next relocates an existing future occurrence without increasing Queue length", async()=>{
  const h=harness();
  await h.player.playContext(["a","b","c"],0);
  await h.player.playNext(["c"]);
  assert.deepEqual(h.queue.currentProjection.state.items,["a","c","b"]);
  assert.equal(h.queue.currentProjection.state.items.length,3);
  assert.equal(currentQueueRef(h.queue.currentProjection.state),"a");
});

test("Add to queue on an empty Queue selects without auto-playing; Play then loads current media", async()=>{
  const h=harness();
  h.player.append(["b","c"]);
  assert.equal(currentQueueRef(h.queue.currentProjection.state),"b");
  assert.equal(h.media.playCalls,0);
  assert.equal(h.player.playPausePresentation.compositionRef,"player.load-current");
  await h.player.activatePlayPause();
  assert.equal(h.media.dataset.mediaRef,"b");
  assert.equal(h.media.playCalls,1);
});

test("Play next on an empty Queue preserves V1 behavior by starting the inserted context", async()=>{
  const h=harness();
  await h.player.playNext(["c","b"]);
  assert.deepEqual(h.queue.currentProjection.state.items,["c","b"]);
  assert.equal(currentQueueRef(h.queue.currentProjection.state),"c");
  assert.equal(h.media.dataset.mediaRef,"c");
  assert.equal(h.media.playCalls,1);
});

test("Play queue index is higher composition: Queue selection then Playback realization", async()=>{
  const h=harness();
  await h.player.activatePlayPause();
  await h.player.playQueueIndex(2);
  assert.equal(currentQueueRef(h.queue.currentProjection.state),"c");
  assert.equal(h.media.dataset.mediaRef,"c");
  assert.equal(h.media.paused,false);
});


test("explicit playback-window end advances through Queue composition instead of bypassing it",async()=>{
 const h=harness({
  catalogRows:[
   ["a",{ref:"a",url:"media:a",playbackStartSeconds:10,playbackEndSeconds:20}],
   ["b",{ref:"b",url:"media:b",playbackStartSeconds:30,playbackEndSeconds:null}]
  ]
 });
 await h.player.playContext(["a","b"],0);
 h.media.currentTime=20;
 h.media.dispatchEvent(new Event("timeupdate"));
 await new Promise(resolve=>setTimeout(resolve,0));
 assert.equal(currentQueueRef(h.queue.currentProjection.state),"b");
 assert.equal(h.media.dataset.mediaRef,"b");
 assert.equal(h.media.currentTime,30);
 assert.equal(h.media.paused,false);
 h.player.close();
 h.timeline.close();
 h.playback.close();
});

test("explicit playback-window end at Queue end stops through Playback owner",async()=>{
 const h=harness({
  catalogRows:[
   ["a",{ref:"a",url:"media:a",playbackStartSeconds:10,playbackEndSeconds:20}]
  ]
 });
 await h.player.playContext(["a"],0);
 assert.equal(h.media.paused,false);
 h.media.currentTime=20;
 h.media.dispatchEvent(new Event("timeupdate"));
 await new Promise(resolve=>setTimeout(resolve,0));
 assert.equal(currentQueueRef(h.queue.currentProjection.state),"a");
 assert.equal(h.media.paused,true);
 assert.equal(h.playback.currentProjection.control.textRef,"playback.play");
 h.player.close();
 h.timeline.close();
 h.playback.close();
});
