<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue';

const props = withDefaults(
  defineProps<{
    modelValue: string;
    format?: 'markdown' | 'html' | 'text';
    multiline?: boolean;
    placeholder?: string;
    saving?: boolean;
    error?: string | null;
  }>(),
  { format: 'markdown', multiline: false, placeholder: '', saving: false, error: null },
);

const emit = defineEmits<{
  (e: 'save', value: string, format: 'markdown' | 'html' | 'text'): void;
  (e: 'cancel'): void;
}>();

const editing = ref(false);
const draft = ref(props.modelValue);
const draftFormat = ref<'markdown' | 'html' | 'text'>(props.format);
const inputRef = ref<HTMLInputElement | HTMLTextAreaElement | null>(null);

watch(
  () => props.modelValue,
  (v) => {
    if (!editing.value) draft.value = v;
  },
);

watch(
  () => props.format,
  (f) => {
    if (!editing.value) draftFormat.value = f;
  },
);

watch(
  () => props.error,
  (e) => {
    if (e && !editing.value) {
      editing.value = true;
    }
  },
);

async function enterEdit() {
  if (editing.value) return;
  draft.value = props.modelValue;
  draftFormat.value = props.format;
  editing.value = true;
  await nextTick();
  inputRef.value?.focus();
  if (inputRef.value && 'select' in inputRef.value) {
    (inputRef.value as HTMLInputElement).select();
  }
}

function commit() {
  const next = draft.value;
  const nextFormat = draftFormat.value;
  if (next === props.modelValue && nextFormat === props.format) {
    editing.value = false;
    return;
  }
  emit('save', next, nextFormat);
  editing.value = false;
}

function cancel() {
  draft.value = props.modelValue;
  draftFormat.value = props.format;
  editing.value = false;
  emit('cancel');
}

function onKeydown(e: KeyboardEvent) {
  if (e.key === 'Escape') {
    e.preventDefault();
    cancel();
  } else if (props.multiline && (e.metaKey || e.ctrlKey) && e.key === 'Enter') {
    e.preventDefault();
    commit();
  } else if (!props.multiline && e.key === 'Enter') {
    e.preventDefault();
    commit();
  }
}

const displayValue = computed(() => props.modelValue || props.placeholder);
</script>

<template>
  <div @dblclick="enterEdit">
    <template v-if="editing">
      <div v-if="multiline" class="mb-1 flex items-center gap-2 text-xs">
        <label class="text-muted-fg">Format</label>
        <select v-model="draftFormat" class="rounded border border-border px-1 py-0.5">
          <option value="markdown">Markdown</option>
          <option value="text">Plain text</option>
          <option value="html">HTML</option>
        </select>
      </div>
      <textarea
        v-if="multiline"
        ref="inputRef"
        v-model="draft"
        :disabled="saving"
        rows="6"
        class="w-full rounded border border-border px-2 py-1 text-sm focus:border-border focus:outline-none"
        @blur="commit"
        @keydown="onKeydown"
      />
      <input
        v-else
        ref="inputRef"
        v-model="draft"
        :disabled="saving"
        type="text"
        class="w-full rounded border border-border px-2 py-1 text-sm focus:border-border focus:outline-none"
        @blur="commit"
        @keydown="onKeydown"
      />
      <p v-if="error" class="mt-1 text-xs text-destructive-fg">{{ error }}</p>
    </template>
    <div
      v-else
      class="cursor-text rounded px-1 hover:bg-muted"
      :class="{ 'text-muted-fg italic': !modelValue, 'whitespace-pre-wrap': !$slots.default }"
    >
      <slot :value="modelValue" :placeholder="placeholder">{{ displayValue }}</slot>
    </div>
  </div>
</template>
