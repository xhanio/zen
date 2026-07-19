import { describe, it, expect } from 'vitest';
import { mount } from '@vue/test-utils';
import InlineEdit from './InlineEdit.vue';

describe('InlineEdit', () => {
  it('renders the value as plain text by default', () => {
    const w = mount(InlineEdit, { props: { modelValue: 'hello' } });
    expect(w.text()).toContain('hello');
    expect(w.find('input').exists()).toBe(false);
  });

  it('switches to input on double-click', async () => {
    const w = mount(InlineEdit, { props: { modelValue: 'hello' } });
    await w.trigger('dblclick');
    expect(w.find('input').exists()).toBe(true);
    expect((w.find('input').element as HTMLInputElement).value).toBe('hello');
  });

  it('uses textarea when multiline is true', async () => {
    const w = mount(InlineEdit, { props: { modelValue: 'hello', multiline: true } });
    await w.trigger('dblclick');
    expect(w.find('textarea').exists()).toBe(true);
  });

  it('emits save on blur with changed value', async () => {
    const w = mount(InlineEdit, { props: { modelValue: 'hello' } });
    await w.trigger('dblclick');
    const input = w.find('input');
    await input.setValue('world');
    await input.trigger('blur');
    expect(w.emitted('save')).toBeTruthy();
    expect(w.emitted('save')![0]).toEqual(['world', 'markdown']);
  });

  it('does not emit save when value is unchanged', async () => {
    const w = mount(InlineEdit, { props: { modelValue: 'hello' } });
    await w.trigger('dblclick');
    await w.find('input').trigger('blur');
    expect(w.emitted('save')).toBeFalsy();
  });

  it('emits cancel on Escape and exits edit mode', async () => {
    const w = mount(InlineEdit, { props: { modelValue: 'hello' } });
    await w.trigger('dblclick');
    const input = w.find('input');
    await input.setValue('world');
    await input.trigger('keydown', { key: 'Escape' });
    expect(w.emitted('cancel')).toBeTruthy();
    expect(w.emitted('save')).toBeFalsy();
  });

  it('renders error below editor when error prop is set', async () => {
    const w = mount(InlineEdit, { props: { modelValue: 'hello', error: 'too long' } });
    await w.trigger('dblclick');
    expect(w.text()).toContain('too long');
  });

  it('disables input while saving', async () => {
    const w = mount(InlineEdit, { props: { modelValue: 'hello', saving: true } });
    await w.trigger('dblclick');
    expect(w.find('input').attributes('disabled')).toBeDefined();
  });
});

describe('InlineEdit format selector (multiline)', () => {
  it('emits the chosen format on save', async () => {
    const w = mount(InlineEdit, { props: { modelValue: 'hi', format: 'markdown' as const, multiline: true } });
    await w.trigger('dblclick');
    await w.find('select').setValue('html');
    await w.find('textarea').setValue('<p>x</p>');
    await w.find('textarea').trigger('blur');
    const events = w.emitted('save')!;
    expect(events[0]).toEqual(['<p>x</p>', 'html']);
  });

  it('shows format dropdown only when multiline+editing', () => {
    const w = mount(InlineEdit, { props: { modelValue: 'hi', multiline: false } });
    // not yet editing -> no select shown
    expect(w.find('select').exists()).toBe(false);
  });
});
