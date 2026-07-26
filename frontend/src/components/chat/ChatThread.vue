<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue';
import { storeToRefs } from 'pinia';
import { useConversationsStore } from '../../stores/conversations';
import { usePresenceStore } from '../../stores/presence';
import ConversationTurn from './ConversationTurn.vue';
import SnapshotRecord from './SnapshotRecord.vue';
import { useSnapshotsStore } from '../../stores/snapshots';
import { buildTimeline } from '../../utils/timeline';
import { useRoute, useRouter } from 'vue-router';
import type { Message } from '../../types/entity';

const props = defineProps<{ conversationId: string | null }>();
const store = useConversationsStore();
const presence = usePresenceStore();
const snapshots = useSnapshotsStore();
const router = useRouter();
const route = useRoute();
const { messagesByConv } = storeToRefs(store);
const { byConversation } = storeToRefs(snapshots);

const scrollRef = ref<HTMLElement | null>(null);
const messages = computed<Message[]>(() =>
  props.conversationId ? (messagesByConv.value[props.conversationId] ?? []) : [],
);

// Snapshot records are peers of messages, ordered purely by time — see
// utils/timeline. Records for this thread are refetched whenever a message
// arrives (stores/conversations), which is why no bus event is needed.
const timeline = computed(() =>
  buildTimeline(
    messages.value,
    props.conversationId ? (byConversation.value[props.conversationId] ?? []) : [],
  ),
);

// Refresh the records whenever the thread gains a message. The watcher skill
// replies after every mutation, so an arriving message is the cue that
// snapshots may have been written — which is why card snapshots need no
// message-bus event of their own. Watching the messages (rather than hooking
// the store's socket) also covers the reconnect catch-up, and keeps the
// conversations store free of any snapshot dependency.
//
// Records are supplementary: a failed load leaves the transcript intact, so
// swallow rather than reject into the void.
watch(
  [() => props.conversationId, () => messages.value.length],
  ([id]) => {
    if (id) void snapshots.loadForConversation(id).catch(() => undefined);
  },
  { immediate: true },
);

// The divider marks a change of session between two *messages*; records sit
// between them without interrupting that reading, so look past them.
function previousMessage(idx: number): Message | null {
  for (let i = idx - 1; i >= 0; i--) {
    const item = timeline.value[i];
    if (item.kind === 'message') return item.message;
  }
  return null;
}

function showDividerAt(idx: number, m: Message): boolean {
  const prev = previousMessage(idx);
  if (!prev) return false;
  return !!m.session_id && m.session_id !== prev.session_id;
}

// Optional chaining because the thread can be mounted without a router
// context (component tests do it). Degrading to "no active record" beats
// throwing through the render.
function activeSnapshotID(): string | null {
  const q = route?.query?.snapshot;
  return typeof q === 'string' ? q : null;
}

function openSnapshot(snapshotID: string) {
  const all = props.conversationId ? (byConversation.value[props.conversationId] ?? []) : [];
  const found = all.find((s) => s.id === snapshotID);
  if (!found) return;
  // Clicking the active record again leaves the change view — it is a toggle.
  const query = activeSnapshotID() === snapshotID ? {} : { snapshot: snapshotID };
  void router?.push({ name: 'card', params: { cardId: found.card_id }, query });
}

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
    <!-- Keyed on the timeline, not the messages: a thread can hold snapshot
         records with no messages at all (an agent edit citing a conversation
         nobody has spoken in), and "No messages yet." above a visible record
         is the UI contradicting itself. -->
    <p v-if="timeline.length === 0" class="py-8 text-center text-xs text-muted-fg">No messages yet.</p>
    <template v-for="(item, i) in timeline" :key="item.id">
      <template v-if="item.kind === 'message'">
        <div
          v-if="showDividerAt(i, item.message)"
          data-test="session-divider"
          class="flex items-center gap-2 py-1 text-[10px] uppercase tracking-wide text-muted-fg"
        >
          <span class="h-px flex-1 bg-border"></span>
          now talking to {{ sessionNameFor(item.message) }}
          <span class="h-px flex-1 bg-border"></span>
        </div>
        <ConversationTurn
          :message="item.message"
          :speaker="speakerFor(item.message)"
          :session-tag="sessionTagFor(item.message)"
          :session-color="sessionColorFor(item.message)"
          :state="stateFor(item.message)"
          @copy="copy(item.message)"
          @resend="store.resend(item.message.id)"
        />
      </template>
      <SnapshotRecord
        v-else-if="item.kind === 'snapshot'"
        :snapshot="item.snapshot"
        :active="activeSnapshotID() === item.snapshot.id"
        @open="openSnapshot"
      />
      <SnapshotRecord
        v-else
        :group="{ snapshots: item.snapshots, linesAdded: item.linesAdded, linesRemoved: item.linesRemoved }"
        @open="openSnapshot"
      />
    </template>
  </div>
</template>
