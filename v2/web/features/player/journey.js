// [VXG RealForever]
export const playerJourney = Object.freeze({
  ref: "journey.vexstream.desktop.player.queue-basics",
  screenRef: "vexstream.desktop.player-practicum",
  title: "Start listening and move through the queue",
  steps: Object.freeze([
    Object.freeze({ kind: "observe", controlRef: "vexstream.playback.play-pause", expectTextRef: "playback.play" }),
    Object.freeze({ kind: "activate", controlRef: "vexstream.playback.play-pause", expectTextRef: "playback.pause", expectCurrentRef: "media.vexstream.test-tone.1" }),
    Object.freeze({ kind: "activate", controlRef: "vexstream.queue.next", expectCurrentRef: "media.vexstream.test-tone.2" }),
    Object.freeze({ kind: "activate", controlRef: "vexstream.queue.previous", expectCurrentRef: "media.vexstream.test-tone.1" }),
    Object.freeze({ kind: "activate", controlRef: "vexstream.queue.shuffle", expectSelected: true }),
    Object.freeze({ kind: "activate", controlRef: "vexstream.queue.repeat", expectSelected: true }),
  ]),
});
