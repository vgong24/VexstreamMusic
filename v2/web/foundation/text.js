// [VXG RealForever]
//
// Text identity is separate from rendered language. A locale used by evidence tooling does not
// automatically become a product-supported language.

export function resolveText(catalog, textRef, values = {}) {
  const template = catalog?.[textRef];
  if (typeof template !== "string") {
    throw new Error(`missing textRef: ${textRef}`);
  }

  return template.replace(/\{([A-Za-z0-9_.-]+)\}/g, (match, key) =>
    Object.prototype.hasOwnProperty.call(values, key)
      ? String(values[key])
      : match,
  );
}
