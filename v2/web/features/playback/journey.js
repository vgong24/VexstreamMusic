// [VXG RealForever]
export const playbackJourney=Object.freeze({
 ref:"journey.vexstream.desktop.playback.basic",
 screenRef:"vexstream.desktop.playback",
 title:"Play and pause one known track",
 steps:Object.freeze([
  Object.freeze({kind:"observe",controlRef:"vexstream.playback.play-pause",expectTextRef:"playback.play"}),
  Object.freeze({kind:"activate",controlRef:"vexstream.playback.play-pause",expectTextRef:"playback.pause"}),
  Object.freeze({kind:"activate",controlRef:"vexstream.playback.play-pause",expectTextRef:"playback.play"})
 ])
});
