import { mount } from '@vue/test-utils';
import { describe, it, expect } from 'vitest';
import TextBody from './TextBody.vue';

describe('TextBody', () => {
  it('renders content verbatim with whitespace preserved', async () => {
    const w = mount(TextBody, { props: { source: '**not bold**\n  line 2' } });
    await w.vm.$nextTick();
    const pre = w.find('pre');
    expect(pre.exists()).toBe(true);
    expect(pre.element.textContent).toBe('**not bold**\n  line 2');
  });

  it('does not interpret HTML', async () => {
    const w = mount(TextBody, { props: { source: '<script>x</script>**bold**' } });
    await w.vm.$nextTick();
    const pre = w.find('pre');
    expect(pre.element.textContent).toBe('<script>x</script>**bold**');
  });

  it('wraps highlighted text in <mark> when highlights prop is set', async () => {
    const w = mount(TextBody, {
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
