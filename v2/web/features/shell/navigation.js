// [VXG RealForever]
// Human desktop location is its own small product question.
//
// ShellNavigation owns which recognizable VexStream terrain is visible.
// It does not own Library data, Playback, Queue, Playlists, Trash, Explore,
// Imports, or provider behavior.

export const ShellView=Object.freeze({
 HOME:"home",
 SONGS:"songs",
 ALBUMS:"albums",
 ARTISTS:"artists",
 PLAYLISTS:"playlists",
 DELETED:"deleted",
 DISCOVER:"discover"
});

const VIEWS=new Set(Object.values(ShellView));
const LIBRARY_VIEWS=new Set([
 ShellView.SONGS,
 ShellView.ALBUMS,
 ShellView.ARTISTS
]);
const LIBRARY_DETAIL_VIEWS=Object.freeze({
 album:ShellView.ALBUMS,
 artist:ShellView.ARTISTS
});

export function shellPanelForView(view){
 const resolved=resolveShellView(view);
 return LIBRARY_VIEWS.has(resolved)?"library":resolved;
}

export function libraryModeForShellView(view){
 const resolved=resolveShellView(view);
 return LIBRARY_VIEWS.has(resolved)?resolved:null;
}

export function createShellNavigation({
 initialView=ShellView.HOME,
 sidebarOpen=true
}={}){
 let state=freezeState({
  view:resolveShellView(initialView),
  sidebarOpen:Boolean(sidebarOpen),
  libraryDetail:null
 });
 const listeners=new Set();

 function publish(){
  const projection=currentProjection();
  for(const listener of listeners)listener(projection);
  return projection;
 }

 function currentProjection(){
  return Object.freeze({
   state,
   view:state.view,
   panel:shellPanelForView(state.view),
   libraryMode:libraryModeForShellView(state.view),
   libraryDetail:state.libraryDetail,
   sidebarOpen:state.sidebarOpen
  });
 }

 return Object.freeze({
  get currentProjection(){return currentProjection();},
  select(view){
   const next=resolveShellView(view);
   if(next===state.view&&state.libraryDetail===null)return currentProjection();
   state=freezeState({...state,view:next,libraryDetail:null});
   return publish();
  },
  openLibraryDetail(kind,key){
   const detail=resolveLibraryDetail(kind,key);
   state=freezeState({
    ...state,
    view:LIBRARY_DETAIL_VIEWS[detail.kind],
    libraryDetail:detail
   });
   return publish();
  },
  closeLibraryDetail(){
   if(state.libraryDetail===null)return currentProjection();
   state=freezeState({...state,libraryDetail:null});
   return publish();
  },
  toggleSidebar(){
   state=freezeState({...state,sidebarOpen:!state.sidebarOpen});
   return publish();
  },
  subscribe(listener){
   listeners.add(listener);
   return ()=>listeners.delete(listener);
  }
 });
}

function resolveLibraryDetail(kind,key){
 const detailKind=String(kind??"").trim();
 if(!Object.hasOwn(LIBRARY_DETAIL_VIEWS,detailKind)){
  throw new Error("Library detail kind must be album or artist");
 }
 const detailKey=String(key??"").trim();
 if(!detailKey){
  throw new Error("Library detail key is required");
 }
 return Object.freeze({kind:detailKind,key:detailKey});
}

function resolveShellView(value){
 const view=String(value??"").trim();
 if(!VIEWS.has(view)){
  throw new Error(`Unknown VexStream shell view: ${view||"(empty)"}`);
 }
 return view;
}

function freezeState(state){
 return Object.freeze({...state});
}
