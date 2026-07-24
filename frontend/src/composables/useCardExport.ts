import { ref } from 'vue';
import { getCard, listChildren } from '../api/client';
import { BackendError } from '../types/api';
import type { Card, EntityFormat } from '../types/entity';
import {
  serializeCard,
  filenameFor,
  downloadText,
  EXPORT_FORMATS,
  type ExportNode,
} from '../utils/exportCard';

// Module-level singletons: only one export runs per tab at a time, and the
// shared binding lets tests spy on the same `exportCard` the button calls
// (mirrors composables/useChatSidebar.ts).
const exporting = ref(false);
const error = ref<string | null>(null);

// Recursively fetch a card's LIVE descendants (trashed excluded) into an
// ExportNode tree. Every node is fetched (a leaf's listChildren returns []),
// siblings concurrently; children are position-sorted to match the reader.
async function buildNode(card: Card): Promise<ExportNode> {
  const children = await listChildren(card.id, false);
  const ordered = children.slice().sort((a, b) => a.position - b.position);
  const childNodes = await Promise.all(ordered.map(buildNode));
  return { card, children: childNodes };
}

export const actions = {
  async exportCard(rootId: string): Promise<void> {
    if (exporting.value) return; // re-entrancy guard
    exporting.value = true;
    error.value = null;
    try {
      const root = await getCard(rootId);
      const tree = await buildNode(root);
      const format: EntityFormat = root.format ?? 'markdown';
      const mime = (EXPORT_FORMATS[format] ?? EXPORT_FORMATS.markdown).mime;
      downloadText(filenameFor(root.title, format), mime, serializeCard(tree, format));
    } catch (e) {
      error.value = e instanceof BackendError ? e.message : 'Export failed.';
    } finally {
      exporting.value = false;
    }
  },
};

export function useCardExport() {
  return { exporting, error, exportCard: actions.exportCard };
}
