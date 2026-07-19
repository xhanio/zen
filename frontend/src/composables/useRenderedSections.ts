import { computed, type ComputedRef, type Ref } from 'vue';
import { storeToRefs } from 'pinia';
import { useCardsStore } from '../stores/cards';
import { useTilePrefsStore } from '../stores/tilePrefs';
import { useContainerFilterStore } from '../stores/containerFilter';
import type { Card } from '../types/entity';

/**
 * The sections actually on the page, in the order they appear.
 *
 * The gutter draws one dot per section and the pill targets one of them, but
 * the gutter renders outside ContainerBody. Both must agree, exactly, about
 * which sections are displayed — a dot for a hidden section parks at the wrong
 * offset, and the pill would point at a card the reader cannot see.
 */
export function useRenderedSections(parentId: Ref<string>): ComputedRef<Card[]> {
  const cardsStore = useCardsStore();
  const { byChildren } = storeToRefs(cardsStore);
  const { showTrashedSections } = storeToRefs(useTilePrefsStore());
  const containerFilter = useContainerFilterStore();

  return computed(() => {
    const all = (byChildren.value[parentId.value] ?? []).slice()
      .sort((a, b) => a.position - b.position);
    const trashFiltered = showTrashedSections.value ? all : all.filter((c) => !c.deleted_at);
    return trashFiltered.filter((c) => !containerFilter.isCollapsed(c.id));
  });
}
