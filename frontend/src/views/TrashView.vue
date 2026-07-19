<script setup lang="ts">
import { onMounted, ref } from 'vue';
import { storeToRefs } from 'pinia';
import { useTrashStore } from '../stores/trash';
import ConfirmDialog from '../components/ConfirmDialog.vue';

const store = useTrashStore();
const { cards, loading, error } = storeToRefs(store);

onMounted(() => store.load());

const purgeTarget = ref<string | null>(null);
const purgeOpen = ref(false);
const emptyOpen = ref(false);
const emptying = ref(false);

function startPurge(id: string) {
  purgeTarget.value = id;
  purgeOpen.value = true;
}

async function confirmPurge() {
  if (purgeTarget.value) {
    await store.purge(purgeTarget.value);
  }
  purgeTarget.value = null;
  purgeOpen.value = false;
}

async function confirmEmpty() {
  emptying.value = true;
  try {
    await store.empty();
  } finally {
    emptying.value = false;
    emptyOpen.value = false;
  }
}

function timeSince(iso: string | null): string {
  if (!iso) return '';
  const ms = Date.now() - new Date(iso).getTime();
  const min = Math.floor(ms / 60000);
  if (min < 60) return `${min}m ago`;
  const hr = Math.floor(min / 60);
  if (hr < 24) return `${hr}h ago`;
  return `${Math.floor(hr / 24)}d ago`;
}
</script>

<template>
  <div>
    <div class="mb-3 flex items-center justify-between">
      <h1 class="text-xl font-semibold text-fg">Trash</h1>
      <button
        v-if="cards.length > 0"
        type="button"
        data-test="empty-trash"
        :disabled="emptying"
        class="rounded border border-destructive-border px-3 py-1 text-sm text-destructive-fg hover:bg-destructive-bg disabled:opacity-50"
        @click="emptyOpen = true"
      >Empty Trash</button>
    </div>
    <p v-if="loading" class="text-sm text-muted-fg">Loading…</p>
    <p v-else-if="error" class="text-sm text-destructive-fg">{{ error }}</p>
    <p v-else-if="cards.length === 0" class="text-sm text-muted-fg">
      Trash is empty. Soft-deleted cards land here.
    </p>
    <ul v-else class="space-y-2">
      <li
        v-for="card in cards"
        :key="card.id"
        data-test="trash-row"
        class="flex items-center justify-between rounded border border-border bg-surface p-3"
      >
        <div>
          <router-link
            :to="{ name: 'card', params: { cardId: card.id } }"
            data-test="trash-row-title"
            class="text-sm font-medium text-fg hover:underline"
          >{{ card.title }}</router-link>
          <p class="text-xs text-muted-fg">
            deleted {{ timeSince(card.deleted_at) }}
          </p>
        </div>
        <div class="flex gap-2">
          <button
            type="button"
            class="rounded border border-border px-2 py-1 text-xs text-fg hover:bg-muted"
            @click="store.restore(card.id)"
          >Restore</button>
          <button
            type="button"
            class="rounded border border-destructive-border px-2 py-1 text-xs text-destructive-fg hover:bg-destructive-bg"
            @click="startPurge(card.id)"
          >Delete Permanently</button>
        </div>
      </li>
    </ul>

    <ConfirmDialog
      v-model:open="purgeOpen"
      title="Delete card permanently?"
      description="This cannot be undone."
      confirm-label="Delete"
      destructive
      @confirm="confirmPurge"
    />
    <ConfirmDialog
      v-model:open="emptyOpen"
      title="Empty Trash?"
      :description="`Permanently delete all ${cards.length} card${cards.length === 1 ? '' : 's'} in Trash. This cannot be undone.`"
      confirm-label="Empty Trash"
      destructive
      @confirm="confirmEmpty"
    />
  </div>
</template>
