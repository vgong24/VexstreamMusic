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
  sidebarOpen:Boolean(sidebarOpen)
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
   sidebarOpen:state.sidebarOpen
  });
 }

 return Object.freeze({
  get currentProjection(){return currentProjection();},
  select(view){
   const next=resolveShellView(view);
   if(next===state.view)return currentProjection();
   state=freezeState({...state,view:next});
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
