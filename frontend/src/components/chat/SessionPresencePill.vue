<script setup lang="ts">
import { computed } from 'vue';
import { storeToRefs } from 'pinia';
import { usePresenceStore } from '../../stores/presence';

defineEmits<{ (e: 'toggle'): void }>();

const presence = usePresenceStore();
const { sessions, selectedSessionID } = storeToRefs(presence);

const current = computed(() => sessions.value.find((s) => s.session_id === selectedSessionID.value) ?? null);
const connected = computed(() => current.value !== null);
const name = computed(() => (current.value ? presence.displayName(current.value) : 'No session'));
const shortId = computed(() => current.value?.session_id.slice(0, 8) ?? '');
</script>

<template>
  <button
    type="button"
    data-test="presence-pill"
    class="flex w-full items-center gap-2 rounded-md bg-muted px-2 py-1 text-xs text-fg hover:bg-border"
    :title="connected ? 'Talking to this Claude Code session — click to switch' : 'No session selected — click to pick one'"
    @click="$emit('toggle')"
  >
    <span
      data-test="presence-dot"
      class="inline-block h-1.5 w-1.5 rounded-full"
      :class="connected ? 'bg-accent-fg' : 'bg-muted-fg'"
    ></span>
    <span class="min-w-0 flex-1 truncate text-left">
      {{ name }}<span v-if="shortId" class="text-muted-fg"> · {{ shortId }}</span>
    </span>
    <span class="text-muted-fg">▾</span>
  </button>
</template>
