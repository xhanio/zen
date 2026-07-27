import { ref } from 'vue';
import { useConversationsStore } from '../stores/conversations';

// Module-level singleton state: only one sidebar per browser tab (spec §6.4).
const open = ref(false);
const anchorKind = ref<string | null>(null);
const anchorID = ref<string | null>(null);
const pendingSelection = ref<string | null>(null);
// The rendered-text range of pendingSelection, captured at the drag. Only the
// SPA can know it, so it travels with the message and nowhere else.
const pendingRange = ref<{ start: number; end: number } | null>(null);
const pendingSeq = ref<number | null>(null);

// Module-level singleton actions: same function identity per call so test
// spies (vi.spyOn(useChatSidebar(), 'openForConversation')) intercept the
// same binding the consuming components see.
export const actions = {
  async openFor(
    kind: string | null,
    id: string | null,
    selectionText: string | null = null,
    range: { start: number; end: number } | null = null,
    seq: number | null = null,
  ) {
    const store = useConversationsStore();
    anchorKind.value = kind;
    anchorID.value = id;
    pendingSelection.value = selectionText;
    pendingRange.value = range;
    pendingSeq.value = seq;
    open.value = true;
    if (kind && id) {
      // Fetch the anchored conversation directly — NOT via the shared list,
      // whose sequence guard a concurrent ChatHeader load can trip, leaving us
      // stuck on an empty "New conversation".
      const existing = await store.mostRecentForAnchor(kind, id);
      if (existing) {
        await store.setActive(existing.id);
        return;
      }
    }
    await store.setActive(null);
  },
  async openForConversation(conversationID: string) {
    const store = useConversationsStore();
    anchorKind.value = null;
    anchorID.value = null;
    pendingSelection.value = null;
    pendingRange.value = null;
    pendingSeq.value = null;
    open.value = true;
    await store.setActive(conversationID);
  },
  close() {
    open.value = false;
    pendingSelection.value = null;
    pendingRange.value = null;
    pendingSeq.value = null;
  },
  clearSelection() {
    pendingSelection.value = null;
    pendingRange.value = null;
    pendingSeq.value = null;
  },
};

export function useChatSidebar() {
  return {
    open,
    anchorKind,
    anchorID,
    pendingSelection,
    pendingRange,
    pendingSeq,
    openFor: actions.openFor,
    openForConversation: actions.openForConversation,
    close: actions.close,
    clearSelection: actions.clearSelection,
  };
}
