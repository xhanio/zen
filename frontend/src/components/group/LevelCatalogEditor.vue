<script setup lang="ts">
import { computed } from 'vue';
import type { LevelEntry } from '../../types/entity';
import { sortCatalog } from '../../utils/levelCatalog';
import { levelColor } from '../../utils/levelPalette';

const props = defineProps<{ modelValue: LevelEntry[] }>();
const emit = defineEmits<{
  (e: 'update:modelValue', next: LevelEntry[]): void;
}>();

const sorted = computed(() => sortCatalog(props.modelValue));

// New entries emit with id: '' so the server assigns a fresh ULID.
// Existing entries preserve their id through every edit, which is the
// contract the backend cascade rules rely on.
function update(index: number, patch: Partial<LevelEntry>) {
  const next = sorted.value.map((e, i) => (i === index ? { ...e, ...patch } : e));
  emit('update:modelValue', next);
}

function remove(index: number) {
  emit(
    'update:modelValue',
    sorted.value.filter((_, i) => i !== index),
  );
}

function add() {
  const next =
    sorted.value.length === 0 ? 0 : sorted.value[sorted.value.length - 1].weight + 1;
  emit('update:modelValue', [...sorted.value, { id: '', weight: next, name: 'new' }]);
}

// Client-side pre-check for duplicate names. Server also rejects with
// Conflict, but flagging in-editor gives faster feedback.
const nameConflicts = computed(() => {
  const seen = new Map<string, number>();
  const conflicts = new Set<number>();
  sorted.value.forEach((e, i) => {
    const n = e.name.trim();
    if (!n) return;
    if (seen.has(n)) {
      conflicts.add(i);
      conflicts.add(seen.get(n)!);
    } else {
      seen.set(n, i);
    }
  });
  return conflicts;
});
</script>

<template>
  <div>
    <ul class="space-y-1">
      <li v-for="(entry, idx) in sorted" :key="entry.id || `new-${idx}`" class="flex items-center gap-1">
        <span
          class="h-2.5 w-2.5 shrink-0 rounded-full"
          :style="{ background: levelColor(modelValue, entry.weight).fg }"
        ></span>
        <input
          type="number"
          step="0.001"
          :value="entry.weight"
          class="w-16 rounded border border-border px-1 py-0.5 text-xs"
          @input="(e) => update(idx, { weight: parseFloat((e.target as HTMLInputElement).value) })"
        />
        <input
          type="text"
          :value="entry.name"
          :class="[
            'flex-1 rounded border px-1 py-0.5 text-xs',
            nameConflicts.has(idx) ? 'border-destructive-border bg-destructive-bg' : 'border-border',
          ]"
          @input="(e) => update(idx, { name: (e.target as HTMLInputElement).value })"
        />
        <button
          type="button"
          data-test="entry-remove"
          class="rounded px-1 text-xs text-destructive-fg hover:bg-destructive-bg"
          @click="remove(idx)"
        >×</button>
      </li>
    </ul>
    <button
      type="button"
      data-test="add-entry"
      class="mt-2 w-full rounded border border-accent-border px-2 py-1 text-xs text-accent-fg hover:bg-accent-bg"
      @click="add"
    >+ Add level</button>
  </div>
</template>
