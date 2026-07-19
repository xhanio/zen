import { defineStore } from 'pinia';

// Multi-select tag filter that scopes what's visible on the current
// group view. Clicking a tag in the sidebar toggles it. Semantics are
// intersection (AND): a card must carry every selected tag to remain
// visible. An empty set means "show everything".
//
// Kept as ephemeral session state — the filter is a browsing move, not
// a stored preference. Refresh returns to "show everything".

interface TagFilterState {
  activeTags: string[];
}

export const useTagFilterStore = defineStore('tagFilter', {
  state: (): TagFilterState => ({ activeTags: [] }),
  getters: {
    isActive: (state) => (name: string) => state.activeTags.includes(name),
  },
  actions: {
    toggle(name: string) {
      const i = this.activeTags.indexOf(name);
      if (i === -1) this.activeTags = [...this.activeTags, name];
      else this.activeTags = this.activeTags.filter((n) => n !== name);
    },
    remove(name: string) {
      this.activeTags = this.activeTags.filter((n) => n !== name);
    },
    clear() {
      this.activeTags = [];
    },
  },
});
