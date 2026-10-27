import assert from "node:assert/strict";
import test from "node:test";

import {
 canonicalGoCheckoutText,
 physicalLocalCommand,
} from "./verify-local-support.mjs";

test("Go checkout normalization removes only CRLF transport representation",()=>{
 assert.equal(
  canonicalGoCheckoutText("package x\r\n\r\nfunc X() {}\r\n"),
  "package x\n\nfunc X() {}\n"
 );
 assert.equal(
  canonicalGoCheckoutText("package x\n\nfunc X() {}\n"),
  "package x\n\nfunc X() {}\n"
 );
});

test("bare carriage returns are not silently rewritten",()=>{
 assert.equal(canonicalGoCheckoutText("a\rb"),"a\rb");
});

test("Windows cmd and bat tools are realized through ComSpec",()=>{
 assert.deepEqual(
  physicalLocalCommand("npm.cmd",["test"],{
   platform:"win32",
   comspec:"C:\\Windows\\System32\\cmd.exe"
  }),
  {
   command:"C:\\Windows\\System32\\cmd.exe",
   args:["/d","/s","/c","npm.cmd test"]
  }
 );
 assert.deepEqual(
  physicalLocalCommand("go",["test","./v2/..."],{
   platform:"win32",
   comspec:"C:\\Windows\\System32\\cmd.exe"
  }),
  {
   command:"go",
   args:["test","./v2/..."]
  }
 );
});

test("Windows command realization quotes ordinary whitespace without shell widening",()=>{
 assert.deepEqual(
  physicalLocalCommand("npm.cmd",["run","test lived"],{
   platform:"win32",
   comspec:"cmd.exe"
  }),
  {
   command:"cmd.exe",
   args:["/d","/s","/c",'npm.cmd run "test lived"']
  }
 );
 assert.throws(
  ()=>physicalLocalCommand("npm.cmd",["test & echo no"],{
   platform:"win32",
   comspec:"cmd.exe"
  }),
  /unsafe Windows command token/
 );
});
