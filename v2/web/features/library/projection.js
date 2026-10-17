// [VXG RealForever]
// Pure browser projections over current Library truth.
//
// Search/sort/grouping answer "how should the current catalog be viewed?" They never mutate Library,
// Queue, source configuration, or durable metadata.

const SORT_KEYS=new Set([
 "title",
 "added",
 "artist",
 "album",
 "genre",
 "duration"
]);

const SORT_PRESETS=new Set([
 "title",
 "artist",
 "album",
 "genre",
 "recent",
 "duration",
 "duration-desc"
]);

export function librarySortStackForPreset(value="title"){
 const preset=SORT_PRESETS.has(value)?value:"title";
 switch(preset){
 case "artist":
  return freezeSortStack([{key:"artist",dir:"asc"},{key:"title",dir:"asc"}]);
 case "album":
  return freezeSortStack([{key:"album",dir:"asc"},{key:"title",dir:"asc"}]);
 case "genre":
  return freezeSortStack([{key:"genre",dir:"asc"},{key:"title",dir:"asc"}]);
 case "recent":
  return freezeSortStack([{key:"added",dir:"desc"},{key:"title",dir:"asc"}]);
 case "duration":
  return freezeSortStack([{key:"duration",dir:"asc"},{key:"title",dir:"asc"}]);
 case "duration-desc":
  return freezeSortStack([{key:"duration",dir:"desc"},{key:"title",dir:"asc"}]);
 default:
  return freezeSortStack([{key:"title",dir:"asc"}]);
 }
}

export function activateLibrarySortStack(stack,key){
 const sortKey=String(key??"").trim();
 if(!SORT_KEYS.has(sortKey)){
  throw new Error("Library sort supports title, added, artist, album, genre, or duration");
 }
 const next=normalizeSortStack(stack).map(spec=>({...spec}));
 const index=next.findIndex(spec=>spec.key===sortKey);
 if(index===0){
  next[0].dir=next[0].dir==="asc"?"desc":"asc";
 }else if(index>0){
  const [spec]=next.splice(index,1);
  next.unshift(spec);
 }else{
  next.unshift({
   key:sortKey,
   dir:sortKey==="added"?"desc":"asc"
  });
 }
 return freezeSortStack(next.slice(0,4));
}

export function resetLibrarySortStack(){
 return librarySortStackForPreset("title");
}

export function librarySortPresetForStack(stack){
 const normalized=normalizeSortStack(stack);
 for(const preset of SORT_PRESETS){
  if(sortStacksEqual(
   normalized,
   librarySortStackForPreset(preset)
  ))return preset;
 }
 return "custom";
}

export function projectLibraryView(
 catalog,
 {
  query="",
  genre="all",
  sort="title",
  sortStack=null
 }={}
){
 const source=Array.isArray(catalog)?catalog:[];
 const needle=normalize(query);
 const wantedGenre=normalize(genre);
 const activeSortStack=Array.isArray(sortStack)
  ?normalizeSortStack(sortStack,{fallback:sort})
  :librarySortStackForPreset(sort);

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

 visible.sort(comparator(activeSortStack));
 return Object.freeze([...visible]);
}

export function projectLibraryHome(
 catalog,
 {recentLimit=8}={}
){
 const source=Array.isArray(catalog)?catalog:[];
 const limit=Math.max(0,Number(recentLimit)||0);
 return Object.freeze({
  songCount:source.length,
  albumCount:groupLibrary(source,"album").length,
  artistCount:groupLibrary(source,"artist").length,
  recent:Object.freeze(
   projectLibraryView(source,{sort:"recent"}).slice(0,limit)
  )
 });
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

function comparator(stack){
 return (left,right)=>{
  for(const spec of stack){
   const compared=compareSortKey(spec.key,left,right);
   if(compared){
    return spec.dir==="desc"?-compared:compared;
   }
  }
  if(!stack.some(spec=>spec.key==="title")){
   const byTitle=compareText(left?.title,right?.title);
   if(byTitle)return byTitle;
  }
  return compareText(left?.ref,right?.ref);
 };
}

function compareSortKey(key,left,right){
 switch(key){
 case "added":
  return dateValue(left?.addedAt)-dateValue(right?.addedAt);
 case "artist":
  return compareText(left?.artist,right?.artist);
 case "album":
  return compareText(left?.album,right?.album);
 case "genre":
  return compareText(left?.genre,right?.genre);
 case "duration":
  return (Number(left?.durationSeconds)||0)-(Number(right?.durationSeconds)||0);
 default:
  return compareText(left?.title,right?.title);
 }
}

function normalizeSortStack(stack,{fallback="title"}={}){
 const out=[];
 const seen=new Set();
 for(const value of Array.isArray(stack)?stack:[]){
  const key=String(value?.key??"").trim();
  const dir=value?.dir==="desc"?"desc":"asc";
  if(!SORT_KEYS.has(key)||seen.has(key))continue;
  seen.add(key);
  out.push({key,dir});
  if(out.length===4)break;
 }
 return out.length
  ?freezeSortStack(out)
  :librarySortStackForPreset(fallback);
}

function freezeSortStack(values){
 return Object.freeze(values.map(value=>Object.freeze({
  key:value.key,
  dir:value.dir==="desc"?"desc":"asc"
 })));
}

function sortStacksEqual(left,right){
 return left.length===right.length&&left.every(
  (spec,index)=>
   spec.key===right[index].key&&
   spec.dir===right[index].dir
 );
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
