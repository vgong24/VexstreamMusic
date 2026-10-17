// [VXG RealForever]
import { QueueState, QueueEffect } from "./behavior.js";

function uniqueRefs(refs) {
  return [...refs].filter(ref => typeof ref === "string" && ref.length > 0);
}

export function shuffleUpcomingRefs(items, currentIndex, shuffleFn) {
  const start = currentIndex >= 0 ? currentIndex + 1 : 0;
  return [
    ...items.slice(0, start),
    ...shuffleFn([...items.slice(start)]),
  ];
}

export function replaceQueueContext(current, refs, startIndex = 0, shuffleFn = value => value) {
  const items = uniqueRefs(refs);
  if (!items.length) return QueueState.empty(current);
  const bounded = Math.max(0, Math.min(Number(startIndex) || 0, items.length - 1));
  const ordered = current.shuffle
    ? shuffleUpcomingRefs(items, bounded, shuffleFn)
    : items;
  return QueueState.positioned({
    items: ordered,
    currentIndex: bounded,
    shuffle: current.shuffle,
    repeat: current.repeat,
  });
}

export function appendQueueRefs(current, refs) {
  const additions = uniqueRefs(refs);
  if (!additions.length) return current;
  if (current.kind === "empty") {
    return QueueState.positioned({
      items: additions,
      currentIndex: 0,
      shuffle: current.shuffle,
      repeat: current.repeat,
    });
  }
  return QueueState.positioned({
    ...current,
    items: [...current.items, ...additions],
  });
}

export function insertNextQueueRefs(current, refs) {
  const additions = uniqueRefs(refs);
  if (!additions.length) return current;
  if (current.kind === "empty") {
    return QueueState.positioned({
      items: additions,
      currentIndex: 0,
      shuffle: current.shuffle,
      repeat: current.repeat,
    });
  }
  return QueueState.positioned({
    ...current,
    items: [
      ...current.items.slice(0, current.currentIndex + 1),
      ...additions,
      ...current.items.slice(current.currentIndex + 1),
    ],
  });
}

export function selectQueueIndex(current, index) {
  if (current.kind === "empty") return current;
  const length = current.items.length;
  const normalized = ((Number(index) % length) + length) % length;
  return QueueState.positioned({ ...current, currentIndex: normalized });
}

export function moveQueueItem(current, from, to) {
  if (current.kind === "empty") return current;
  if (!Number.isInteger(from) || from < 0 || from >= current.items.length) return current;
  if (from === current.currentIndex) return current;

  let target = Math.max(0, Math.min(Number(to) || 0, current.items.length));
  if (from === target || from + 1 === target) return current;

  const items = [...current.items];
  const [item] = items.splice(from, 1);
  if (from < target) target -= 1;
  target = Math.max(0, Math.min(target, items.length));

  let nextCurrent = current.currentIndex;
  if (from < nextCurrent) nextCurrent -= 1;
  if (target <= nextCurrent) nextCurrent += 1;

  items.splice(target, 0, item);
  return QueueState.positioned({ ...current, items, currentIndex: nextCurrent });
}

export function removeQueueItem(current, index) {
  if (current.kind === "empty") return current;
  if (!Number.isInteger(index) || index < 0 || index >= current.items.length) return current;
  if (index === current.currentIndex) return current;

  const items = [...current.items];
  items.splice(index, 1);
  if (!items.length) return QueueState.empty(current);

  const currentIndex = index < current.currentIndex
    ? current.currentIndex - 1
    : current.currentIndex;

  return QueueState.positioned({ ...current, items, currentIndex });
}

export function applyQueueEffect(current, effect, shuffleFn = value => value) {
  if (effect === QueueEffect.MovePrevious) {
    return current.kind === "positioned"
      ? QueueState.positioned({ ...current, currentIndex: current.currentIndex - 1 })
      : current;
  }

  if (effect === QueueEffect.Advance) {
    return current.kind === "positioned"
      ? QueueState.positioned({ ...current, currentIndex: current.currentIndex + 1 })
      : current;
  }

  if (effect === QueueEffect.Wrap) {
    return current.kind === "positioned"
      ? QueueState.positioned({ ...current, currentIndex: 0 })
      : current;
  }

  if (effect === QueueEffect.ShuffleUpcoming) {
    if (current.kind === "empty") return QueueState.empty({ ...current, shuffle: true });
    return QueueState.positioned({
      ...current,
      items: shuffleUpcomingRefs(current.items, current.currentIndex, shuffleFn),
      shuffle: true,
    });
  }

  if (effect === QueueEffect.DisableShuffle) {
    return current.kind === "empty"
      ? QueueState.empty({ ...current, shuffle: false })
      : QueueState.positioned({ ...current, shuffle: false });
  }

  if (effect === QueueEffect.EnableRepeat) {
    return current.kind === "empty"
      ? QueueState.empty({ ...current, repeat: true })
      : QueueState.positioned({ ...current, repeat: true });
  }

  if (effect === QueueEffect.DisableRepeat) {
    return current.kind === "empty"
      ? QueueState.empty({ ...current, repeat: false })
      : QueueState.positioned({ ...current, repeat: false });
  }

  if (effect === QueueEffect.ClearAfterCurrent) {
    if (current.kind === "empty") return current;
    return QueueState.positioned({
      ...current,
      items: current.items.slice(0, current.currentIndex + 1),
    });
  }

  throw new Error(`vexstream.queue: unknown Effect ${effect?.ref ?? effect}`);
}
