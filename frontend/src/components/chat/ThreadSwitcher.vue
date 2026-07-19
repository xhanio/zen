<script setup lang="ts">
import { computed, onMounted } from 'vue';
import { storeToRefs } from 'pinia';
import { useConversationsStore } from '../../stores/conversations';
import { useChatSidebar } from '../../composables/useChatSidebar';

const store = useConversationsStore();
const sidebar = useChatSidebar();
const { list, activeID } = storeToRefs(store);

const threads = computed(() =>
  list.value.filter(
    (c) => c.anchor_kind === sidebar.anchorKind.value && c.anchor_id === sidebar.anchorID.value,
  ),
);

onMounted(() => {
  // Load THIS anchor's threads. Not pending:true — that endpoint
  // (ListPendingConversations) returns the GLOBAL pending set and ignores the
  // anchor, so it would list the wrong conversations.
  store.loadList({
    anchorKind: sidebar.anchorKind.value ?? undefined,
    anchorID: sidebar.anchorID.value ?? undefined,
  });
});

async function open(id: string) {
  await store.setActive(id);
}
async function newThread() {
  const conv = await store.create({
    title: '',
    anchor_kind: sidebar.anchorKind.value,
    anchor_id: sidebar.anchorID.value,
  });
  await store.setActive(conv.id);
}
</script>

<template>
  <div class="rounded-lg border border-border bg-paper p-1 shadow-lg">
    <button
      v-for="t in threads"
      :key="t.id"
      type="button"
      data-test="thread-row"
      class="flex w-full items-center gap-2 rounded px-2 py-1.5 text-left text-xs hover:bg-muted"
      :class="t.id === activeID ? 'bg-accent-bg' : ''"
      @click="open(t.id)"
    >
      <span class="min-w-0 flex-1 truncate">{{ t.title || 'Untitled thread' }}</span>
    </button>
    <button
      type="button"
      data-test="thread-new"
      class="mt-1 w-full border-t border-border px-2 py-1.5 text-left text-xs text-accent-fg hover:bg-muted"
      @click="newThread"
    >+ New thread on this card</button>
  </div>
</template>
