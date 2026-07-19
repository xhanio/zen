import { mount } from '@vue/test-utils';
import { describe, it, expect } from 'vitest';
import MarkdownBody from './MarkdownBody.vue';

describe('MarkdownBody', () => {
  it('renders markdown to HTML', async () => {
    const w = mount(MarkdownBody, { props: { source: '**bold**' } });
    await w.vm.$nextTick();
    expect(w.html()).toContain('<strong>bold</strong>');
  });

  it('wraps highlighted text in <mark> when highlights prop is set', async () => {
    const w = mount(MarkdownBody, {
      props: {
        source: 'The quick brown fox',
        highlights: [{ id: 'r1', text: 'quick' }],
      },
    });
    await w.vm.$nextTick();
    const marks = w.element.querySelectorAll('mark.zen-ref');
    expect(marks.length).toBe(1);
    expect((marks[0] as HTMLElement).dataset.refId).toBe('r1');
  });
});
