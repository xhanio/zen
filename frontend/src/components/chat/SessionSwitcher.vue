<script setup lang="ts">
import { storeToRefs } from 'pinia';
import { usePresenceStore } from '../../stores/presence';

const presence = usePresenceStore();
const { sessions, selectedSessionID } = storeToRefs(presence);

function pick(id: string) {
  presence.select(id);
}
</script>

<template>
  <div class="rounded-lg border border-border bg-paper p-1 shadow-lg">
    <button
      v-for="s in sessions"
      :key="s.session_id"
      type="button"
      data-test="session-row"
      class="flex w-full items-center gap-2 rounded px-2 py-1.5 text-left text-xs hover:bg-muted"
      :class="s.session_id === selectedSessionID ? 'bg-accent-bg' : ''"
      @click="pick(s.session_id)"
    >
      <span aria-hidden="true" class="text-fg">●</span>
      <span class="min-w-0 flex-1 truncate">
        {{ presence.displayName(s) }}<span class="text-muted-fg"> · {{ s.session_id.slice(0, 8) }}</span>
      </span>
    </button>
    <p v-if="sessions.length === 0" data-test="session-empty" class="px-2 py-1.5 text-xs text-muted-fg">
      No sessions connected. Start Claude Code with the zen plugin.
    </p>
  </div>
</template>
