import { defineStore } from 'pinia';

// Viewer-side preference for the left rail's width, persisted in localStorage
// so a dragged width survives refreshes. Nothing here is server-authoritative.

const KEY = 'zen:railWidth';
export const RAIL_DEFAULT_WIDTH = 256;
export const RAIL_MIN_WIDTH = 200;
export const RAIL_MAX_WIDTH = 480;

function clamp(w: number): number {
  return Math.min(RAIL_MAX_WIDTH, Math.max(RAIL_MIN_WIDTH, Math.round(w)));
}
function load(): number {
  if (typeof localStorage === 'undefined') return RAIL_DEFAULT_WIDTH;
  try {
    const raw = localStorage.getItem(KEY);
    if (!raw) return RAIL_DEFAULT_WIDTH;
    const n = Number(raw);
    return Number.isFinite(n) ? clamp(n) : RAIL_DEFAULT_WIDTH;
  } catch {
    return RAIL_DEFAULT_WIDTH;
  }
}
function persist(w: number) {
  if (typeof localStorage === 'undefined') return;
  try {
    localStorage.setItem(KEY, String(w));
  } catch {
    // ignore quota / disabled-storage errors — the width just won't survive
  }
}

export const useRailPrefsStore = defineStore('railPrefs', {
  state: () => ({ width: load() }),
  actions: {
    setWidth(w: number) {
      this.width = clamp(w);
      persist(this.width);
    },
    reset() {
      this.width = RAIL_DEFAULT_WIDTH;
      persist(this.width);
    },
  },
});
