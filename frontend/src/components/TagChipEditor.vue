<script setup lang="ts">
import { ref } from 'vue';

const props = withDefaults(
  defineProps<{
    tags: string[];
    allTags: string[];
    saving?: boolean;
    error?: string | null;
    readonly?: boolean;
  }>(),
  { saving: false, error: null, readonly: false },
);
const emit = defineEmits<{ (e: 'update', tags: string[]): void }>();

const draft = ref('');
const listId = `taglist-${Math.random().toString(36).slice(2, 8)}`;

function commit() {
  const name = draft.value.trim().toLowerCase();
  if (!name) return;
  if (props.tags.includes(name)) {
    draft.value = '';
    return;
  }
  emit('update', [...props.tags, name]);
  draft.value = '';
}

function removeAt(idx: number) {
  const next = props.tags.slice();
  next.splice(idx, 1);
  emit('update', next);
}

function onKeydown(e: KeyboardEvent) {
  if (e.key === 'Enter' || e.key === ',') {
    e.preventDefault();
    commit();
  } else if (e.key === 'Backspace' && draft.value === '' && props.tags.length > 0) {
    e.preventDefault();
    removeAt(props.tags.length - 1);
  }
}
</script>

<template>
  <div>
    <div class="flex flex-wrap items-center gap-1">
      <span
        v-for="(tag, idx) in tags"
        :key="tag"
        class="inline-flex items-center gap-1 rounded bg-muted px-2 py-0.5 text-xs text-fg"
      >
        {{ tag }}
        <button
          v-if="!readonly"
          type="button"
          class="rounded text-muted-fg hover:text-destructive-fg focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-gray-500"
          :aria-label="`Remove tag ${tag}`"
          @click="removeAt(idx)"
        >
          ×
        </button>
      </span>
      <input
        v-if="!readonly"
        v-model="draft"
        type="text"
        :list="listId"
        :disabled="saving"
        placeholder="add tag…"
        aria-label="Add tag"
        class="min-w-[6rem] flex-1 rounded border border-transparent px-2 py-0.5 text-xs focus:border-border focus:outline-none focus-visible:ring-2 focus-visible:ring-gray-500"
        @keydown="onKeydown"
      />
      <datalist v-if="!readonly" :id="listId">
        <option v-for="t in allTags" :key="t" :value="t" />
      </datalist>
    </div>
    <p v-if="error" class="mt-1 text-xs text-destructive-fg">{{ error }}</p>
  </div>
</template>
