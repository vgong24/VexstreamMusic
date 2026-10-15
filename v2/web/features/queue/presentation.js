// [VXG RealForever]
import { QueueControl, resolveQueueControl } from "./behavior.js";

const TEXT = new Map([
  [QueueControl.Previous, "queue.previous"],
  [QueueControl.Next, "queue.next"],
  [QueueControl.Shuffle, "queue.shuffle"],
  [QueueControl.Repeat, "queue.repeat"],
  [QueueControl.ClearAfterCurrent, "queue.clear-after-current"],
]);

export function presentQueueControl(currentState, control) {
  const act = resolveQueueControl(currentState, control);
  return Object.freeze({
    control,
    act,
    enabled: act !== null,
    selected:
      control === QueueControl.Shuffle ? currentState.shuffle :
      control === QueueControl.Repeat ? currentState.repeat :
      false,
    textRef: TEXT.get(control),
  });
}

export function presentQueueControls(currentState) {
  return Object.freeze({
    previous: presentQueueControl(currentState, QueueControl.Previous),
    next: presentQueueControl(currentState, QueueControl.Next),
    shuffle: presentQueueControl(currentState, QueueControl.Shuffle),
    repeat: presentQueueControl(currentState, QueueControl.Repeat),
    clearAfterCurrent: presentQueueControl(currentState, QueueControl.ClearAfterCurrent),
  });
}
