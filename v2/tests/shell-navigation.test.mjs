import assert from "node:assert/strict";
import test from "node:test";

import {
 ShellView,
 createShellNavigation,
 libraryModeForShellView,
 shellPanelForView
} from "../web/features/shell/navigation.js";

test("recognizable V3 library terrains resolve through one shell owner",()=>{
 assert.equal(shellPanelForView(ShellView.HOME),"home");
 assert.equal(shellPanelForView(ShellView.SONGS),"library");
 assert.equal(shellPanelForView(ShellView.ALBUMS),"library");
 assert.equal(shellPanelForView(ShellView.ARTISTS),"library");
 assert.equal(shellPanelForView(ShellView.PLAYLISTS),"playlists");
 assert.equal(shellPanelForView(ShellView.DELETED),"deleted");
 assert.equal(shellPanelForView(ShellView.DISCOVER),"discover");

 assert.equal(libraryModeForShellView(ShellView.SONGS),"songs");
 assert.equal(libraryModeForShellView(ShellView.ALBUMS),"albums");
 assert.equal(libraryModeForShellView(ShellView.ARTISTS),"artists");
 assert.equal(libraryModeForShellView(ShellView.HOME),null);
});

test("shell selection is independent from feature state and publishes current location",()=>{
 const shell=createShellNavigation();
 const observed=[];
 shell.subscribe(projection=>observed.push({
  view:projection.view,
  panel:projection.panel,
  libraryMode:projection.libraryMode
 }));

 assert.equal(shell.currentProjection.view,ShellView.HOME);
 shell.select(ShellView.SONGS);
 shell.select(ShellView.PLAYLISTS);

 assert.deepEqual(observed,[
  {view:"songs",panel:"library",libraryMode:"songs"},
  {view:"playlists",panel:"playlists",libraryMode:null}
 ]);
});

test("sidebar visibility is shell presentation state, not Library state",()=>{
 const shell=createShellNavigation({initialView:ShellView.ALBUMS});
 assert.equal(shell.currentProjection.sidebarOpen,true);
 const after=shell.toggleSidebar();
 assert.equal(after.sidebarOpen,false);
 assert.equal(after.view,ShellView.ALBUMS);
 assert.equal(after.libraryMode,"albums");
});

test("album and artist detail remain nested shell locations",()=>{
 const shell=createShellNavigation();

 const album=shell.openLibraryDetail("album","road");
 assert.equal(album.view,ShellView.ALBUMS);
 assert.equal(album.panel,"library");
 assert.equal(album.libraryMode,"albums");
 assert.deepEqual(album.libraryDetail,{kind:"album",key:"road"});

 const albumOverview=shell.select(ShellView.ALBUMS);
 assert.equal(albumOverview.libraryDetail,null);

 const artist=shell.openLibraryDetail("artist","alpha");
 assert.equal(artist.view,ShellView.ARTISTS);
 assert.deepEqual(artist.libraryDetail,{kind:"artist",key:"alpha"});

 const closed=shell.closeLibraryDetail();
 assert.equal(closed.view,ShellView.ARTISTS);
 assert.equal(closed.libraryDetail,null);

 assert.throws(
  ()=>shell.openLibraryDetail("genre","ambient"),
  /album or artist/
 );
 assert.throws(
  ()=>shell.openLibraryDetail("album",""),
  /detail key/
 );
});

test("unknown shell terrain fails instead of silently inventing product navigation",()=>{
 const shell=createShellNavigation();
 assert.throws(()=>shell.select("settings"),/Unknown VexStream shell view/);
});
