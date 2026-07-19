import { describe, it, expect } from 'vitest';
import { mount } from '@vue/test-utils';
import TagChipEditor from './TagChipEditor.vue';

describe('TagChipEditor', () => {
  it('renders existing tags as chips', () => {
    const w = mount(TagChipEditor, {
      props: { tags: ['alpha', 'beta'], allTags: ['alpha', 'beta', 'gamma'] },
    });
    expect(w.text()).toContain('alpha');
    expect(w.text()).toContain('beta');
  });

  it('emits update with tag removed when × is clicked', async () => {
    const w = mount(TagChipEditor, {
      props: { tags: ['alpha', 'beta'], allTags: [] },
    });
    const buttons = w.findAll('button').filter((b) => b.text() === '×');
    await buttons[0].trigger('click');
    expect(w.emitted('update')).toBeTruthy();
    expect(w.emitted('update')![0][0]).toEqual(['beta']);
  });

  it('emits update with new tag on Enter', async () => {
    const w = mount(TagChipEditor, {
      props: { tags: ['alpha'], allTags: [] },
    });
    const input = w.find('input');
    await input.setValue('beta');
    await input.trigger('keydown', { key: 'Enter' });
    expect(w.emitted('update')).toBeTruthy();
    expect(w.emitted('update')![0][0]).toEqual(['alpha', 'beta']);
  });

  it('does not emit on Enter when input is empty', async () => {
    const w = mount(TagChipEditor, { props: { tags: [], allTags: [] } });
    const input = w.find('input');
    await input.trigger('keydown', { key: 'Enter' });
    expect(w.emitted('update')).toBeFalsy();
  });

  it('does not duplicate existing tags', async () => {
    const w = mount(TagChipEditor, { props: { tags: ['alpha'], allTags: [] } });
    const input = w.find('input');
    await input.setValue('alpha');
    await input.trigger('keydown', { key: 'Enter' });
    expect(w.emitted('update')).toBeFalsy();
  });

  it('removes last chip on Backspace when input is empty', async () => {
    const w = mount(TagChipEditor, { props: { tags: ['alpha', 'beta'], allTags: [] } });
    const input = w.find('input');
    await input.trigger('keydown', { key: 'Backspace' });
    expect(w.emitted('update')).toBeTruthy();
    expect(w.emitted('update')![0][0]).toEqual(['alpha']);
  });

  it('lowercases the new tag', async () => {
    const w = mount(TagChipEditor, { props: { tags: [], allTags: [] } });
    const input = w.find('input');
    await input.setValue('MixedCase');
    await input.trigger('keydown', { key: 'Enter' });
    expect(w.emitted('update')![0][0]).toEqual(['mixedcase']);
  });

  it('renders error message when error prop is set', () => {
    const w = mount(TagChipEditor, {
      props: { tags: [], allTags: [], error: 'too long' },
    });
    expect(w.text()).toContain('too long');
  });

  it('disables input while saving', () => {
    const w = mount(TagChipEditor, {
      props: { tags: [], allTags: [], saving: true },
    });
    expect(w.find('input').attributes('disabled')).toBeDefined();
  });

  it('hides the input and the × buttons when readonly', () => {
    const w = mount(TagChipEditor, {
      props: { tags: ['alpha', 'beta'], allTags: [], readonly: true },
    });
    expect(w.text()).toContain('alpha');
    expect(w.text()).toContain('beta');
    expect(w.find('input[aria-label="Add tag"]').exists()).toBe(false);
    expect(w.findAll('button').filter((b) => b.text() === '×')).toHaveLength(0);
  });
});
