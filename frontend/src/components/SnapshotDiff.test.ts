import { describe, it, expect } from 'vitest';
import { mount } from '@vue/test-utils';
import SnapshotDiff from './SnapshotDiff.vue';
import type { CardDiff } from '../types/entity';

const diff: CardDiff = {
  fields: [{ key: 'title', spans: [{ op: 'eq', text: 'A ' }, { op: 'del', text: 'old' }, { op: 'add', text: 'new' }] }],
  lines: [
    { op: 'ctx', text: 'unchanged' },
    { op: 'del', text: '版本行只记对话', spans: [{ op: 'del', text: '版本' }, { op: 'eq', text: '行只记对话' }] },
    { op: 'add', text: '快照行只记对话', spans: [{ op: 'add', text: '快照' }, { op: 'eq', text: '行只记对话' }] },
  ],
};

describe('SnapshotDiff', () => {
  it('renders context, deleted, and added lines distinctly', () => {
    const w = mount(SnapshotDiff, { props: { diff, truncated: false, before: '', after: '' } });
    expect(w.findAll('[data-test="diff-line-ctx"]')).toHaveLength(1);
    expect(w.findAll('[data-test="diff-line-del"]')).toHaveLength(1);
    expect(w.findAll('[data-test="diff-line-add"]')).toHaveLength(1);
  });

  // The payoff of rune-level diffing: only the changed characters light up.
  it('highlights only the changed spans inside a line', () => {
    const w = mount(SnapshotDiff, { props: { diff, truncated: false, before: '', after: '' } });
    const marks = w.findAll('mark').map((m) => m.text());
    expect(marks).toContain('版本');
    expect(marks).toContain('快照');
    expect(marks).not.toContain('行只记对话');
  });

  it('renders a field row for the title change', () => {
    const w = mount(SnapshotDiff, { props: { diff, truncated: false, before: '', after: '' } });
    expect(w.find('[data-test="diff-field-title"]').exists()).toBe(true);
    expect(w.find('[data-test="diff-field-title"]').text()).toContain('old');
    expect(w.find('[data-test="diff-field-title"]').text()).toContain('new');
  });

  it('falls back to both bodies when the diff was truncated', () => {
    const w = mount(SnapshotDiff, {
      props: { diff: { fields: [], lines: [] }, truncated: true, before: 'OLD BODY', after: 'NEW BODY' },
    });
    expect(w.find('[data-test="diff-truncated"]').exists()).toBe(true);
    expect(w.text()).toContain('OLD BODY');
    expect(w.text()).toContain('NEW BODY');
  });

  // A line inserted with no counterpart has no spans; it must still render.
  it('renders an unpaired line without spans', () => {
    const w = mount(SnapshotDiff, {
      props: {
        diff: { fields: [], lines: [{ op: 'add', text: 'brand new line' }] },
        truncated: false, before: '', after: '',
      },
    });
    expect(w.find('[data-test="diff-line-add"]').text()).toContain('brand new line');
  });
});
