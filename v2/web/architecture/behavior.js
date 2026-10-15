// [VXG RealForever]
//
// Small browser-native semantic kernel earned by the first real V2 consumer: Playback.
//
// This is intentionally not a scheduler, state store, event bus, provider adapter, or UI framework.
// It only protects family/role kinship and deterministic State + Act -> Effect resolution.

const ROLES = new Set(["control", "act", "effect"]);

export function semanticToken(familyRef, role, ref) {
  if (!familyRef || !ref || !ROLES.has(role)) {
    throw new Error("invalid semantic token");
  }
  return Object.freeze({ familyRef, role, ref });
}

export function stateOf(familyRef, kind, fields = {}) {
  if (!familyRef || !kind) {
    throw new Error("state requires familyRef and kind");
  }
  return Object.freeze({ familyRef, kind, ...fields });
}

export function defineBehaviorFamily({ ref, relationships }) {
  if (!ref) {
    throw new Error("behavior family requires ref");
  }

  const rows = relationships.map((row, index) => {
    if (!row?.stateKind || !row?.act || !row?.effect) {
      throw new Error(`${ref}: relationship ${index} is incomplete`);
    }
    for (const [role, token] of [["act", row.act], ["effect", row.effect]]) {
      if (token.familyRef !== ref || token.role !== role) {
        throw new Error(`${ref}: ${role} belongs to another family or role`);
      }
    }
    return Object.freeze({
      stateKind: row.stateKind,
      act: row.act,
      effect: row.effect,
      admittedStateKind: row.admittedStateKind ?? row.stateKind,
      when: row.when ?? (() => true),
    });
  });

  function resolve(currentState, act) {
    if (currentState?.familyRef !== ref) {
      throw new Error(`${ref}: State belongs to another family`);
    }
    if (act?.familyRef !== ref || act?.role !== "act") {
      throw new Error(`${ref}: Act belongs to another family or role`);
    }

    const matches = rows.filter(
      row =>
        row.stateKind === currentState.kind &&
        row.act === act &&
        row.when(currentState),
    );

    if (matches.length > 1) {
      throw new Error(
        `${ref}: ambiguous relationship for ${currentState.kind} + ${act.ref}`,
      );
    }

    return matches[0] ?? null;
  }

  return Object.freeze({
    ref,
    relationships: Object.freeze(rows),
    resolve,
  });
}
