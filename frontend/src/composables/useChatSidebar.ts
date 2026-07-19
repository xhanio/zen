import { ref } from 'vue';
import { useConversationsStore } from '../stores/conversations';

// Module-level singleton state: only one sidebar per browser tab (spec §6.4).
const open = ref(false);
const anchorKind = ref<string | null>(null);
const anchorID = ref<string | null>(null);
const pendingSelection = ref<string | null>(null);

// Module-level singleton actions: same function identity per call so test
// spies (vi.spyOn(useChatSidebar(), 'openForConversation')) intercept the
// same binding the consuming components see.
export const actions = {
  async openFor(kind: string | null, id: string | null, selectionText: string | null = null) {
    const store = useConversationsStore();
    anchorKind.value = kind;
    anchorID.value = id;
    pendingSelection.value = selectionText;
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
    open.value = true;
    await store.setActive(conversationID);
  },
  close() {
    open.value = false;
    pendingSelection.value = null;
  },
  clearSelection() {
    pendingSelection.value = null;
  },
};

export function useChatSidebar() {
  return {
    open,
    anchorKind,
    anchorID,
    pendingSelection,
    openFor: actions.openFor,
    openForConversation: actions.openForConversation,
    close: actions.close,
    clearSelection: actions.clearSelection,
  };
}
