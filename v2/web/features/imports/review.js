// [VXG RealForever]
// Feature-local presentation/planning for one reviewed Import source.
//
// Provider inspection and duplicate evidence are inputs. This module does not inspect providers,
// choose Library membership, run acquisition, or own durable job state.

export function formImportReview(
  reviewed,
  destinations,
) {
  const inspection = reviewed?.inspection;
  if (!inspection || typeof inspection !== "object") {
    throw new Error("Import review requires inspection");
  }

  const title = stringValue(inspection.title) || "Untitled source";
  const sections = Object.freeze(
    Array.isArray(inspection.sections)
      ? inspection.sections.map((section, index) =>
          Object.freeze({
            index: Number(section?.index ?? index + 1),
            sourceTitle:
              stringValue(section?.title) ||
              `Track ${index + 1}`,
            title:
              stringValue(section?.title) ||
              `Track ${index + 1}`,
            artist: stringValue(section?.artist),
            start: finiteNonNegative(section?.start),
            end: finiteNonNegative(section?.end),
            selected: section?.selected !== false,
          }),
        )
      : [],
  );
  const splitAvailable =
    inspection.hasSections === true &&
    sections.length >= 2;

  const duplicateItems = Object.freeze(
    Array.isArray(reviewed?.items)
      ? reviewed.items.map(item =>
          Object.freeze({
            ...item,
            matches: Object.freeze(
              Array.isArray(item?.matches)
                ? item.matches.map(match =>
                    Object.freeze({ ...match }),
                  )
                : [],
            ),
          }),
        )
      : [],
  );

  return Object.freeze({
    url: stringValue(inspection.url),
    title,
    creator:
      stringValue(inspection.channel) ||
      stringValue(inspection.artist),
    durationSeconds: finiteNonNegative(inspection.duration),
    thumbnail: stringValue(inspection.thumbnail),
    destinations: Object.freeze(
      uniqueStrings(destinations),
    ),
    defaultMode: splitAvailable ? "split" : "whole",
    splitAvailable,
    collection:
      stringValue(inspection.album) ||
      title,
    artistOverride: stringValue(inspection.artist),
    genre:
      stringValue(inspection.genre) ||
      firstString(inspection.categories),
    sections,
    duplicateItems,
    duplicateItemCount: duplicateItems.length,
    advisory: reviewed?.advisory === true,
  });
}

export function buildImportPlan(
  review,
  form,
) {
  if (!review || typeof review !== "object") {
    throw new Error("Import plan requires review");
  }
  const destination = stringValue(form?.destination);
  if (!destination) {
    throw new Error("Choose a VexStream music folder");
  }
  if (!review.destinations.includes(destination)) {
    throw new Error("Import destination is no longer admitted");
  }

  let mode = stringValue(form?.mode) || review.defaultMode;
  if (!review.splitAvailable) {
    mode = "whole";
  }
  if (!["whole", "split", "both"].includes(mode)) {
    throw new Error("Import mode is unsupported");
  }

  const edits = Array.isArray(form?.sections)
    ? form.sections
    : [];
  const byIndex = new Map(
    edits.map(value => [
      Number(value?.index),
      value,
    ]),
  );
  const sections = review.sections.map(section => {
    const edit = byIndex.get(section.index);
    return Object.freeze({
      index: section.index,
      sourceTitle: section.sourceTitle,
      title:
        stringValue(edit?.title) ||
        section.title,
      artist:
        stringValue(edit?.artist) ||
        section.artist,
      start: section.start,
      end: section.end,
      selected:
        edit?.selected === undefined
          ? section.selected
          : edit.selected !== false,
    });
  });

  if (
    (mode === "split" || mode === "both") &&
    !sections.some(section => section.selected)
  ) {
    throw new Error("Select at least one track section");
  }

  return Object.freeze({
    sourceTitle: review.title,
    sourceCreator: review.creator,
    thumbnail: review.thumbnail,
    sourceDuration: review.durationSeconds,
    destination,
    url: review.url,
    mode,
    wholeTitle: review.title,
    artistOverride: stringValue(form?.artistOverride),
    collection:
      stringValue(form?.collection) ||
      review.collection,
    genre:
      stringValue(form?.genre) ||
      review.genre,
    sections: Object.freeze(sections),
  });
}

function uniqueStrings(values) {
  const seen = new Set();
  const out = [];
  for (const value of Array.isArray(values) ? values : []) {
    const text = stringValue(value);
    if (!text || seen.has(text)) continue;
    seen.add(text);
    out.push(text);
  }
  return out;
}

function firstString(values) {
  if (!Array.isArray(values)) return "";
  for (const value of values) {
    const text = stringValue(value);
    if (text) return text;
  }
  return "";
}

function stringValue(value) {
  return typeof value === "string"
    ? value.trim()
    : "";
}

function finiteNonNegative(value) {
  const number = Number(value);
  return Number.isFinite(number) && number > 0
    ? number
    : 0;
}
