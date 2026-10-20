// [VXG RealForever]
// Bounded DOM projection over whole Queue truth.
//
// The Queue remains semantically complete and durable. This module only decides which rows should
// be materialized in the browser at one time, in the same spirit as a RecyclerView window.

export const QueueWindowPolicy=Object.freeze({
 size:100,
 lead:25,
 advanceAt:.75,
 retreatAt:.15
});

function boundedLength(value){
 return Math.max(0,Math.floor(Number(value)||0));
}

function boundedIndex(value,length){
 if(!length)return 0;
 return Math.max(0,Math.min(Math.floor(Number(value)||0),length-1));
}

function maxStart(length,size){
 return Math.max(0,length-size);
}

export function queueWindowAroundIndex(
 length,
 index,
 policy=QueueWindowPolicy
){
 const total=boundedLength(length);
 const size=Math.max(1,boundedLength(policy?.size)||100);
 const lead=Math.max(0,boundedLength(policy?.lead)||0);
 const current=boundedIndex(index,total);
 const start=Math.max(
  0,
  Math.min(current-lead,maxStart(total,size))
 );
 return Object.freeze({
  start,
  end:Math.min(total,start+size),
  size
 });
}

export function queueWindowForScroll({
 length,
 start=0,
 firstVisible=0,
 policy=QueueWindowPolicy
}={}){
 const total=boundedLength(length);
 const size=Math.max(1,boundedLength(policy?.size)||100);
 const lead=Math.max(0,boundedLength(policy?.lead)||0);
 const currentStart=Math.max(
  0,
  Math.min(boundedLength(start),maxStart(total,size))
 );
 const visible=boundedIndex(firstVisible,total);
 const progress=size
  ?(visible-currentStart)/size
  :0;
 const shouldAdvance=progress>=Number(policy?.advanceAt??.75);
 const shouldRetreat=progress<=Number(policy?.retreatAt??.15);
 const nextStart=(shouldAdvance||shouldRetreat)
  ?Math.max(
    0,
    Math.min(visible-lead,maxStart(total,size))
   )
  :currentStart;
 return Object.freeze({
  start:nextStart,
  end:Math.min(total,nextStart+size),
  size,
  changed:nextStart!==currentStart
 });
}
