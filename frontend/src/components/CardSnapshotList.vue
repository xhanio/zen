<script setup lang="ts">
import { computed, watch } from 'vue';
import { storeToRefs } from 'pinia';
import { useSnapshotsStore } from '../stores/snapshots';

// The fallback entry point. Hand edits and backfilled baselines belong to no
// conversation, so the thread timeline can never show them; without this list
// they would be the one kind of change you cannot get back to.
const props = defineProps<{ cardId: string; activeId?: string | null }>();
const emit = defineEmits<{ (e: 'open', id: string): void }>();

const store = useSnapshotsStore();
const { byCard } = storeToRefs(store);

watch(
  () => props.cardId,
  (id) => {
    if (id) void store.loadForCard(id).catch(() => undefined);
  },
  { immediate: true },
);

const rows = computed(() => byCard.value[props.cardId] ?? []);

const kindLabel: Record<string, string> = {
  baseline: '基线',
  create: '创建',
  decompose: '拆分为章节',
  update: '',
};
const actorLabel: Record<string, string> = { agent: 'Claude', user: '你', system: '系统' };

function labelFor(changeKind: string, actor: string): string {
  return kindLabel[changeKind] || actorLabel[actor] || actor;
}

function timeFor(at: string): string {
  const d = new Date(at);
  return isNaN(d.getTime()) ? '' : d.toLocaleString([], { month: 'numeric', day: 'numeric', hour: 'numeric', minute: '2-digit' });
}
</script>

<template>
  <section data-test="snapshot-list" class="text-xs">
    <h4 class="mb-1 font-medium text-fg">全部快照</h4>
    <p v-if="rows.length === 0" class="text-muted-fg">暂无快照</p>
    <button
      v-for="s in rows"
      :key="s.id"
      type="button"
      data-test="snapshot-list-row"
      class="flex w-full items-center gap-2 rounded px-1.5 py-1 text-left hover:bg-muted"
      :class="activeId === s.id ? 'bg-accent-bg' : ''"
      @click="emit('open', s.id)"
    >
      <span class="shrink-0 tabular-nums text-muted-fg">#{{ s.seq }}</span>
      <span class="min-w-0 flex-1 truncate">{{ labelFor(s.change_kind, s.actor) }}</span>
      <span class="shrink-0 tabular-nums text-accent-fg">+{{ s.lines_added }}</span>
      <span class="shrink-0 tabular-nums text-destructive-fg">−{{ s.lines_removed }}</span>
      <span class="shrink-0 tabular-nums text-muted-fg">{{ timeFor(s.created_at) }}</span>
    </button>
  </section>
</template>
