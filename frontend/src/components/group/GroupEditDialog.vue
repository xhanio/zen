<script setup lang="ts">
import { computed, ref } from 'vue';
import { useGroupsStore } from '../../stores/groups';
import { parseRuleFormat, composeRule, type CardFormat } from '../../utils/groupRuleFormat';
import { BackendError } from '../../types/api';
import LevelCatalogEditor from './LevelCatalogEditor.vue';
import ConfirmDialog from '../ConfirmDialog.vue';
import type { Group, LevelEntry } from '../../types/entity';

const props = defineProps<{ group: Group }>();
const emit = defineEmits<{ (e: 'close'): void }>();

const groupsStore = useGroupsStore();

const parsed = parseRuleFormat(props.group.rule ?? '');
const name = ref(props.group.name);
const format = ref<CardFormat | null>(parsed.format);
const body = ref(parsed.body);
const catalog = ref<LevelEntry[]>(props.group.level_catalog.map((e) => ({ ...e })));

const saving = ref(false);
const errorMsg = ref<string | null>(null);
const deleteOpen = ref(false);

const FORMATS: { value: CardFormat | null; label: string }[] = [
  { value: null, label: 'None' },
  { value: 'text', label: 'Plain text' },
  { value: 'markdown', label: 'Markdown' },
  { value: 'html', label: 'HTML' },
];

const preview = computed(() => composeRule(body.value, format.value));
const canSave = computed(() => name.value.trim().length > 0);

async function save() {
  if (!canSave.value || saving.value) return;
  saving.value = true;
  errorMsg.value = null;
  try {
    await groupsStore.update(props.group.id, {
      name: name.value.trim(),
      level_catalog: catalog.value,
      rule: composeRule(body.value, format.value),
    });
    emit('close');
  } catch (e) {
    errorMsg.value = e instanceof BackendError ? e.message : String(e);
  } finally {
    saving.value = false;
  }
}

async function confirmDelete() {
  try {
    await groupsStore.remove(props.group.id);
    deleteOpen.value = false;
    emit('close');
  } catch (e) {
    errorMsg.value = e instanceof BackendError ? e.message : String(e);
    deleteOpen.value = false;
  }
}
</script>

<template>
  <div class="w-[46rem] max-w-[92vw] rounded-lg border border-border bg-paper text-paper-fg shadow-xl">
    <div class="flex items-center justify-between border-b border-border px-4 py-3">
      <h2 class="font-serif text-lg font-medium">Edit group</h2>
      <button type="button" data-test="dialog-close" aria-label="Close" class="text-muted-fg hover:text-fg" @click="emit('close')">✕</button>
    </div>

    <div class="grid grid-cols-1 gap-5 p-4 sm:grid-cols-2">
      <div class="space-y-4">
        <div>
          <label class="mb-1 block text-xs font-semibold uppercase tracking-wide text-muted-fg">Name</label>
          <input v-model="name" data-test="group-name" type="text" class="w-full rounded border border-border bg-surface px-2 py-1.5 text-sm focus:border-accent-border focus:outline-none" />
        </div>
        <div>
          <label class="mb-1 block text-xs font-semibold uppercase tracking-wide text-muted-fg">Default card format</label>
          <div class="inline-flex overflow-hidden rounded border border-border">
            <button
              v-for="opt in FORMATS"
              :key="String(opt.value)"
              type="button"
              :data-test="`fmt-${opt.value ?? 'none'}`"
              :aria-pressed="format === opt.value"
              class="border-r border-border px-2.5 py-1 text-xs last:border-r-0"
              :class="format === opt.value ? 'bg-accent-bg font-semibold text-accent-fg' : 'text-muted-fg hover:text-fg'"
              @click="format = opt.value"
            >{{ opt.label }}</button>
          </div>
        </div>
        <div>
          <label class="mb-1 block text-xs font-semibold uppercase tracking-wide text-muted-fg">Levels</label>
          <LevelCatalogEditor v-model="catalog" />
        </div>
      </div>

      <div class="flex flex-col">
        <label class="mb-1 block text-xs font-semibold uppercase tracking-wide text-muted-fg">Rule</label>
        <textarea v-model="body" data-test="rule-input" class="min-h-[9rem] flex-1 resize-y rounded border border-border bg-surface px-2 py-1.5 text-sm focus:border-accent-border focus:outline-none" placeholder="Free-text guidance for the AI when creating cards here."></textarea>
        <div class="mt-3 rounded border border-dashed border-border bg-surface p-2">
          <div class="mb-1 text-[10px] font-semibold uppercase tracking-wide text-muted-fg">Saved rule</div>
          <pre data-test="rule-preview" class="whitespace-pre-wrap font-mono text-xs text-fg">{{ preview || '(empty)' }}</pre>
        </div>
      </div>
    </div>

    <p v-if="errorMsg" class="px-4 text-xs text-destructive-fg">{{ errorMsg }}</p>

    <div class="flex items-center justify-between border-t border-border px-4 py-3">
      <button type="button" data-test="group-delete" class="text-xs text-destructive-fg hover:underline" @click="deleteOpen = true">Delete group</button>
      <div class="flex gap-2 text-xs">
        <button type="button" data-test="dialog-cancel" class="rounded border border-border px-3 py-1 text-muted-fg hover:text-fg" @click="emit('close')">Cancel</button>
        <button type="button" data-test="group-save" :disabled="!canSave || saving" class="rounded bg-accent-fg px-3 py-1 text-surface disabled:opacity-50" @click="save">{{ saving ? 'Saving…' : 'Save' }}</button>
      </div>
    </div>

    <ConfirmDialog
      v-model:open="deleteOpen"
      :title="`Delete group '${group.name}'?`"
      description="This will also delete every card in this group."
      confirm-label="Delete"
      destructive
      @confirm="confirmDelete"
    />
  </div>
</template>
