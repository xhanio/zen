<script setup lang="ts">
import { onMounted, ref, watch } from 'vue';
import { sanitizeCardHtml } from '../utils/sanitizeHtml';
import { wrapHighlights, type Highlight } from '../utils/highlightText';

const props = defineProps<{
  source: string;
  highlights?: Highlight[];
}>();

const hostRef = ref<HTMLDivElement | null>(null);

const HIGHLIGHT_STYLE =
  '.zen-ref{background:#fef9c3;border-radius:2px;cursor:pointer;padding:0 1px}' +
  '.zen-ref:hover{background:#fde68a}';

function paint() {
  const host = hostRef.value;
  if (!host) return;
  // Open shadow root: CSS isolation (in/out) is preserved by the shadow
  // boundary regardless of mode. We use `open` so document.getSelection()
  // surfaces user selections made inside the shadow tree — the bubble's
  // "Ask" affordance depends on it. Reuse a single root across re-renders
  // since attachShadow throws on a second call against the same host.
  let root: ShadowRoot | null = host.shadowRoot;
  if (!root) root = host.attachShadow({ mode: 'open' });
  root.innerHTML = sanitizeCardHtml(props.source);
  const style = document.createElement('style');
  style.setAttribute('data-zen-ref', '');
  style.textContent = HIGHLIGHT_STYLE;
  root.appendChild(style);
  if (props.highlights && props.highlights.length > 0) {
    wrapHighlights(root, props.highlights);
  }
}

onMounted(paint);
watch(() => [props.source, props.highlights], paint);
</script>

<template>
  <div ref="hostRef" class="html-body-host bg-paper text-paper-fg"></div>
</template>
