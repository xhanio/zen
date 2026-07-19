<script setup lang="ts">
import { computed } from 'vue';
import MarkdownBody from '../MarkdownBody.vue';
import type { Message } from '../../types/entity';

const props = defineProps<{
  message: Message;
  speaker: string;
  sessionTag?: string | null;
  sessionColor?: string | null;
  state: 'sent' | 'delivered' | 'undelivered' | null;
}>();
defineEmits<{ (e: 'resend'): void; (e: 'copy'): void }>();

const isUser = computed(() => props.message.role === 'user');
const isAssistant = computed(() => props.message.role === 'assistant');
const time = computed(() => {
  const t = props.message.created_at;
  if (!t) return '';
  const d = new Date(t);
  return isNaN(d.getTime()) ? '' : d.toLocaleTimeString([], { hour: 'numeric', minute: '2-digit' });
});
const ribbonClass = computed(() =>
  props.state === 'undelivered' ? 'bg-destructive-fg' : isUser.value ? 'bg-accent-fg' : 'bg-border',
);
</script>

<template>
  <div data-test="turn" class="group flex gap-2.5">
    <div data-test="turn-ribbon" class="w-[3px] shrink-0 self-stretch rounded-full" :class="ribbonClass"></div>
    <div class="min-w-0 flex-1">
      <div class="mb-1 flex items-center gap-2">
        <span
          class="text-[10px] font-bold uppercase tracking-[0.03em]"
          :class="isUser ? 'text-accent-fg' : 'text-muted-fg'"
        >{{ speaker }}</span>
        <span
          v-if="sessionColor"
          data-test="turn-session-dot"
          class="inline-block h-1.5 w-1.5 rounded-full"
          :style="{ backgroundColor: sessionColor }"
        ></span>
        <span
          v-if="sessionTag"
          data-test="turn-session"
          class="text-[10px] text-muted-fg"
        >→ {{ sessionTag }}</span>
        <span class="text-[10px] text-muted-fg">{{ time }}</span>
        <span class="ml-auto opacity-0 transition-opacity group-hover:opacity-100">
          <button
            type="button"
            data-test="turn-copy"
            class="rounded border border-border px-1.5 text-[10px] text-muted-fg hover:text-fg"
            @click="$emit('copy')"
          >copy</button>
        </span>
      </div>

      <div
        v-if="message.selection_text"
        data-test="turn-quote"
        class="mb-1 border-l-2 border-accent-border bg-accent-bg px-2 py-0.5 text-[11px] italic text-muted-fg"
      >{{ message.selection_text }}</div>

      <MarkdownBody v-if="isAssistant" :source="message.content" class="text-[12.5px]" />
      <div v-else class="whitespace-pre-wrap text-[12.5px] leading-relaxed text-fg">{{ message.content }}</div>

      <div
        v-if="state"
        data-test="turn-state"
        class="mt-1 flex items-center gap-1.5 text-[10px]"
        :class="state === 'undelivered' ? 'text-destructive-fg' : state === 'delivered' ? 'text-accent-fg' : 'text-muted-fg'"
      >
        <template v-if="state === 'sent'">✓ sent</template>
        <template v-else-if="state === 'delivered'">
          <span class="inline-block h-1.5 w-1.5 animate-pulse rounded-full bg-accent-fg"></span> Claude Code has it…
        </template>
        <template v-else>
          ⚠ not delivered — no session picked it up
          <button
            type="button"
            data-test="turn-resend"
            class="rounded border border-border px-1.5 text-muted-fg hover:text-fg"
            @click="$emit('resend')"
          >resend</button>
        </template>
      </div>
    </div>
  </div>
</template>
