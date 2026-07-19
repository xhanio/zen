import { describe, it, expect, beforeEach, vi } from 'vitest';
import { setActivePinia, createPinia } from 'pinia';
import { useGroupsStore } from './groups';
import * as client from '../api/client';
import type { Group } from '../types/entity';

const G: Group = { id: 'G1', name: 'Design', rule: '', position: 0, level_catalog: [], created_at: '', updated_at: '' };

beforeEach(() => setActivePinia(createPinia()));

describe('groupsStore.update', () => {
  it('calls updateGroup with the patch and replaces the group in the list', async () => {
    const store = useGroupsStore();
    store.groups = [{ ...G }];
    const updated: Group = { ...G, name: 'Renamed', rule: 'r' };
    const spy = vi.spyOn(client, 'updateGroup').mockResolvedValue(updated);

    const patch = { name: 'Renamed', level_catalog: [], rule: 'r' };
    const result = await store.update('G1', patch);

    expect(spy).toHaveBeenCalledWith('G1', patch);
    expect(result).toEqual(updated);
    expect(store.groups[0].name).toBe('Renamed');
  });
});
