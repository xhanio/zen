<script setup lang="ts">
import { onBeforeUnmount, ref, watch } from 'vue';
import { useConversationsStore } from '../../stores/conversations';
import { useChatSidebar } from '../../composables/useChatSidebar';
import { BackendError } from '../../types/api';
import SessionPresencePill from './SessionPresencePill.vue';
import SessionSwitcher from './SessionSwitcher.vue';

const store = useConversationsStore();
const sidebar = useChatSidebar();

const content = ref('');
const sending = ref(false);
const errorMsg = ref<string | null>(null);

// The session picker lives here, at the send decision, because one thread can
// target different sessions. The switcher opens UPWARD — the composer sits at
// the panel's bottom edge. Close it on an outside click, like the header menus.
const sessionOpen = ref(false);
const pickerEl = ref<HTMLElement | null>(null);
function onDocClick(e: MouseEvent) {
  if (pickerEl.value && !pickerEl.value.contains(e.target as Node)) sessionOpen.value = false;
}
watch(sessionOpen, (open) => {
  if (open) document.addEventListener('click', onDocClick, true);
  else document.removeEventListener('click', onDocClick, true);
});
onBeforeUnmount(() => document.removeEventListener('click', onDocClick, true));

async function send() {
  const text = content.value.trim();
  if (!text || sending.value) return;
  sending.value = true;
  errorMsg.value = null;
  try {
    if (!store.activeID) {
      const conv = await store.create({
        title: '',
        anchor_kind: sidebar.anchorKind.value,
        anchor_id: sidebar.anchorID.value,
      });
      await store.setActive(conv.id);
    }
    await store.optimisticPost(text, sidebar.pendingSelection.value);
    content.value = '';
    sidebar.clearSelection();
  } catch (e) {
    errorMsg.value = e instanceof BackendError ? e.message : String(e);
  } finally {
    sending.value = false;
  }
}
function onKeydown(e: KeyboardEvent) {
  if (e.key === 'Enter' && !e.shiftKey) {
    e.preventDefault();
    send();
  }
}
</script>

<template>
  <div class="border-t border-border bg-paper p-2">
    <div
      v-if="sidebar.pendingSelection.value"
      class="mb-2 flex items-start gap-2 rounded border-l-2 border-accent-border bg-accent-bg px-2 py-1 text-xs italic text-muted-fg"
    >
      <span class="flex-1">{{ sidebar.pendingSelection.value }}</span>
      <button type="button" class="text-muted-fg hover:text-fg" aria-label="Clear selection" @click="sidebar.clearSelection">✕</button>
    </div>
    <textarea
      v-model="content"
      data-test="composer-input"
      rows="3"
      placeholder="Ask the session — Enter to send, Shift+Enter for a newline"
      class="w-full resize-none rounded border border-border bg-surface px-2 py-1.5 text-sm focus:border-accent-border focus:outline-none"
      @keydown="onKeydown"
    />
    <div class="mt-1 flex items-center justify-between gap-2 text-xs">
      <div ref="pickerEl" class="relative min-w-0 max-w-[60%]">
        <SessionPresencePill @toggle="sessionOpen = !sessionOpen" />
        <div
          v-if="sessionOpen"
          data-test="composer-session-pop"
          class="absolute bottom-full left-0 z-10 mb-1 w-64"
          @click="sessionOpen = false"
        >
          <SessionSwitcher />
        </div>
      </div>
      <button
        type="button"
        data-test="composer-send"
        class="shrink-0 rounded bg-accent-fg px-3 py-1 text-surface disabled:opacity-50"
        :disabled="sending || content.trim().length === 0"
        @click="send"
      >{{ sending ? 'Sending…' : 'Send' }}</button>
    </div>
    <p v-if="errorMsg" class="mt-1 text-xs text-destructive-fg">{{ errorMsg }}</p>
  </div>
</template>
