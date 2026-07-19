import { defineStore } from 'pinia';

// Container reading-view collapse: which section CARDS the reader has folded
// to a title-only row. Cleared when navigating to another card so it doesn't
// leak across container views. Not persisted — folding is a "while I'm reading
// this doc" gesture, not a preference.
//
// Two things write here, both keyed by card id:
//   - the per-section title click (toggleCard) folds one section;
//   - the legend's bulk "collapse this whole level" (setCollapsed over the
//     level's card ids) folds/unfolds every section of a level at once.
// A section is collapsed iff its id is in the set: it renders title-only and
// carries no grade (the grade gutter reads the expanded list).

interface ContainerFilterState {
  collapsedIds: Set<string>;
}

export const useContainerFilterStore = defineStore('containerFilter', {
  state: (): ContainerFilterState => ({ collapsedIds: new Set() }),
  getters: {
    isCollapsed:
      (state) =>
      (cardId: string): boolean =>
        state.collapsedIds.has(cardId),
  },
  actions: {
    toggleCard(cardId: string) {
      const next = new Set(this.collapsedIds);
      if (next.has(cardId)) next.delete(cardId);
      else next.add(cardId);
      this.collapsedIds = next;
    },
    setCollapsed(cardIds: string[], collapsed: boolean) {
      const next = new Set(this.collapsedIds);
      for (const id of cardIds) {
        if (collapsed) next.add(id);
        else next.delete(id);
      }
      this.collapsedIds = next;
    },
    clear() {
      this.collapsedIds = new Set();
    },
  },
});
