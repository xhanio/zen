<script setup lang="ts">
import { computed, onMounted, onBeforeUnmount, ref, watch } from 'vue';
import { storeToRefs } from 'pinia';
import { useSnapshotsStore } from '../stores/snapshots';
import CardSnapshotList from './CardSnapshotList.vue';

// The secondary entry to a card's history. Hand edits and backfilled baselines
// belong to no conversation, so the thread timeline can never show them — this
// is how they stay reachable.
//
// It is deliberately a chip + popover rather than a standing panel: the design
// calls this surface secondary, and a permanent rail next to the references
// panel would take more of the card view than the reading column it supports.
// The gesture matches SectionConversationChip, which readers already know.
const props = defineProps<{ cardId: string; activeId?: string | null }>();
const emit = defineEmits<{ (e: 'open', id: string): void }>();

const store = useSnapshotsStore();
const { byCard } = storeToRefs(store);

const open = ref(false);
const root = ref<HTMLElement | null>(null);

function load() {
  if (props.cardId) void store.loadForCard(props.cardId).catch(() => undefined);
}
onMounted(load);
watch(() => props.cardId, load);

const rows = computed(() => byCard.value[props.cardId] ?? []);
// A card with only its baseline has no history worth reading, so the chip
// stays quiet — lit means "something actually happened here".
const hasHistory = computed(() => rows.value.length > 1);

function toggle() {
  open.value = !open.value;
  if (open.value) load(); // freshen while peeking
}

function onOpenSnapshot(id: string) {
  open.value = false;
  emit('open', id);
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
      type="button"
      data-test="snapshot-chip"
      :aria-expanded="open"
      :title="hasHistory ? `${rows.length} 条快照` : '暂无改动记录'"
      class="inline-flex items-center gap-1 rounded-full border px-2 py-0.5 text-[11px]"
      :class="hasHistory ? 'border-accent-border bg-accent-bg text-accent-fg' : 'border-border text-muted-fg'"
      @click.stop="toggle"
    >🕘<span v-if="hasHistory" data-test="snapshot-chip-count">{{ rows.length }}</span></button>

    <div
      v-if="open"
      data-test="snapshot-popover"
      class="absolute right-0 top-6 z-20 w-64 rounded-lg border border-border bg-paper p-1.5 shadow-lg"
    >
      <CardSnapshotList :card-id="cardId" :active-id="activeId" @open="onOpenSnapshot" />
    </div>
  </span>
</template>
