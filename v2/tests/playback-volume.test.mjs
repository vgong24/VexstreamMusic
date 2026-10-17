import assert from "node:assert/strict";
import test from "node:test";

import {
 createHTMLMediaPlaybackVolumeConnection,
 projectPlaybackVolume
} from "../web/features/playback/volume.js";

class FakeMedia extends EventTarget{
 constructor(){
  super();
  this.volume=1;
 }
}

test("Volume projects current HTMLMediaElement provider truth",()=>{
 const media=new FakeMedia();
 media.volume=.35;
 assert.deepEqual(
  projectPlaybackVolume(media),
  {volume:.35,percent:35}
 );
});

test("Volume requests clamp into the provider's 0 to 1 range",()=>{
 const media=new FakeMedia();
 const volume=createHTMLMediaPlaybackVolumeConnection({media});
 volume.setVolume(2);
 assert.equal(media.volume,1);
 assert.equal(volume.currentProjection.percent,100);
 volume.setVolume(-1);
 assert.equal(media.volume,0);
 assert.equal(volume.currentProjection.percent,0);
 volume.close();
});

test("provider-side volume changes publish current truth",()=>{
 const media=new FakeMedia();
 const volume=createHTMLMediaPlaybackVolumeConnection({media});
 const observed=[];
 volume.subscribe(projection=>observed.push(projection.percent));
 media.volume=.62;
 media.dispatchEvent(new Event("volumechange"));
 assert.equal(observed.at(-1),62);
 volume.close();
});

test("V3 default output level is an explicit one-time product initialization",()=>{
 const media=new FakeMedia();
 const volume=createHTMLMediaPlaybackVolumeConnection({
  media,
  initialVolume:.8
 });
 assert.equal(media.volume,.8);
 assert.equal(volume.currentProjection.percent,80);
 volume.close();
});

test("non-finite volume requests fail instead of inventing provider meaning",()=>{
 const media=new FakeMedia();
 const volume=createHTMLMediaPlaybackVolumeConnection({media});
 assert.throws(()=>volume.setVolume(Number.NaN),/finite level/);
 volume.close();
});
