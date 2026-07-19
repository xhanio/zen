<script setup lang="ts">
import { onMounted, ref, watch } from 'vue';
import { RouterLink } from 'vue-router';
import { storeToRefs } from 'pinia';
import { useConversationsStore } from '../stores/conversations';
import { useChatSidebar } from '../composables/useChatSidebar';
import { useTilePrefsStore } from '../stores/tilePrefs';
import ChatThread from '../components/chat/ChatThread.vue';
import ChatComposer from '../components/chat/ChatComposer.vue';
import ResizableSplitter from '../components/ResizableSplitter.vue';
import { BackendError } from '../types/api';
import type { Conversation } from '../types/entity';

function anchorHref(kind: string, id: string): string {
  if (kind === 'card') return `/c/${id}`;
  if (kind === 'group') return `/g/${id}`;
  return '#';
}

const store = useConversationsStore();
const sidebar = useChatSidebar();
const tilePrefs = useTilePrefsStore();
const { list, listLoading, activeID } = storeToRefs(store);
const { chatListWidth } = storeToRefs(tilePrefs);

const errorMsg = ref<string | null>(null);

// The anchor pill shows the source's title (card title / group name), not its
// raw id. Titles resolve async via the store's cached resolver; until one lands
// (or for a document, which the resolver doesn't title) the label falls back to
// "<kind> · <id6>". Resolve for every anchored conversation whenever the list
// changes, deduped by the store's cache.
const anchorTitles = ref<Record<string, string>>({});
async function loadAnchorTitles() {
  for (const c of list.value) {
    if (!c.anchor_kind || !c.anchor_id) continue;
    const key = `${c.anchor_kind}:${c.anchor_id}`;
    if (anchorTitles.value[key]) continue;
    const title = await store.resolveAnchorTitle(c.anchor_kind, c.anchor_id);
    if (title) anchorTitles.value = { ...anchorTitles.value, [key]: title };
  }
}
function anchorLabel(c: Conversation): string {
  if (!c.anchor_kind || !c.anchor_id) return '';
  return anchorTitles.value[`${c.anchor_kind}:${c.anchor_id}`]
    ?? `${c.anchor_kind} · ${c.anchor_id.slice(-6)}`;
}
// The source card's genesis, shown as a tooltip on the pill — resolved into the
// store cache alongside the title.
function genesisFor(c: Conversation): string | null {
  return store.anchorGenesis(c.anchor_kind, c.anchor_id);
}
watch(list, loadAnchorTitles);

async function doDelete(id: string) {
  if (!window.confirm('Delete this conversation? This cannot be undone.')) return;
  try {
    await store.deleteOne(id);
  } catch (e) {
    errorMsg.value = e instanceof BackendError ? e.message : String(e);
  }
}

onMounted(async () => {
  await store.loadList();
  await loadAnchorTitles();
});

async function open(id: string) {
  await store.setActive(id);
}

async function newChat() {
  errorMsg.value = null;
  try {
    // Clear sidebar anchor state so ChatComposer doesn't re-create with stale anchor.
    await sidebar.openFor(null, null);
    const conv = await store.create({ title: '', anchor_kind: null, anchor_id: null });
    await store.setActive(conv.id);
    // /chat hosts the thread inline; the slide-over sidebar shouldn't double-show.
    sidebar.close();
  } catch (e) {
    errorMsg.value = e instanceof BackendError ? e.message : String(e);
  }
}
</script>

<template>
  <div class="flex h-full gap-4">
    <div
      data-test="chat-list-col"
      class="flex shrink-0 flex-col border-r border-border pr-2 min-h-0"
      :style="{ width: chatListWidth + 'px' }"
    >
      <button
        type="button"
        class="mb-2 w-full shrink-0 rounded border border-accent-border px-2 py-1 text-xs text-accent-fg hover:bg-accent-bg"
        @click="newChat"
      >+ New chat</button>
      <div v-if="errorMsg" class="mb-2 shrink-0 text-xs text-destructive-fg">{{ errorMsg }}</div>
      <div v-if="listLoading" class="text-xs text-muted-fg">Loading…</div>
      <ul v-else class="min-h-0 flex-1 space-y-1 overflow-y-auto text-xs">
        <li v-for="c in list" :key="c.id" class="space-y-0.5">
          <div class="group flex items-center gap-1">
            <button
              type="button"
              class="min-w-0 flex-1 truncate rounded px-2 py-1 text-left hover:bg-muted"
              :class="activeID === c.id ? 'bg-muted font-medium' : ''"
              @click="open(c.id)"
            >
              {{ c.title || '(untitled)' }}
            </button>
            <button
              type="button"
              data-test="conv-delete"
              class="shrink-0 rounded px-1 text-muted-fg opacity-0 transition-opacity hover:text-destructive-fg group-hover:opacity-100"
              title="Delete conversation"
              @click.stop="doDelete(c.id)"
            >✕</button>
          </div>
          <RouterLink
            v-if="c.anchor_kind && c.anchor_id"
            :to="anchorHref(c.anchor_kind, c.anchor_id)"
            class="ml-2 inline-block rounded bg-accent-bg px-1.5 py-0.5 text-[10px] text-accent-fg hover:bg-accent-bg"
            :title="genesisFor(c) || undefined"
            @click.stop
          >{{ anchorLabel(c) }}</RouterLink>
        </li>
        <li v-if="list.length === 0" class="px-2 py-1 text-muted-fg">No conversations yet.</li>
      </ul>
    </div>
    <ResizableSplitter
      data-test="chat-list-splitter"
      :width="chatListWidth"
      :min="180"
      :max="480"
      :default-width="256"
      side="right"
      aria-label="Resize conversation list"
      class="w-1.5 shrink-0 self-stretch rounded"
      @update:width="tilePrefs.setChatListWidth"
    />
    <div class="flex min-h-0 flex-1 flex-col">
      <ChatThread :conversation-id="activeID" />
      <ChatComposer />
    </div>
  </div>
</template>
