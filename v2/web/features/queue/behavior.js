// [VXG RealForever]
import { defineBehaviorFamily, semanticToken, stateOf } from "../../architecture/behavior.js";

export const VexStreamQueueFamily = "vexstream.queue";

export const QueueControl = Object.freeze({
  Previous: semanticToken(VexStreamQueueFamily, "control", "previous"),
  Next: semanticToken(VexStreamQueueFamily, "control", "next"),
  Shuffle: semanticToken(VexStreamQueueFamily, "control", "shuffle"),
  Repeat: semanticToken(VexStreamQueueFamily, "control", "repeat"),
  ClearAfterCurrent: semanticToken(VexStreamQueueFamily, "control", "clear-after-current"),
});

export const QueueAct = Object.freeze({
  Previous: semanticToken(VexStreamQueueFamily, "act", "previous"),
  Next: semanticToken(VexStreamQueueFamily, "act", "next"),
  EnableShuffle: semanticToken(VexStreamQueueFamily, "act", "enable-shuffle"),
  DisableShuffle: semanticToken(VexStreamQueueFamily, "act", "disable-shuffle"),
  EnableRepeat: semanticToken(VexStreamQueueFamily, "act", "enable-repeat"),
  DisableRepeat: semanticToken(VexStreamQueueFamily, "act", "disable-repeat"),
  ClearAfterCurrent: semanticToken(VexStreamQueueFamily, "act", "clear-after-current"),
});

export const QueueEffect = Object.freeze({
  MovePrevious: semanticToken(VexStreamQueueFamily, "effect", "move-previous"),
  Advance: semanticToken(VexStreamQueueFamily, "effect", "advance"),
  Wrap: semanticToken(VexStreamQueueFamily, "effect", "wrap"),
  ShuffleUpcoming: semanticToken(VexStreamQueueFamily, "effect", "shuffle-upcoming"),
  DisableShuffle: semanticToken(VexStreamQueueFamily, "effect", "disable-shuffle"),
  EnableRepeat: semanticToken(VexStreamQueueFamily, "effect", "enable-repeat"),
  DisableRepeat: semanticToken(VexStreamQueueFamily, "effect", "disable-repeat"),
  ClearAfterCurrent: semanticToken(VexStreamQueueFamily, "effect", "clear-after-current"),
});

function flags({ shuffle = false, repeat = false } = {}) {
  return { shuffle: Boolean(shuffle), repeat: Boolean(repeat) };
}

export const QueueState = Object.freeze({
  empty(options = {}) {
    return stateOf(VexStreamQueueFamily, "empty", {
      items: Object.freeze([]),
      currentIndex: -1,
      ...flags(options),
    });
  },

  positioned({ items, currentIndex, shuffle = false, repeat = false }) {
    const refs = [...items];
    if (!refs.length) return QueueState.empty({ shuffle, repeat });
    if (!Number.isInteger(currentIndex) || currentIndex < 0 || currentIndex >= refs.length) {
      throw new Error("vexstream.queue: currentIndex must address an item");
    }
    return stateOf(VexStreamQueueFamily, "positioned", {
      items: Object.freeze(refs),
      currentIndex,
      shuffle: Boolean(shuffle),
      repeat: Boolean(repeat),
    });
  },
});

export const vexStreamQueueBehavior = defineBehaviorFamily({
  ref: VexStreamQueueFamily,
  relationships: [
    { stateKind: "positioned", act: QueueAct.Previous, effect: QueueEffect.MovePrevious, when: current => current.currentIndex > 0 },
    { stateKind: "positioned", act: QueueAct.Next, effect: QueueEffect.Advance, when: current => current.currentIndex < current.items.length - 1 },
    { stateKind: "positioned", act: QueueAct.Next, effect: QueueEffect.Wrap, when: current => current.currentIndex === current.items.length - 1 && current.repeat },
    { stateKind: "empty", act: QueueAct.EnableShuffle, effect: QueueEffect.ShuffleUpcoming, when: current => !current.shuffle },
    { stateKind: "positioned", act: QueueAct.EnableShuffle, effect: QueueEffect.ShuffleUpcoming, when: current => !current.shuffle },
    { stateKind: "empty", act: QueueAct.DisableShuffle, effect: QueueEffect.DisableShuffle, when: current => current.shuffle },
    { stateKind: "positioned", act: QueueAct.DisableShuffle, effect: QueueEffect.DisableShuffle, when: current => current.shuffle },
    { stateKind: "empty", act: QueueAct.EnableRepeat, effect: QueueEffect.EnableRepeat, when: current => !current.repeat },
    { stateKind: "positioned", act: QueueAct.EnableRepeat, effect: QueueEffect.EnableRepeat, when: current => !current.repeat },
    { stateKind: "empty", act: QueueAct.DisableRepeat, effect: QueueEffect.DisableRepeat, when: current => current.repeat },
    { stateKind: "positioned", act: QueueAct.DisableRepeat, effect: QueueEffect.DisableRepeat, when: current => current.repeat },
    { stateKind: "positioned", act: QueueAct.ClearAfterCurrent, effect: QueueEffect.ClearAfterCurrent, when: current => current.currentIndex < current.items.length - 1 },
  ],
});

export function resolveQueueControl(currentState, control) {
  if (control?.familyRef !== VexStreamQueueFamily || control?.role !== "control") {
    throw new Error("vexstream.queue: Control belongs to another family or role");
  }

  if (control === QueueControl.Previous) {
    return currentState.kind === "positioned" && currentState.currentIndex > 0
      ? QueueAct.Previous
      : null;
  }

  if (control === QueueControl.Next) {
    return currentState.kind === "positioned" &&
      (currentState.currentIndex < currentState.items.length - 1 || currentState.repeat)
      ? QueueAct.Next
      : null;
  }

  if (control === QueueControl.Shuffle) {
    return currentState.shuffle ? QueueAct.DisableShuffle : QueueAct.EnableShuffle;
  }

  if (control === QueueControl.Repeat) {
    return currentState.repeat ? QueueAct.DisableRepeat : QueueAct.EnableRepeat;
  }

  if (control === QueueControl.ClearAfterCurrent) {
    return currentState.kind === "positioned" && currentState.currentIndex < currentState.items.length - 1
      ? QueueAct.ClearAfterCurrent
      : null;
  }

  throw new Error(`vexstream.queue: unknown Control ${control?.ref ?? control}`);
}

export function currentQueueRef(state) {
  return state.kind === "positioned" ? state.items[state.currentIndex] ?? null : null;
}
