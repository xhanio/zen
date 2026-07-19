import type { Card } from '../types/entity';

// Card ↔ document helpers — the single source of truth for "what is a
// document". A document is a live top-level card that has at least one live
// child; it spans multiple levels, so it is the multi-section unit of a group.

// Live (non-deleted) children of a card within the given list.
export function childrenOf(cards: Card[], parentId: string): Card[] {
  return cards.filter((c) => c.parent_card_id === parentId && !c.deleted_at);
}

// True when `card` is a live top-level card with at least one live child.
export function isDocument(cards: Card[], card: Card): boolean {
  return !card.deleted_at && !card.parent_card_id && childrenOf(cards, card.id).length > 0;
}

// Every document in the list, newest `updated_at` first.
export function documentsIn(cards: Card[]): Card[] {
  return cards
    .filter((c) => isDocument(cards, c))
    .slice()
    .sort((a, b) => (a.updated_at < b.updated_at ? 1 : -1));
}
