<script setup lang="ts">
import type { CardDiff } from '../types/entity';

// Unified view: deleted and added lines adjacent, with the changed runs
// highlighted inside. Spans arrive from the backend already computed at rune
// level, so Chinese highlights a clause rather than repainting the paragraph.
defineProps<{ diff: CardDiff; truncated: boolean; before: string; after: string }>();
</script>

<template>
  <div v-if="truncated" data-test="diff-truncated" class="space-y-3 text-sm">
    <p class="text-xs text-muted-fg">改动过大，未生成 diff；以下为改动前后的全文。</p>
    <div>
      <div class="mb-1 text-xs text-muted-fg">改动前</div>
      <pre class="overflow-x-auto whitespace-pre-wrap rounded bg-destructive-bg p-2 text-xs">{{ before }}</pre>
    </div>
    <div>
      <div class="mb-1 text-xs text-muted-fg">改动后</div>
      <pre class="overflow-x-auto whitespace-pre-wrap rounded bg-accent-bg p-2 text-xs">{{ after }}</pre>
    </div>
  </div>

  <div v-else class="text-[13px] leading-relaxed">
    <div
      v-for="f in diff.fields"
      :key="f.key"
      :data-test="`diff-field-${f.key}`"
      class="flex gap-2 border-b border-border px-2 py-1"
    >
      <span class="w-14 shrink-0 text-xs text-muted-fg">{{ f.key }}</span>
      <span class="min-w-0 flex-1">
        <template v-for="(s, i) in f.spans" :key="i">
          <mark v-if="s.op === 'del'" class="bg-destructive-bg line-through">{{ s.text }}</mark>
          <mark v-else-if="s.op === 'add'" class="bg-accent-bg">{{ s.text }}</mark>
          <span v-else>{{ s.text }}</span>
        </template>
      </span>
    </div>

    <div
      v-for="(ln, i) in diff.lines"
      :key="i"
      :data-test="`diff-line-${ln.op}`"
      class="grid grid-cols-[1.25rem_1fr] gap-2 px-2"
      :class="{
        'bg-destructive-bg': ln.op === 'del',
        'bg-accent-bg': ln.op === 'add',
        'text-muted-fg': ln.op === 'ctx',
      }"
    >
      <span class="select-none text-center text-xs" aria-hidden="true">
        {{ ln.op === 'del' ? '−' : ln.op === 'add' ? '+' : '' }}
      </span>
      <span class="min-w-0 whitespace-pre-wrap break-words">
        <template v-if="ln.spans && ln.spans.length">
          <template v-for="(s, j) in ln.spans" :key="j">
            <mark v-if="s.op === 'del'" class="rounded-sm bg-destructive-border/60">{{ s.text }}</mark>
            <mark v-else-if="s.op === 'add'" class="rounded-sm bg-accent-border/60">{{ s.text }}</mark>
            <span v-else>{{ s.text }}</span>
          </template>
        </template>
        <template v-else>{{ ln.text }}</template>
      </span>
    </div>
  </div>
</template>
