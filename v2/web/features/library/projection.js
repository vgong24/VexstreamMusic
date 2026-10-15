// [VXG RealForever]
// Pure browser projections over current Library truth.
//
// Search/sort/grouping answer "how should the current catalog be viewed?" They never mutate Library,
// Queue, source configuration, or durable metadata.

const SORTS=new Set([
 "title",
 "artist",
 "album",
 "recent",
 "duration-desc"
]);

export function projectLibraryView(
 catalog,
 {
  query="",
  genre="all",
  sort="title"
 }={}
){
 const source=Array.isArray(catalog)?catalog:[];
 const needle=normalize(query);
 const wantedGenre=normalize(genre);
 const sortRef=SORTS.has(sort)?sort:"title";

 const visible=source.filter(item=>{
  if(
   wantedGenre &&
   wantedGenre!=="all" &&
   normalize(item?.genre)!==wantedGenre
  ){
   return false;
  }
  if(!needle)return true;
  return searchableText(item).includes(needle);
 });

 visible.sort(comparator(sortRef));
 return Object.freeze([...visible]);
}

export function libraryGenres(catalog){
 const seen=new Map();
 for(const item of Array.isArray(catalog)?catalog:[]){
  const value=String(item?.genre??"").trim();
  if(!value)continue;
  const key=normalize(value);
  if(!seen.has(key))seen.set(key,value);
 }
 return Object.freeze(
  [...seen.values()].sort((a,b)=>compareText(a,b))
 );
}

export function groupLibrary(catalog,key){
 if(key!=="album"&&key!=="artist"){
  throw new Error("Library grouping supports album or artist");
 }
 const groups=new Map();
 for(const item of Array.isArray(catalog)?catalog:[]){
  const label=String(item?.[key]??"").trim();
  const fallback=key==="album"?"Unknown album":"Unknown artist";
  const name=label||fallback;
  const normalized=normalize(name);
  const existing=groups.get(normalized)??{
   key:normalized,
   name,
   refs:[],
   tracks:[]
  };
  existing.refs.push(item.ref);
  existing.tracks.push(item);
  groups.set(normalized,existing);
 }
 return Object.freeze(
  [...groups.values()]
   .sort((a,b)=>compareText(a.name,b.name))
   .map(group=>Object.freeze({
    key:group.key,
    name:group.name,
    refs:Object.freeze([...group.refs]),
    tracks:Object.freeze([...group.tracks])
   }))
 );
}

function comparator(sort){
 return (left,right)=>{
  switch(sort){
  case "artist":{
   const byArtist=compareText(left?.artist,right?.artist);
   if(byArtist)return byArtist;
   break;
  }
  case "album":{
   const byAlbum=compareText(left?.album,right?.album);
   if(byAlbum)return byAlbum;
   break;
  }
  case "recent":{
   const leftTime=dateValue(left?.addedAt);
   const rightTime=dateValue(right?.addedAt);
   if(leftTime!==rightTime)return rightTime-leftTime;
   break;
  }
  case "duration-desc":{
   const leftDuration=Number(left?.durationSeconds)||0;
   const rightDuration=Number(right?.durationSeconds)||0;
   if(leftDuration!==rightDuration)return rightDuration-leftDuration;
   break;
  }
  }
  const byTitle=compareText(left?.title,right?.title);
  if(byTitle)return byTitle;
  return compareText(left?.ref,right?.ref);
 };
}

function searchableText(item){
 return normalize([
  item?.title,
  item?.artist,
  item?.album,
  item?.genre,
  item?.year,
  item?.channel,
  ...(Array.isArray(item?.tags)?item.tags:[]),
  ...(Array.isArray(item?.categories)?item.categories:[])
 ].filter(Boolean).join(" "));
}

function normalize(value){
 return String(value??"")
  .toLocaleLowerCase()
  .replace(/\s+/g," ")
  .trim();
}

function compareText(left,right){
 return String(left??"").localeCompare(
  String(right??""),
  undefined,
  {sensitivity:"base",numeric:true}
 );
}

function dateValue(value){
 const parsed=Date.parse(String(value??""));
 return Number.isFinite(parsed)?parsed:0;
}
