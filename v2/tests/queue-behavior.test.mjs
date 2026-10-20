import assert from "node:assert/strict";
import test from "node:test";
import {
  QueueAct, QueueControl, QueueEffect, QueueState,
  currentQueueRef, resolveQueueControl, vexStreamQueueBehavior
} from "../web/features/queue/behavior.js";
import {
  appendQueueRefs, applyQueueEffect, insertNextQueueRefs,
  moveQueueItem, removeQueueItem, replaceQueueContext
} from "../web/features/queue/model.js";
import { createQueueConnection } from "../web/features/queue/connection.js";

const reverse = values => [...values].reverse();

test("Next advances while another item exists", () => {
  const state=QueueState.positioned({items:["a","b","c"],currentIndex:0});
  assert.equal(resolveQueueControl(state,QueueControl.Next),QueueAct.Next);
  assert.equal(vexStreamQueueBehavior.resolve(state,QueueAct.Next).effect,QueueEffect.Advance);
  assert.equal(applyQueueEffect(state,QueueEffect.Advance).currentIndex,1);
});

test("Next is unavailable at the end until repeat is on", () => {
  const stopped=QueueState.positioned({items:["a","b"],currentIndex:1,repeat:false});
  assert.equal(resolveQueueControl(stopped,QueueControl.Next),null);
  const looping=QueueState.positioned({items:["a","b"],currentIndex:1,repeat:true});
  assert.equal(vexStreamQueueBehavior.resolve(looping,QueueAct.Next).effect,QueueEffect.Wrap);
  assert.equal(currentQueueRef(applyQueueEffect(looping,QueueEffect.Wrap)),"a");
});

test("Previous only owns moving to an earlier queue item", () => {
  const first=QueueState.positioned({items:["a","b"],currentIndex:0});
  const second=QueueState.positioned({items:["a","b"],currentIndex:1});
  assert.equal(resolveQueueControl(first,QueueControl.Previous),null);
  assert.equal(vexStreamQueueBehavior.resolve(second,QueueAct.Previous).effect,QueueEffect.MovePrevious);
});

test("enabling shuffle preserves history/current and shuffles upcoming only", () => {
  const state=QueueState.positioned({items:["a","b","c","d"],currentIndex:1});
  const next=applyQueueEffect(state,QueueEffect.ShuffleUpcoming,reverse);
  assert.deepEqual(next.items,["a","b","d","c"]);
  assert.equal(next.currentIndex,1);
  assert.equal(next.shuffle,true);
});

test("disabling shuffle does not reconstruct the old order", () => {
  const state=QueueState.positioned({items:["a","b","d","c"],currentIndex:1,shuffle:true});
  const next=applyQueueEffect(state,QueueEffect.DisableShuffle,reverse);
  assert.deepEqual(next.items,["a","b","d","c"]);
  assert.equal(next.shuffle,false);
});

test("fresh shuffled context keeps selected current and makes every other item upcoming", () => {
  const flags=QueueState.empty({shuffle:true});
  const next=replaceQueueContext(flags,["a","b","c","d"],1,reverse);
  assert.deepEqual(next.items,["b","d","c","a"]);
  assert.equal(next.currentIndex,0);
  assert.equal(currentQueueRef(next),"b");
});

test("append to empty queue establishes first current item without playback semantics", () => {
  const next=appendQueueRefs(QueueState.empty(),["a","b"]);
  assert.equal(next.currentIndex,0);
  assert.deepEqual(next.items,["a","b"]);
});

test("play-next insertion is directly after the current item", () => {
  const state=QueueState.positioned({items:["a","d"],currentIndex:0});
  const next=insertNextQueueRefs(state,["b","c"]);
  assert.deepEqual(next.items,["a","b","c","d"]);
  assert.equal(next.currentIndex,0);
});

test("reorder can move current or adjacent items while preserving current identity", () => {
  const state=QueueState.positioned({items:["a","b","c","d"],currentIndex:2});

  const movedCurrent=moveQueueItem(state,2,0);
  assert.deepEqual(movedCurrent.items,["c","a","b","d"]);
  assert.equal(currentQueueRef(movedCurrent),"c");
  assert.equal(movedCurrent.currentIndex,0);

  const movedAdjacent=moveQueueItem(state,3,2);
  assert.deepEqual(movedAdjacent.items,["a","b","d","c"]);
  assert.equal(currentQueueRef(movedAdjacent),"c");
  assert.equal(movedAdjacent.currentIndex,3);
});

test("removing an item before current adjusts index without changing current identity", () => {
  const state=QueueState.positioned({items:["a","b","c"],currentIndex:2});
  const next=removeQueueItem(state,0);
  assert.deepEqual(next.items,["b","c"]);
  assert.equal(next.currentIndex,1);
  assert.equal(currentQueueRef(next),"c");
});

test("clear-after-current preserves history and current", () => {
  const state=QueueState.positioned({items:["a","b","c","d"],currentIndex:2});
  const next=applyQueueEffect(state,QueueEffect.ClearAfterCurrent);
  assert.deepEqual(next.items,["a","b","c"]);
  assert.equal(currentQueueRef(next),"c");
});

test("Queue Connection keeps payload operations separate from stable Control semantics", () => {
  const queue=createQueueConnection({shuffle:reverse});
  queue.replaceContext(["a","b","c"],0);
  const relation=queue.activate(QueueControl.Shuffle);
  assert.equal(relation.effect,QueueEffect.ShuffleUpcoming);
  assert.deepEqual(queue.currentProjection.state.items,["a","c","b"]);
  queue.playNext(["x"]);
  assert.deepEqual(queue.currentProjection.state.items,["a","x","c","b"]);
});


test("Queue Connection hydration preserves durable flags and order without reshuffling", () => {
  const queue=createQueueConnection({shuffle:reverse});
  queue.restore({
    items:["a","b","d","c"],
    currentIndex:1,
    shuffle:true,
    repeat:true
  });

  const state=queue.currentProjection.state;
  assert.deepEqual(state.items,["a","b","d","c"]);
  assert.equal(state.currentIndex,1);
  assert.equal(currentQueueRef(state),"b");
  assert.equal(state.shuffle,true);
  assert.equal(state.repeat,true);
});

test("Queue Connection hydration rejects an invalid durable current witness", () => {
  const queue=createQueueConnection();
  assert.throws(
    () => queue.restore({
      items:["a"],
      currentIndex:1
    }),
    /currentIndex must address an item/
  );
});
