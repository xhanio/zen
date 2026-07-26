<script setup lang="ts">
import { computed, ref } from 'vue';
import type { CardSnapshot } from '../../types/entity';

// A record is an event on the timeline, not speech: no speaker, no bubble.
// Keeping it visually distinct from ConversationTurn is a design requirement,
// not a preference — it must never read as something Claude said.
const props = defineProps<{
  snapshot?: CardSnapshot;
  group?: { snapshots: CardSnapshot[]; linesAdded: number; linesRemoved: number };
  active?: boolean;
}>();
const emit = defineEmits<{ (e: 'open', id: string): void }>();

const expanded = ref(false);

const added = computed(() => props.group?.linesAdded ?? props.snapshot?.lines_added ?? 0);
const removed = computed(() => props.group?.linesRemoved ?? props.snapshot?.lines_removed ?? 0);

const label = computed(() => {
  if (props.group) return `${props.group.snapshots.length} 张卡片被修改`;
  return props.snapshot?.card_title ?? '未命名卡片';
});

// Decompose empties the parent, so its diff reads as a full deletion. Naming
// it a restructure stops that from looking like data loss.
const kindLabel = computed(() => (props.snapshot?.change_kind === 'decompose' ? '拆分为章节' : ''));

const seqPair = computed(() => {
  const s = props.snapshot;
  if (!s || s.seq <= 1) return '';
  return `#${s.seq - 1}→#${s.seq}`;
});

const time = computed(() => {
  const at = props.group?.snapshots[0]?.created_at ?? props.snapshot?.created_at;
  if (!at) return '';
  const d = new Date(at);
  return isNaN(d.getTime()) ? '' : d.toLocaleTimeString([], { hour: 'numeric', minute: '2-digit' });
});

function onClick() {
  if (props.group) {
    expanded.value = !expanded.value;
    return;
  }
  if (props.snapshot) emit('open', props.snapshot.id);
}
</script>

<template>
  <div>
    <button
      type="button"
      data-test="snapshot-record"
      class="flex w-full items-center gap-2 rounded border border-l-[3px] border-border px-2 py-1 text-left text-[11px] text-fg hover:bg-muted"
      :class="active ? 'border-accent-border bg-accent-bg' : 'border-l-muted-fg'"
      @click="onClick"
    >
      <span aria-hidden="true">◧</span>
      <span class="min-w-0 flex-1 truncate">{{ label }}</span>
      <span v-if="kindLabel" data-test="snapshot-kind" class="shrink-0 text-muted-fg">{{ kindLabel }}</span>
      <span v-if="seqPair" class="shrink-0 tabular-nums text-muted-fg">{{ seqPair }}</span>
      <span class="shrink-0 tabular-nums text-accent-fg">+{{ added }}</span>
      <span class="shrink-0 tabular-nums text-destructive-fg">−{{ removed }}</span>
      <span v-if="group" data-test="snapshot-fold" aria-hidden="true">{{ expanded ? '⌃' : '⌄' }}</span>
      <span class="shrink-0 tabular-nums text-muted-fg">{{ time }}</span>
    </button>

    <div v-if="expanded && group" class="mt-0.5 space-y-0.5 pl-3">
      <SnapshotRecord
        v-for="s in group.snapshots"
        :key="s.id"
        :snapshot="s"
        @open="(id: string) => emit('open', id)"
      />
    </div>
  </div>
</template>
