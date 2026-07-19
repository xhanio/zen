import { ref } from 'vue';
import { defineStore } from 'pinia';
import {
  listCards,
  getCard,
  createCard,
  updateCard,
  deleteCard,
  restoreCard,
  purgeCard,
  listChildren,
  reorderCard,
  reviewCard,
} from '../api/client';
import { BackendError, type CreateCardRequest, type UpdateCardRequest } from '../types/api';
import type { Card, ReviewGrade } from '../types/entity';
import { gradeRank } from '../utils/gradeRank';
import { useTagsStore } from './tags';

export const useCardsStore = defineStore('cards', () => {
  const byGroup = ref<Record<string, Card[]>>({});
  const byChildren = ref<Record<string, Card[]>>({});
  const byID = ref<Record<string, Card>>({});
  const loading = ref(false);
  const error = ref<BackendError | null>(null);

  const groupSeq: Record<string, number> = {};

  function index(c: Card) {
    byID.value[c.id] = c;
  }

  // Fire-and-forget refresh of the tags store so sidebar card_count stays
  // in sync with card mutations (create/update/remove/restore/purge). The
  // tags store's own load() has a re-entrancy guard, so parallel mutations
  // coalesce naturally.
  function refreshTags(): void {
    void useTagsStore().refresh();
  }

  async function loadByGroup(groupId: string) {
    const seq = (groupSeq[groupId] = (groupSeq[groupId] ?? 0) + 1);
    loading.value = true;
    error.value = null;
    try {
      const cs = await listCards({ groupId });
      if (seq !== groupSeq[groupId]) return;
      // Default list excludes soft-deleted; just keep the live cards.
      byGroup.value[groupId] = cs;
      for (const c of cs) index(c);
    } catch (e) {
      error.value = e instanceof BackendError ? e : new BackendError(0, '', String(e));
    } finally {
      loading.value = false;
    }
  }

  async function loadChildren(parentID: string, includeTrashed = false): Promise<void> {
    const cs = await listChildren(parentID, includeTrashed);
    byChildren.value[parentID] = cs;
    for (const c of cs) index(c);
  }

  async function loadOne(id: string) {
    loading.value = true;
    error.value = null;
    try {
      const c = await getCard(id);
      index(c);
    } catch (e) {
      error.value = e instanceof BackendError ? e : new BackendError(0, '', String(e));
    } finally {
      loading.value = false;
    }
  }

  async function create(req: CreateCardRequest): Promise<Card> {
    const c = await createCard(req);
    index(c);
    const list = byGroup.value[c.group_id] ?? [];
    list.push(c);
    byGroup.value[c.group_id] = list;
    groupSeq[c.group_id] = (groupSeq[c.group_id] ?? 0) + 1;
    // A new card only affects tag counts if it was created with any
    // tags. Skipping the refresh otherwise avoids an unnecessary GET
    // /tags on every card create.
    if ((req.tags?.length ?? 0) > 0) refreshTags();
    return c;
  }

  async function update(id: string, req: UpdateCardRequest): Promise<Card> {
    const c = await updateCard(id, req);
    const prev = byID.value[id];
    index(c);
    if (prev && prev.group_id !== c.group_id) {
      byGroup.value[prev.group_id] = (byGroup.value[prev.group_id] ?? []).filter((x) => x.id !== id);
      const list = byGroup.value[c.group_id] ?? [];
      list.push(c);
      byGroup.value[c.group_id] = list;
    } else {
      const list = byGroup.value[c.group_id];
      if (list) {
        const idx = list.findIndex((x) => x.id === id);
        if (idx >= 0) list[idx] = c;
      }
    }
    // Only refresh tags when the update actually touched the card's tag
    // set. Drag-and-drop / rename / level change don't affect counts.
    if (req.tags !== undefined) refreshTags();
    return c;
  }

  // remove performs a soft-delete on the backend; the card disappears from
  // byGroup but the byID entry stays (with deleted_at set) for any consumer
  // that wants it (e.g. the trash view).
  async function remove(id: string, cascade = true): Promise<void> {
    const prev = byID.value[id];
    await deleteCard(id, cascade);
    if (prev) {
      const nowIso = new Date().toISOString();
      const updated = { ...prev, deleted_at: nowIso };
      index(updated);
      byGroup.value[prev.group_id] = (byGroup.value[prev.group_id] ?? []).filter((x) => x.id !== id);
      // When cascade=true, descendants also flipped server-side. Reflect
      // locally by walking byID and marking anything with this id as an
      // ancestor as trashed. Cheap for the small card counts we hold.
      if (cascade) {
        const trashed = new Set<string>([id]);
        let grew = true;
        while (grew) {
          grew = false;
          for (const c of Object.values(byID.value)) {
            if (!c.deleted_at && c.parent_card_id && trashed.has(c.parent_card_id)) {
              const upd = { ...c, deleted_at: nowIso };
              index(upd);
              byGroup.value[c.group_id] = (byGroup.value[c.group_id] ?? []).filter((x) => x.id !== c.id);
              trashed.add(c.id);
              grew = true;
            }
          }
        }
      }
    }
    refreshTags();
  }

  async function restore(id: string): Promise<void> {
    const c = await restoreCard(id);
    byID.value[id] = c;
    const list = byGroup.value[c.group_id] ?? [];
    if (!list.some((x) => x.id === c.id)) list.push(c);
    byGroup.value[c.group_id] = list;
    refreshTags();
  }

  async function purge(id: string): Promise<void> {
    const prev = byID.value[id];
    await purgeCard(id);
    if (prev) {
      byGroup.value[prev.group_id] = (byGroup.value[prev.group_id] ?? []).filter(
        (x) => x.id !== id,
      );
    }
    delete byID.value[id];
    refreshTags();
  }

  async function reorderWithinGroup(groupId: string, fromIndex: number, toIndex: number): Promise<void> {
    const list = byGroup.value[groupId];
    if (!list) return;
    const snapshot = list.slice();
    const [moved] = list.splice(fromIndex, 1);
    list.splice(toIndex, 0, moved);
    try {
      await updateCard(moved.id, { position: toIndex });
    } catch (e) {
      byGroup.value[groupId] = snapshot;
      throw e;
    }
  }

  async function moveToGroup(id: string, groupId: string, newPosition: number): Promise<void> {
    const prev = byID.value[id];
    if (!prev) return;
    const fromList = byGroup.value[prev.group_id];
    const toList = byGroup.value[groupId];
    const fromSnap = fromList ? fromList.slice() : null;
    const toSnap = toList ? toList.slice() : null;
    if (fromList) byGroup.value[prev.group_id] = fromList.filter((x) => x.id !== id);
    if (toList) {
      const next = toList.slice();
      const insertAt = Math.max(0, Math.min(newPosition, next.length));
      next.splice(insertAt, 0, { ...prev, group_id: groupId, position: newPosition });
      byGroup.value[groupId] = next;
    }
    try {
      const c = await updateCard(id, { group_id: groupId, position: newPosition });
      index(c);
      if (byGroup.value[groupId]) {
        byGroup.value[groupId] = byGroup.value[groupId].map((x) => (x.id === id ? c : x));
      }
    } catch (e) {
      if (fromSnap) byGroup.value[prev.group_id] = fromSnap;
      if (toSnap) byGroup.value[groupId] = toSnap;
      throw e;
    }
  }

  // reorderChild moves a section card to a new index inside its parent.
  // Local byChildren[parentId] is spliced and reindexed optimistically;
  // on server error we reload children from the server and rethrow.
  async function reorderChild(id: string, newPosition: number): Promise<Card> {
    const prev = byID.value[id];
    if (!prev) throw new Error(`reorderChild: unknown card ${id}`);
    const parentId = prev.parent_card_id;
    if (!parentId) throw new Error('reorderChild: card has no parent');

    const list = (byChildren.value[parentId] ?? [])
      .slice()
      .sort((a, b) => a.position - b.position);
    const oldIdx = list.findIndex((x) => x.id === id);
    if (oldIdx < 0) throw new Error(`reorderChild: card ${id} not in parent ${parentId}`);
    const clamped = Math.max(0, Math.min(newPosition, list.length - 1));
    if (clamped === oldIdx) return prev;

    const [moved] = list.splice(oldIdx, 1);
    list.splice(clamped, 0, moved);
    list.forEach((c, i) => {
      c.position = i;
      index(c);
    });
    byChildren.value[parentId] = list;

    try {
      const c = await reorderCard(id, { position: clamped });
      index(c);
      return c;
    } catch (e) {
      await loadChildren(parentId);
      throw e;
    }
  }

  // setReviewGrade flips a card's review_grade with optimistic UI + rollback.
  // On success, reloads the parent card (or the card itself if it's top-level)
  // so review_score refreshes on the aggregating ancestor.
  async function setReviewGrade(id: string, grade: ReviewGrade): Promise<Card> {
    const prev = byID.value[id];
    if (!prev) throw new Error(`setReviewGrade: unknown card ${id}`);
    const prevGrade = prev.review_grade;
    const parentId = prev.parent_card_id;

    // Optimistic write
    prev.review_grade = grade;
    index(prev);

    try {
      const c = await reviewCard(id, { grade });
      // Merge server response into the existing object so both byID and
      // byChildren keep pointing to the same instance. Replacing via
      // index(c) would leave byChildren stale, breaking subsequent clicks.
      Object.assign(prev, c);
      // Refresh the aggregating card so its review_score updates. Top-level
      // card refreshes itself; nested card refreshes its parent chain.
      if (parentId) {
        await loadOne(parentId);
      } else {
        await loadOne(id);
      }
      return prev;
    } catch (e) {
      // Roll back local state and reload authoritatively.
      prev.review_grade = prevGrade;
      index(prev);
      if (parentId) {
        await loadChildren(parentId);
      } else {
        await loadOne(id);
      }
      throw e;
    }
  }

  // escalateReviewGrade raises a card's grade toward `target`, but never lowers
  // it — auto-transitions (expand → DIGESTED, ask → GRILLED) only escalate.
  // No-ops for an unknown card or when the card is already at/above target.
  async function escalateReviewGrade(id: string, target: ReviewGrade): Promise<void> {
    const card = byID.value[id];
    if (!card) return;
    if (gradeRank(card.review_grade) >= gradeRank(target)) return;
    await setReviewGrade(id, target);
  }

  return {
    byGroup,
    byChildren,
    byID,
    loading,
    error,
    loadByGroup,
    loadChildren,
    loadOne,
    create,
    update,
    remove,
    restore,
    purge,
    reorderWithinGroup,
    moveToGroup,
    reorderChild,
    setReviewGrade,
    escalateReviewGrade,
  };
});
