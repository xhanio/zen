import { mount } from '@vue/test-utils';
import { describe, it, expect } from 'vitest';
import HtmlBody from './HtmlBody.vue';

describe('HtmlBody', () => {
  it('mounts content in a shadow root so styles do not leak', async () => {
    const wrapper = mount(HtmlBody, {
      props: { source: '<style>p{background:red}</style><p>hello</p>' },
    });
    await wrapper.vm.$nextTick();
    const host = wrapper.find('div.html-body-host').element as HTMLElement;
    const root = host.shadowRoot;
    expect(root).toBeTruthy();
    expect(root!.innerHTML).toContain('<p>hello</p>');
    expect(root!.innerHTML).toContain('background:red');
    // Outer doc has no <style> tag for this card.
    expect(wrapper.html()).not.toContain('background:red');
  });

  it('strips scripts before mounting', async () => {
    const wrapper = mount(HtmlBody, {
      props: { source: '<p>visible</p><script>(window as any).zenPwned=1</script>' },
    });
    await wrapper.vm.$nextTick();
    const host = wrapper.find('div.html-body-host').element as HTMLElement;
    const root = host.shadowRoot;
    expect(root!.innerHTML).toContain('visible');
    expect(root!.innerHTML).not.toContain('<script');
    expect((window as unknown as { zenPwned?: number }).zenPwned).toBeUndefined();
  });

  it('wraps highlighted text in <mark> inside the shadow root', async () => {
    const w = mount(HtmlBody, {
      props: {
        source: '<p>hello world</p>',
        highlights: [{ id: 'r1', text: 'hello' }],
      },
    });
    await w.vm.$nextTick();
    const host = w.find('div.html-body-host').element as HTMLElement;
    const marks = host.shadowRoot!.querySelectorAll('mark.zen-ref');
    expect(marks.length).toBe(1);
    expect((marks[0] as HTMLElement).dataset.refId).toBe('r1');
  });
});
