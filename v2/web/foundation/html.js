// [VXG RealForever]
// Tiny browser-presentation primitive: dynamic product data must remain text, never authored markup.

const ESCAPE = Object.freeze({
  "&": "&amp;",
  "<": "&lt;",
  ">": "&gt;",
  '"': "&quot;",
  "'": "&#39;",
});

export function escapeHTML(value) {
  return String(value ?? "").replace(/[&<>"']/g, character => ESCAPE[character]);
}
