import type { Highlight } from './highlightText';
import type { Card, CardSelection } from '../types/entity';

// The one place a card's paintable spans are assembled, shared by the leaf view
// and the container view so the two cannot drift on ordering or on which kind
// demands a range.
//
// References come first so message underlines nest INSIDE reference spans
// rather than splitting them. Safe because renderedText traverses into existing
// marks, leaving offsets valid across both passes.
export function buildCardHighlights(
  card: Card | undefined,
  selections: CardSelection[] | undefined,
): Highlight[] {
  return [
    ...(card?.references ?? []).map((r) => ({
      id: r.id,
      text: r.selection_text,
      start: r.selection_start,
      end: r.selection_end,
      kind: 'reference' as const,
    })),
    ...(selections ?? []).map((s) => ({
      id: s.message_id,
      text: s.selection_text,
      start: s.selection_start,
      end: s.selection_end,
      kind: 'message' as const,
      requireRange: true,
    })),
  ];
}
