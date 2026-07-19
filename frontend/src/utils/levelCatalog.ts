import type { LevelEntry } from '../types/entity';

export function sortCatalog(entries: LevelEntry[]): LevelEntry[] {
  return [...entries].sort((a, b) => a.weight - b.weight);
}

export function findByWeight(entries: LevelEntry[], weight: number): LevelEntry | undefined {
  return entries.find((e) => e.weight === weight);
}

export function findByName(entries: LevelEntry[], name: string): LevelEntry | undefined {
  return entries.find((e) => e.name === name);
}

export function findById(entries: LevelEntry[], id: string): LevelEntry | undefined {
  return entries.find((e) => e.id === id);
}

// Card ↔ catalog lookups. Every place that needs to render a card's
// level color, level name, or level weight should go through these —
// scattered inline `catalog.find((e) => e.id === card.level_entry_id)`
// calls drift over time.

export function entryForCard(
  catalog: LevelEntry[],
  card: { level_entry_id: string | null },
): LevelEntry | undefined {
  if (!card.level_entry_id) return undefined;
  return findById(catalog, card.level_entry_id);
}

export function weightForCard(
  catalog: LevelEntry[],
  card: { level_entry_id: string | null },
): number | null {
  return entryForCard(catalog, card)?.weight ?? null;
}
