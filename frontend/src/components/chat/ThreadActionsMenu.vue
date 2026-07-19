<script setup lang="ts">
import { ref, onBeforeUnmount } from 'vue';
import { useConversationsStore } from '../../stores/conversations';

const props = defineProps<{ conversationId: string }>();
const store = useConversationsStore();
const open = ref(false);
const root = ref<HTMLElement | null>(null);

function close() {
  open.value = false;
  document.removeEventListener('click', onDoc, true);
}
function toggle() {
  open.value = !open.value;
  if (open.value) document.addEventListener('click', onDoc, true);
  else document.removeEventListener('click', onDoc, true);
}
function onDoc(e: MouseEvent) {
  if (root.value && !root.value.contains(e.target as Node)) close();
}
onBeforeUnmount(() => document.removeEventListener('click', onDoc, true));

async function doRename() {
  close();
  const title = window.prompt('Rename conversation');
  if (title && title.trim()) await store.rename(props.conversationId, title.trim());
}
async function doDelete() {
  close();
  if (window.confirm('Delete this conversation? This cannot be undone.')) {
    await store.deleteOne(props.conversationId);
  }
}
</script>

<template>
  <div ref="root" class="relative">
    <button
      type="button"
      data-test="thread-menu-trigger"
      aria-haspopup="menu"
      :aria-expanded="open ? 'true' : 'false'"
      class="rounded px-1.5 text-muted-fg hover:bg-muted"
      @click.stop="toggle"
      @keydown.esc="close"
    >⋯</button>
    <div
      v-if="open"
      data-test="thread-menu"
      role="menu"
      class="absolute right-0 top-7 z-20 w-40 rounded-md border border-border bg-paper p-1 shadow-lg"
    >
      <button
        type="button"
        role="menuitem"
        data-test="thread-rename"
        class="block w-full rounded px-2 py-1 text-left text-xs text-fg hover:bg-muted"
        @click="doRename"
      >Rename</button>
      <button
        type="button"
        role="menuitem"
        data-test="thread-delete"
        class="block w-full rounded px-2 py-1 text-left text-xs text-destructive-fg hover:bg-muted"
        @click="doDelete"
      >Delete conversation</button>
    </div>
  </div>
</template>
