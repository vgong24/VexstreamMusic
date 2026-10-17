// [VXG RealForever]
// V3-recognizable presentation for the Desktop Phone Connection walk.
//
// This module projects Phone Connection state into the shared V3 modal shell.
// It does not own navigation, network requests, TLS, discovery, or pairing custody.

import { escapeHTML } from "../../foundation/html.js";
import { PhoneConnectionAction } from "./connection.js";

export function presentPhoneConnection(projection) {
  if (!projection?.open) return null;

  const title = "Connect your VexLife phone";
  const actionsHTML = '<button type="button" data-modal-close>Close</button>';

  if (projection.phase === "loading") {
    return freezeView(
      title,
      '<div id="phonePairingPanel"><div class="small muted">Checking phone connection…</div></div>',
      actionsHTML,
    );
  }

  if (projection.phase === "failed") {
    return freezeView(
      title,
      '<div id="phonePairingPanel"><div class="notice">' +
        escapeHTML(projection.error || "Phone connection unavailable") +
        "</div></div>",
      actionsHTML,
    );
  }

  const state = projection.pairing;
  if (projection.phase !== "ready" || !state) {
    return freezeView(
      title,
      '<div id="phonePairingPanel"><div class="notice">Phone connection state is unavailable.</div></div>',
      actionsHTML,
    );
  }

  return freezeView(
    title,
    '<div id="phonePairingPanel">' +
      pairingBlock(state) +
      pendingBlock(state.pending) +
      pairedBlock(state.paired) +
      '<div class="small muted" style="margin-top:14px">This nearby pairing is a local VexStream connection candidate. It does not expose imports, delete, diagnostics, repair, or other desktop controls to the phone.</div>' +
      "</div>",
    actionsHTML,
  );
}

function pairingBlock(state) {
  if (state.pairingWindowOpen) {
    return (
      '<div class="notice success">' +
      "<strong>Ready to pair for 5 minutes</strong>" +
      '<div class="small muted" style="margin-top:5px">On your phone open VexLife → VexStream Music → Find VexStream. Choose <strong>' +
      escapeHTML(state.displayName || "this VexStream") +
      "</strong>.</div>" +
      '<div style="font-size:30px;font-weight:800;letter-spacing:.12em;margin-top:12px">' +
      escapeHTML(formatPairingCode(state.pairingCode)) +
      "</div>" +
      '<div class="small muted">The phone should show this same code before it asks for approval.</div>' +
      "</div>"
    );
  }

  const label = state.paired.length ? "Connect another phone" : "Connect a phone";
  return (
    '<div class="notice">' +
    "<strong>" + escapeHTML(label) + "</strong>" +
    '<div class="small muted" style="margin-top:5px">Make this VexStream visible to VexLife on the same Wi-Fi for five minutes.</div>' +
    '<button class="primary" style="margin-top:10px" data-phone-action="' +
    PhoneConnectionAction.StartPairing +
    '">' +
    escapeHTML(label) +
    "</button>" +
    "</div>"
  );
}

function pendingBlock(pending) {
  if (!pending.length) return "";
  return (
    '<div style="margin-top:14px"><h3>Waiting for your approval</h3>' +
    pending
      .map(
        item =>
          '<div class="issue info"><div class="row between"><div><strong>' +
          escapeHTML(item.displayName || "VexLife phone") +
          '</strong><div class="small muted">Confirm the matching code on the phone, then allow it here.</div></div><div class="row">' +
          '<button data-phone-action="' +
          PhoneConnectionAction.Deny +
          '" data-phone-request-ref="' +
          escapeHTML(item.requestRef) +
          '">Deny</button>' +
          '<button class="primary" data-phone-action="' +
          PhoneConnectionAction.Approve +
          '" data-phone-request-ref="' +
          escapeHTML(item.requestRef) +
          '">Allow</button>' +
          "</div></div></div>",
      )
      .join("") +
    "</div>"
  );
}

function pairedBlock(paired) {
  if (!paired.length) return "";
  return (
    '<div style="margin-top:14px"><h3>Remembered phones</h3>' +
    paired
      .map(
        item =>
          '<div class="lifecycle-row"><div><strong>' +
          escapeHTML(item.displayName || "VexLife phone") +
          '</strong><div class="small muted">Approved ' +
          escapeHTML(item.approvedAt || "") +
          '</div></div><button data-phone-action="' +
          PhoneConnectionAction.Forget +
          '" data-phone-device-ref="' +
          escapeHTML(item.deviceRef) +
          '">Forget</button></div>',
      )
      .join("") +
    "</div>"
  );
}

function formatPairingCode(code) {
  const value = String(code ?? "").padStart(6, "0");
  return value.slice(0, 3) + " " + value.slice(3);
}

function freezeView(title, bodyHTML, actionsHTML) {
  return Object.freeze({ title, bodyHTML, actionsHTML });
}
