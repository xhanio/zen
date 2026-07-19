<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue';
import { storeToRefs } from 'pinia';
import { useConversationsStore } from '../../stores/conversations';
import { usePresenceStore } from '../../stores/presence';
import ConversationTurn from './ConversationTurn.vue';
import type { Message } from '../../types/entity';

const props = defineProps<{ conversationId: string | null }>();
const store = useConversationsStore();
const presence = usePresenceStore();
const { messagesByConv } = storeToRefs(store);

const scrollRef = ref<HTMLElement | null>(null);
const messages = computed<Message[]>(() =>
  props.conversationId ? (messagesByConv.value[props.conversationId] ?? []) : [],
);

// A session's cwd basename can be shared by two sessions in one thread. Collect
// the session_ids whose basename is not unique, so their badge gets a short-id
// suffix; a thread with no collision shows the bare basename.
const collisionSessions = computed<Set<string>>(() => {
  const byBase: Record<string, Set<string>> = {};
  for (const msg of messages.value) {
    if (!msg.session_id) continue;
    const base = presence.cwdBasename(msg.session_cwd ?? '');
    (byBase[base] ??= new Set()).add(msg.session_id);
  }
  const collide = new Set<string>();
  for (const set of Object.values(byBase)) {
    if (set.size > 1) for (const sid of set) collide.add(sid);
  }
  return collide;
});

function sessionNameFor(m: Message): string | null {
  const base = presence.badgeFor(m.session_id, m.session_cwd);
  if (!base) return null;
  if (m.session_id && collisionSessions.value.has(m.session_id)) {
    return `${base} · #${m.session_id.slice(-4)}`;
  }
  return base;
}
function speakerFor(m: Message): string {
  if (m.role === 'user') return 'You';
  if (m.role === 'system') return 'System';
  return sessionNameFor(m) ?? 'Claude Code';
}
function sessionTagFor(m: Message): string | null {
  return m.role === 'user' ? sessionNameFor(m) : null;
}
function sessionColorFor(m: Message): string | null {
  return m.session_id ? presence.sessionColor(m.session_id) : null;
}
function showDivider(i: number): boolean {
  if (i === 0) return false;
  const cur = messages.value[i];
  const prev = messages.value[i - 1];
  return !!cur.session_id && cur.session_id !== prev.session_id;
}
function stateFor(m: Message): 'sent' | 'delivered' | 'undelivered' | null {
  return m.role === 'user' ? store.deliveryState(m.id) : null;
}
async function copy(m: Message) {
  try {
    await navigator.clipboard.writeText(m.content);
  } catch {
    /* clipboard blocked — nothing to recover */
  }
}

watch(messages, async () => {
  await nextTick();
  if (scrollRef.value) scrollRef.value.scrollTop = scrollRef.value.scrollHeight;
}, { deep: true });
</script>

<template>
  <div ref="scrollRef" class="flex min-h-0 flex-1 flex-col gap-3.5 overflow-y-auto px-3 py-3">
    <p v-if="messages.length === 0" class="py-8 text-center text-xs text-muted-fg">No messages yet.</p>
    <template v-for="(m, i) in messages" :key="m.id">
      <div
        v-if="showDivider(i)"
        data-test="session-divider"
        class="flex items-center gap-2 py-1 text-[10px] uppercase tracking-wide text-muted-fg"
      >
        <span class="h-px flex-1 bg-border"></span>
        now talking to {{ sessionNameFor(m) }}
        <span class="h-px flex-1 bg-border"></span>
      </div>
      <ConversationTurn
        :message="m"
        :speaker="speakerFor(m)"
        :session-tag="sessionTagFor(m)"
        :session-color="sessionColorFor(m)"
        :state="stateFor(m)"
        @copy="copy(m)"
        @resend="store.resend(m.id)"
      />
    </template>
  </div>
</template>
