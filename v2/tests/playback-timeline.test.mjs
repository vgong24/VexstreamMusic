import assert from "node:assert/strict";
import test from "node:test";

import {
 createHTMLMediaPlaybackTimelineConnection,
 projectPlaybackTimeline
} from "../web/features/playback/timeline.js";

class FakeMedia extends EventTarget{
 constructor(){
  super();
  this.duration=120;
  this.currentTime=0;
 }
}

test("Timeline projects provider position relative to the admitted playback window",()=>{
 const media=new FakeMedia();
 media.currentTime=35;
 const projection=projectPlaybackTimeline(media,{
  startSeconds:20,
  endSeconds:80
 });
 assert.equal(projection.kind,"ready");
 assert.equal(projection.startSeconds,20);
 assert.equal(projection.endSeconds,80);
 assert.equal(projection.elapsedSeconds,15);
 assert.equal(projection.durationSeconds,60);
 assert.equal(projection.normalized,.25);
 assert.equal(projection.windowEndReached,false);
});

test("Timeline normalized seek maps back into provider time without owning Library truth",()=>{
 const media=new FakeMedia();
 const window=Object.freeze({startSeconds:20,endSeconds:80});
 const connection=createHTMLMediaPlaybackTimelineConnection({
  media,
  currentWindow:()=>window
 });
 const projection=connection.seekNormalized(.5);
 assert.equal(media.currentTime,50);
 assert.equal(projection.elapsedSeconds,30);
 assert.equal(projection.normalized,.5);
 assert.deepEqual(window,{startSeconds:20,endSeconds:80});
 connection.close();
});

test("Timeline clamps seek requests to the effective playback window",()=>{
 const media=new FakeMedia();
 const connection=createHTMLMediaPlaybackTimelineConnection({
  media,
  currentWindow:()=>({startSeconds:10,endSeconds:50})
 });
 connection.seekNormalized(2);
 assert.equal(media.currentTime,50);
 connection.seekNormalized(-1);
 assert.equal(media.currentTime,10);
 connection.close();
});

test("only an explicit early playback-window end becomes a window-end signal",()=>{
 const media=new FakeMedia();
 media.currentTime=79.99;
 const cropped=projectPlaybackTimeline(media,{
  startSeconds:20,
  endSeconds:80
 });
 assert.equal(cropped.windowEndReached,true);

 media.currentTime=119.99;
 const full=projectPlaybackTimeline(media,{
  startSeconds:20,
  endSeconds:null
 });
 assert.equal(full.windowEndReached,false);
});

test("Timeline is unavailable until current media and provider duration are both real",()=>{
 const media=new FakeMedia();
 media.duration=Number.NaN;
 assert.equal(
  projectPlaybackTimeline(media,{startSeconds:0,endSeconds:null}).kind,
  "unavailable"
 );
 assert.equal(projectPlaybackTimeline(media,null).kind,"unavailable");
});
