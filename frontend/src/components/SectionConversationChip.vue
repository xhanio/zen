<script setup lang="ts">
import { computed, onMounted, onBeforeUnmount, ref, watch } from 'vue';
import { useConversationsStore } from '../stores/conversations';
import { useChatSidebar } from '../composables/useChatSidebar';
import { relativeTime } from '../utils/relativeTime';

const props = withDefaults(
  defineProps<{
    anchorId: string;
    sourceConversationId: string | null;
    persistent?: boolean;
    disabled?: boolean;
  }>(),
  { persistent: false, disabled: false },
);

const convs = useConversationsStore();
const sidebar = useChatSidebar();

const open = ref(false);
const root = ref<HTMLElement | null>(null);

async function load() {
  await convs.loadForAnchor('card', props.anchorId);
  if (props.sourceConversationId) await convs.ensureConversation(props.sourceConversationId);
}
onMounted(load);
watch(() => props.anchorId, load);
// A thread created through the sidebar changes presence; refresh when it closes.
watch(() => sidebar.open.value, (o, prev) => { if (prev && !o) void load(); });

const items = computed(() => convs.linkedFor(props.anchorId, props.sourceConversationId).items);
const hasAny = computed(() => items.value.length > 0);

function toggle() {
  open.value = !open.value;
  if (open.value) void load(); // freshen the list when peeking
}
function openConversation(id: string) {
  open.value = false;
  void sidebar.openForConversation(id);
}
function newThread() {
  open.value = false;
  void sidebar.openFor('card', props.anchorId, null);
}

function onDocClick(e: MouseEvent) {
  if (open.value && root.value && !root.value.contains(e.target as Node)) open.value = false;
}
function onKey(e: KeyboardEvent) {
  if (e.key === 'Escape') open.value = false;
}
onMounted(() => {
  document.addEventListener('click', onDocClick);
  document.addEventListener('keydown', onKey);
});
onBeforeUnmount(() => {
  document.removeEventListener('click', onDocClick);
  document.removeEventListener('keydown', onKey);
});
</script>

<template>
  <span ref="root" class="relative inline-flex">
    <button
      v-if="persistent || hasAny"
      type="button"
      data-test="conv-chip"
      :aria-expanded="open"
      class="inline-flex items-center gap-1 rounded-full border px-2 py-0.5 text-[11px]"
      :class="hasAny ? 'border-accent-border bg-accent-bg text-accent-fg' : 'border-border text-muted-fg'"
      @click.stop="toggle"
    >💬<template v-if="persistent"> Card</template></button>
    <button
      v-else-if="!disabled"
      type="button"
      data-test="conv-chip-ghost"
      title="Start a discussion on this section"
      class="inline-flex items-center rounded-full border border-dashed border-border px-2 py-0.5 text-[11px] text-muted-fg opacity-0 transition-opacity group-hover:opacity-100 focus:opacity-100"
      @click.stop="newThread"
    >＋💬</button>

    <div
      v-if="open"
      data-test="conv-popover"
      class="absolute right-0 top-6 z-20 w-56 rounded-lg border border-border bg-paper p-1.5 shadow-lg"
    >
      <div class="px-1.5 py-1 text-[10px] font-bold uppercase tracking-wide text-muted-fg">
        {{ persistent ? 'Card' : 'Section' }} · linked
      </div>
      <button
        v-for="it in items"
        :key="it.conversation.id"
        type="button"
        :data-test="`conv-row-${it.conversation.id}`"
        class="flex w-full items-baseline gap-2 rounded px-1.5 py-1 text-left text-xs hover:bg-muted"
        @click="openConversation(it.conversation.id)"
      >
        <span
          class="shrink-0 rounded px-1 text-[9px] font-bold uppercase"
          :class="it.kind === 'origin' ? 'bg-accent-bg text-accent-fg' : 'bg-muted text-muted-fg'"
        >{{ it.kind === 'origin' ? 'Origin' : 'Disc' }}</span>
        <span class="min-w-0 flex-1 truncate">{{ it.conversation.title || 'Untitled' }}</span>
        <span class="shrink-0 text-[10px] text-muted-fg">{{ relativeTime(it.conversation.last_message_at) }}</span>
      </button>
      <p v-if="items.length === 0" class="px-1.5 py-1 text-xs text-muted-fg">No linked conversations.</p>
      <button
        v-if="!disabled"
        type="button"
        data-test="conv-new-thread"
        class="mt-1 w-full border-t border-border px-1.5 py-1.5 text-left text-xs text-accent-fg hover:bg-muted"
        @click="newThread"
      >＋ New thread</button>
    </div>
  </span>
</template>
