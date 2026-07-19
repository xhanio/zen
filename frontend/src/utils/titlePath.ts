import type { Card } from '../types/entity';

// ancestorsOf walks parent_card_id up from `card`, returning the ancestor cards
// root-first (excluding `card` itself). Stops at the first parent missing from
// `byID`, and guards against cycles — a partially-loaded store or a bad link
// yields a short chain rather than an infinite loop. Used by the card view,
// where the full ancestor chain is rendered as breadcrumb links.
export function ancestorsOf(card: Card, byID: Record<string, Card>): Card[] {
  const chain: Card[] = [];
  const seen = new Set<string>([card.id]);
  let pid = card.parent_card_id ?? null;
  while (pid && !seen.has(pid)) {
    const parent = byID[pid];
    if (!parent) break;
    chain.unshift(parent);
    seen.add(parent.id);
    pid = parent.parent_card_id ?? null;
  }
  return chain;
}

// truncateTail keeps the last `max` items and reports whether any were dropped,
// for the compact "… > parent > title" breadcrumb in search results.
export function truncateTail<T>(items: T[], max: number): { items: T[]; overflow: boolean } {
  if (items.length <= max) return { items, overflow: false };
  return { items: items.slice(items.length - max), overflow: true };
}
