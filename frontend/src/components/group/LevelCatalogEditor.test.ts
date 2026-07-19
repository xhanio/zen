import { describe, it, expect } from 'vitest';
import { mount } from '@vue/test-utils';
import LevelCatalogEditor from './LevelCatalogEditor.vue';

describe('LevelCatalogEditor', () => {
  it('preserves id on existing rows during a weight edit', async () => {
    const wrapper = mount(LevelCatalogEditor, {
      props: {
        modelValue: [
          { id: 'ID1', weight: 0, name: '原则' },
          { id: 'ID2', weight: 2, name: '决策' },
        ],
      },
    });
    const input = wrapper.findAll('input[type="number"]')[0];
    await input.setValue(1);
    const emitted = wrapper.emitted('update:modelValue')?.at(-1)?.[0] as Array<{
      id: string; weight: number; name: string;
    }>;
    // Sorted by weight — first row (weight now 1) is our edited entry.
    expect(emitted[0].id).toBe('ID1');
    expect(emitted[0].weight).toBe(1);
    expect(emitted[0].name).toBe('原则');
    expect(emitted[1].id).toBe('ID2');
  });

  it('preserves id on existing rows during a name edit', async () => {
    const wrapper = mount(LevelCatalogEditor, {
      props: {
        modelValue: [{ id: 'ID1', weight: 0, name: '原则' }],
      },
    });
    const input = wrapper.findAll('input[type="text"]')[0];
    await input.setValue('renamed');
    const emitted = wrapper.emitted('update:modelValue')?.at(-1)?.[0] as Array<{
      id: string; weight: number; name: string;
    }>;
    expect(emitted[0].id).toBe('ID1');
    expect(emitted[0].name).toBe('renamed');
    expect(emitted[0].weight).toBe(0);
  });

  it('new row emits with empty id so the server can assign one', async () => {
    const wrapper = mount(LevelCatalogEditor, {
      props: { modelValue: [{ id: 'ID1', weight: 0, name: '原则' }] },
    });
    await wrapper.find('[data-test="add-entry"]').trigger('click');
    const emitted = wrapper.emitted('update:modelValue')?.at(-1)?.[0] as Array<{
      id: string; weight: number; name: string;
    }>;
    expect(emitted.at(-1)!.id).toBe('');
    // Existing entry keeps its id.
    expect(emitted[0].id).toBe('ID1');
  });

  it('flags duplicate names in the editor', () => {
    const wrapper = mount(LevelCatalogEditor, {
      props: {
        modelValue: [
          { id: 'ID1', weight: 0, name: 'x' },
          { id: 'ID2', weight: 1, name: 'x' },
        ],
      },
    });
    // Both inputs marked with the destructive border.
    const inputs = wrapper.findAll('input[type="text"]');
    const flagged = inputs.filter((i) => i.classes().some((c) => c.includes('destructive')));
    expect(flagged.length).toBe(2);
  });

  it('renders a gradient dot per row', () => {
    const wrapper = mount(LevelCatalogEditor, {
      props: { modelValue: [
        { id: 'ID1', weight: 0, name: '原则' },
        { id: 'ID2', weight: 1, name: '决策' },
      ] },
    });
    const dots = wrapper.findAll('li > span');
    expect(dots.length).toBe(2);
    expect(dots[0].attributes('style')).toContain('background');
  });
});
