import type { Card } from '../types/entity';

// A card is a "container" when it has at least one live child. Its own
// content, if any, is the preamble (document metadata) rendered above the
// sections — content is NOT part of the test. The distinction is derived,
// not stored: a card becomes a container the moment it has live children.
export function isContainer(card: Card | null | undefined, liveChildrenCount: number): boolean {
  if (!card) return false;
  return liveChildrenCount > 0;
}
