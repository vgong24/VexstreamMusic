import assert from "node:assert/strict";
import test from "node:test";

import { canonicalGoCheckoutText } from "./verify-local-support.mjs";

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
