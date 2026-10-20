// [VXG RealForever]
import {
  QueueState,
  resolveQueueControl,
  vexStreamQueueBehavior,
} from "./behavior.js";
import {
  appendQueueRefs,
  applyQueueEffect,
  insertNextQueueRefs,
  moveQueueItem,
  playNowQueueRef,
  removeQueueItem,
  replaceQueueContext,
  reshuffleQueueUpcoming,
  selectQueueIndex,
} from "./model.js";
import { presentQueueControls } from "./presentation.js";

export function createQueueConnection({
  initialState = QueueState.empty(),
  shuffle = value => value,
} = {}) {
  let state = initialState;
  const listeners = new Set();

  function projection() {
    return Object.freeze({
      state,
      controls: presentQueueControls(state),
    });
  }

  function publish() {
    const current = projection();
    for (const listener of listeners) listener(current);
    return current;
  }

  function setState(next) {
    state = next;
    return publish();
  }

  function activate(control) {
    const act = resolveQueueControl(state, control);
    if (!act) return null;
    const relationship = vexStreamQueueBehavior.resolve(state, act);
    if (!relationship) return null;
    setState(applyQueueEffect(state, relationship.effect, shuffle));
    return relationship;
  }

  return Object.freeze({
    get currentProjection() {
      return projection();
    },

    activate,

    replaceContext(refs, startIndex = 0) {
      return setState(replaceQueueContext(state, refs, startIndex, shuffle));
    },

    restore({
      items = [],
      currentIndex = -1,
      shuffle: restoredShuffle = false,
      repeat: restoredRepeat = false,
    } = {}) {
      const refs = [...items];
      return setState(
        refs.length
          ? QueueState.positioned({
              items: refs,
              currentIndex,
              shuffle: restoredShuffle,
              repeat: restoredRepeat,
            })
          : QueueState.empty({
              shuffle: restoredShuffle,
              repeat: restoredRepeat,
            }),
      );
    },

    append(refs) {
      return setState(appendQueueRefs(state, refs));
    },

    playNext(refs) {
      return setState(insertNextQueueRefs(state, refs));
    },

    playNow(ref) {
      const next=playNowQueueRef(state,ref);
      return next===state?projection():setState(next);
    },

    shuffleUpcoming() {
      return setState(reshuffleQueueUpcoming(state, shuffle));
    },

    select(index) {
      return setState(selectQueueIndex(state, index));
    },

    move(from, to) {
      return setState(moveQueueItem(state, from, to));
    },

    remove(index) {
      return setState(removeQueueItem(state, index));
    },

    subscribe(listener) {
      if (typeof listener !== "function") throw new Error("Queue subscriber must be a function");
      listeners.add(listener);
      listener(projection());
      return () => listeners.delete(listener);
    },
  });
}
