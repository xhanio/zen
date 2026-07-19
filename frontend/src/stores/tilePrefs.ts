import { defineStore } from 'pinia';

// Viewer-side preferences for how card tiles render. Persisted in
// localStorage so the choice survives refreshes; nothing here is
// server-authoritative. Add fields as more toggles land.

const KEY = 'zen:tilePrefs';

export const DOCUMENTS_DEFAULT_WIDTH = 240;
export const DOCUMENTS_MIN_WIDTH = 160;
export const DOCUMENTS_MAX_WIDTH = 560;

export function clampDocumentsWidth(w: number): number {
  return Math.min(DOCUMENTS_MAX_WIDTH, Math.max(DOCUMENTS_MIN_WIDTH, Math.round(w)));
}

export const CHAT_LIST_DEFAULT_WIDTH = 256;
export const CHAT_LIST_MIN_WIDTH = 180;
export const CHAT_LIST_MAX_WIDTH = 480;

export function clampChatListWidth(w: number): number {
  return Math.min(CHAT_LIST_MAX_WIDTH, Math.max(CHAT_LIST_MIN_WIDTH, Math.round(w)));
}

interface TilePrefsState {
  hideSummaries: boolean;
  showTrashedSections: boolean;
  // Group grid: hide any card whose parent_card_id is set — i.e. sections
  // that live inside a container card. They're already visible when you
  // open the container, and re-listing them alongside their parent
  // clutters the grid. Defaults to true.
  hideSections: boolean;
  // Group grid: the width (px) of the Documents column, dragged via the
  // ResizableSplitter between it and the level board. Global across groups.
  documentsWidth: number;
  // /chat: the width (px) of the conversation-list column, dragged via the
  // ResizableSplitter between it and the thread.
  chatListWidth: number;
}

function fresh(): TilePrefsState {
  return {
    hideSummaries: false,
    showTrashedSections: false,
    hideSections: true,
    documentsWidth: DOCUMENTS_DEFAULT_WIDTH,
    chatListWidth: CHAT_LIST_DEFAULT_WIDTH,
  };
}

function load(): TilePrefsState {
  if (typeof localStorage === 'undefined') return fresh();
  try {
    const raw = localStorage.getItem(KEY);
    if (!raw) return fresh();
    const parsed = JSON.parse(raw) as Partial<TilePrefsState>;
    return {
      hideSummaries: !!parsed.hideSummaries,
      showTrashedSections: !!parsed.showTrashedSections,
      hideSections: parsed.hideSections === undefined ? true : !!parsed.hideSections,
      documentsWidth: Number.isFinite(Number(parsed.documentsWidth))
        ? clampDocumentsWidth(Number(parsed.documentsWidth))
        : DOCUMENTS_DEFAULT_WIDTH,
      chatListWidth: Number.isFinite(Number(parsed.chatListWidth))
        ? clampChatListWidth(Number(parsed.chatListWidth))
        : CHAT_LIST_DEFAULT_WIDTH,
    };
  } catch {
    return fresh();
  }
}

function persist(state: TilePrefsState) {
  if (typeof localStorage === 'undefined') return;
  try {
    localStorage.setItem(KEY, JSON.stringify(state));
  } catch {
    // ignore quota / disabled-storage errors — the toggle just won't survive
  }
}

export const useTilePrefsStore = defineStore('tilePrefs', {
  state: (): TilePrefsState => load(),
  actions: {
    toggleSummaries() {
      this.hideSummaries = !this.hideSummaries;
      persist(this.$state);
    },
    toggleShowTrashedSections() {
      this.showTrashedSections = !this.showTrashedSections;
      persist(this.$state);
    },
    toggleSections() {
      this.hideSections = !this.hideSections;
      persist(this.$state);
    },
    setDocumentsWidth(px: number) {
      this.documentsWidth = clampDocumentsWidth(px);
      persist(this.$state);
    },
    resetDocumentsWidth() {
      this.documentsWidth = DOCUMENTS_DEFAULT_WIDTH;
      persist(this.$state);
    },
    setChatListWidth(px: number) {
      this.chatListWidth = clampChatListWidth(px);
      persist(this.$state);
    },
    resetChatListWidth() {
      this.chatListWidth = CHAT_LIST_DEFAULT_WIDTH;
      persist(this.$state);
    },
  },
});
