import { defineStore } from 'pinia';
import { ref } from 'vue';
import type { LevelEntry } from '../types/entity';

// Sidebar level filter: per-group, which catalog entries are checked
// (visible) in the level board. Keyed by entry id so two entries at
// the same weight are independently toggleable — matching how legend
// filter, drop targets, and every other UI surface treat catalog
// entries in v0.10.
//
// `showMisc` covers the Unfiled sentinel; it's independent of the
// per-entry toggles.

interface LevelFilterState {
  selectedEntryIds: string[];
  showMisc: boolean;
}

function sameIds(a: string[], b: string[]): boolean {
  if (a === b) return true;
  if (a.length !== b.length) return false;
  for (let i = 0; i < a.length; i++) {
    if (a[i] !== b[i]) return false;
  }
  return true;
}

export const useLevelFilterStore = defineStore('levelFilter', () => {
  const byGroup = ref<Record<string, LevelFilterState>>({});
  const seenCatalog = ref<Record<string, string[]>>({});

  // ensure(groupId, catalog) seeds the group's filter state on first
  // access with every entry checked + Unfiled shown. When the catalog
  // grows later (new entries appended), the added ids are also checked
  // by default so the new columns don't spring up hidden. Deletions on
  // the catalog side propagate naturally because the group.update
  // cascade removes those ids everywhere.
  function ensure(groupId: string, catalog: LevelEntry[]): LevelFilterState {
    const ids = catalog.map((e) => e.id);
    const existing = byGroup.value[groupId];
    const seen = seenCatalog.value[groupId];

    if (existing && seen && sameIds(seen, ids)) {
      return existing;
    }

    if (!existing) {
      byGroup.value[groupId] = { selectedEntryIds: ids.slice(), showMisc: true };
    } else {
      const seenSet = new Set(seen ?? []);
      const added = ids.filter((id) => !seenSet.has(id));
      if (added.length > 0) {
        existing.selectedEntryIds = [...existing.selectedEntryIds, ...added];
      }
    }
    seenCatalog.value[groupId] = ids.slice();
    return byGroup.value[groupId];
  }

  function setSelectedEntryIds(groupId: string, next: string[]) {
    const s = byGroup.value[groupId];
    if (s) s.selectedEntryIds = next;
  }

  function setShowMisc(groupId: string, next: boolean) {
    const s = byGroup.value[groupId];
    if (s) s.showMisc = next;
  }

  return {
    byGroup,
    ensure,
    setSelectedEntryIds,
    setShowMisc,
  };
});
