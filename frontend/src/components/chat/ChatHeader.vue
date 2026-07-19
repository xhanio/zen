<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch, watchEffect } from 'vue';
import { RouterLink } from 'vue-router';
import { storeToRefs } from 'pinia';
import { useConversationsStore } from '../../stores/conversations';
import { useChatSidebar } from '../../composables/useChatSidebar';
import ThreadSwitcher from './ThreadSwitcher.vue';
import ThreadActionsMenu from './ThreadActionsMenu.vue';

defineEmits<{ (e: 'close'): void }>();

const store = useConversationsStore();
const sidebar = useChatSidebar();
const { activeID, byID, list } = storeToRefs(store);

const conv = computed(() => (activeID.value ? byID.value[activeID.value] : null));

// The title ▾ opens the thread list for this card. (The session picker moved to
// the composer, where the send decision is made.)
const threadOpen = ref(false);

// Close the popover on an outside click, like the section/thread action menus
// already do — a dropdown that only closes on an inner click gets stuck open
// when you click away.
const rootEl = ref<HTMLElement | null>(null);
function onDocClick(e: MouseEvent) {
  if (rootEl.value && !rootEl.value.contains(e.target as Node)) {
    threadOpen.value = false;
  }
}
watch(threadOpen, (open) => {
  if (open) document.addEventListener('click', onDocClick, true);
  else document.removeEventListener('click', onDocClick, true);
});
onBeforeUnmount(() => document.removeEventListener('click', onDocClick, true));

// Load this anchor's full thread list when the panel opens (openFor only fetches
// limit:1 to pick the active thread). Without it threadCount is always ≤1 and
// the title switcher never appears.
watch(
  () => [sidebar.anchorKind.value, sidebar.anchorID.value] as const,
  ([kind, id]) => {
    if (id) store.loadList({ anchorKind: kind ?? undefined, anchorID: id ?? undefined });
  },
  { immediate: true },
);

// The switcher is available whenever there is a card to have threads on — not
// gated on count, or a card with a single thread would have no way to reach
// "+ New thread". The count only appears once there is more than one.
const hasAnchor = computed(() => !!sidebar.anchorID.value);
const threadCount = computed(
  () =>
    list.value.filter(
      (c) => c.anchor_kind === sidebar.anchorKind.value && c.anchor_id === sidebar.anchorID.value,
    ).length,
);

const anchorTitle = ref<string | null>(null);
watchEffect(async () => {
  anchorTitle.value = await store.resolveAnchorTitle(sidebar.anchorKind.value, sidebar.anchorID.value);
});
const breadcrumbHref = computed(() => {
  const k = sidebar.anchorKind.value;
  const id = sidebar.anchorID.value;
  if (!k || !id) return null;
  return k === 'card' ? `/c/${id}` : k === 'group' ? `/g/${id}` : null;
});
const breadcrumbLabel = computed(() => {
  const k = sidebar.anchorKind.value;
  const id = sidebar.anchorID.value;
  if (!k || !id) return '';
  return `on ${k} · ${anchorTitle.value ?? id.slice(-6)}`;
});
</script>

<template>
  <div ref="rootEl" class="border-b border-border bg-paper px-3 py-2">
    <div class="flex items-start gap-2">
      <div class="relative min-w-0 flex-1">
        <div class="flex items-center gap-1.5">
          <span data-test="chat-title" class="truncate text-sm font-semibold text-fg">{{ conv?.title || 'New conversation' }}</span>
          <button
            v-if="hasAnchor"
            type="button"
            data-test="thread-switch-trigger"
            class="shrink-0 rounded px-1 text-xs text-muted-fg hover:bg-muted"
            :aria-expanded="threadOpen ? 'true' : 'false'"
            title="Switch or start a thread on this card"
            @click="threadOpen = !threadOpen"
          >▾<span v-if="threadCount > 1"> {{ threadCount }}</span></button>
          <ThreadActionsMenu v-if="activeID" :conversation-id="activeID" />
        </div>
        <RouterLink
          v-if="breadcrumbHref"
          :to="breadcrumbHref"
          data-test="chat-breadcrumb"
          class="text-xs text-accent-fg hover:underline"
        >{{ breadcrumbLabel }} ↗</RouterLink>
        <div v-if="threadOpen" data-test="thread-switch-pop" class="absolute left-0 right-0 top-8 z-10" @click="threadOpen = false">
          <ThreadSwitcher />
        </div>
      </div>
      <button
        type="button"
        data-test="chat-close"
        class="rounded px-2 py-1 text-muted-fg hover:bg-muted"
        aria-label="Close sidebar"
        @click="$emit('close')"
      >✕</button>
    </div>
  </div>
</template>
