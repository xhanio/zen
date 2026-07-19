<script setup lang="ts">
import { onMounted, ref, watch } from 'vue';
import { renderMarkdown } from '../utils/markdown';
import { wrapHighlights, type Highlight } from '../utils/highlightText';

const props = defineProps<{
  source: string;
  highlights?: Highlight[];
}>();

const root = ref<HTMLElement | null>(null);

function paint() {
  const el = root.value;
  if (!el) return;
  el.innerHTML = renderMarkdown(props.source);
  if (props.highlights && props.highlights.length > 0) {
    wrapHighlights(el, props.highlights);
  }
}

onMounted(paint);
watch(() => [props.source, props.highlights], paint);
</script>

<template>
  <div ref="root" class="md-body max-w-none bg-paper text-paper-fg"></div>
</template>

<style scoped>
/* markdown-it emits raw HTML into this container via innerHTML, so the
   generated tags carry no scope attribute — reach them with :deep().
   Tailwind's preflight strips heading sizes, list markers, and code
   backgrounds; these rules restore them using the Zen theme tokens.
   Code/quote surfaces use translucent grey so they read on the white
   card paper AND on the adaptive conversation surface, light or dark. */
.md-body {
  line-height: 1.6;
  word-break: break-word;
}
.md-body :deep(> :first-child) { margin-top: 0; }
.md-body :deep(> :last-child) { margin-bottom: 0; }

.md-body :deep(p) { margin: 0.5em 0; }

.md-body :deep(h1),
.md-body :deep(h2),
.md-body :deep(h3),
.md-body :deep(h4),
.md-body :deep(h5),
.md-body :deep(h6) {
  margin: 1.1em 0 0.4em;
  font-weight: 600;
  line-height: 1.3;
  color: inherit;
}
.md-body :deep(h1) { font-size: 1.4em; }
.md-body :deep(h2) { font-size: 1.22em; }
.md-body :deep(h3) { font-size: 1.08em; }
.md-body :deep(h4),
.md-body :deep(h5),
.md-body :deep(h6) {
  font-size: 0.92em;
  text-transform: uppercase;
  letter-spacing: 0.04em;
  color: var(--muted-fg);
}

.md-body :deep(ul),
.md-body :deep(ol) { margin: 0.5em 0; padding-left: 1.4em; }
.md-body :deep(ul) { list-style: disc; }
.md-body :deep(ol) { list-style: decimal; }
.md-body :deep(li) { margin: 0.2em 0; }
.md-body :deep(li::marker) { color: var(--muted-fg); }
.md-body :deep(li > ul),
.md-body :deep(li > ol) { margin: 0.2em 0; }

.md-body :deep(a) {
  color: var(--accent-fg);
  text-decoration: underline;
  text-underline-offset: 2px;
}

.md-body :deep(blockquote) {
  margin: 0.6em 0;
  padding: 0.15em 0.85em;
  border-left: 3px solid var(--accent-border);
  color: var(--muted-fg);
  font-style: italic;
}
.md-body :deep(blockquote > :first-child) { margin-top: 0; }
.md-body :deep(blockquote > :last-child) { margin-bottom: 0; }

.md-body :deep(code) {
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
  font-size: 0.88em;
  background: rgba(127, 127, 127, 0.16);
  border-radius: 3px;
  padding: 0.12em 0.36em;
}
.md-body :deep(pre) {
  margin: 0.6em 0;
  padding: 0.7em 0.85em;
  background: rgba(127, 127, 127, 0.1);
  border: 1px solid rgba(127, 127, 127, 0.22);
  border-radius: 6px;
  overflow-x: auto;
}
.md-body :deep(pre code) {
  display: block;
  background: none;
  padding: 0;
  font-size: 0.86em;
  line-height: 1.5;
}

.md-body :deep(hr) {
  margin: 1em 0;
  border: 0;
  border-top: 1px solid var(--border);
}

.md-body :deep(table) {
  margin: 0.6em 0;
  border-collapse: collapse;
  font-size: 0.95em;
}
.md-body :deep(th),
.md-body :deep(td) {
  border: 1px solid var(--border);
  padding: 0.3em 0.6em;
  text-align: left;
}
.md-body :deep(th) { background: rgba(127, 127, 127, 0.12); font-weight: 600; }

.md-body :deep(strong) { font-weight: 600; }
.md-body :deep(img) { max-width: 100%; }
</style>
