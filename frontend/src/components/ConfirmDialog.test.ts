import { describe, it, expect, afterEach } from 'vitest';
import { mount, flushPromises } from '@vue/test-utils';
import ConfirmDialog from './ConfirmDialog.vue';

afterEach(() => {
  document.body.innerHTML = '';
});

describe('ConfirmDialog', () => {
  it('renders title and description when open', async () => {
    const w = mount(ConfirmDialog, {
      props: { open: true, title: 'Delete?', description: 'Cannot undo.' },
      attachTo: document.body,
    });
    await flushPromises();
    expect(document.body.textContent).toContain('Delete?');
    expect(document.body.textContent).toContain('Cannot undo.');
    w.unmount();
  });

  it('renders nothing when closed', async () => {
    const w = mount(ConfirmDialog, {
      props: { open: false, title: 'Delete?', description: 'Cannot undo.' },
      attachTo: document.body,
    });
    await flushPromises();
    expect(document.body.textContent || '').not.toContain('Delete?');
    w.unmount();
  });

  it('emits confirm when action button is clicked', async () => {
    const w = mount(ConfirmDialog, {
      props: { open: true, title: 'Delete?', description: '', confirmLabel: 'Yes' },
      attachTo: document.body,
    });
    await flushPromises();
    const buttons = Array.from(document.querySelectorAll('button')) as HTMLButtonElement[];
    const confirmBtn = buttons.find((b) => b.textContent?.trim() === 'Yes');
    expect(confirmBtn).toBeTruthy();
    confirmBtn!.click();
    await flushPromises();
    expect(w.emitted('confirm')).toBeTruthy();
    w.unmount();
  });

  it('applies destructive styling when destructive=true', async () => {
    const w = mount(ConfirmDialog, {
      props: { open: true, title: 'Delete?', description: '', destructive: true, confirmLabel: 'X' },
      attachTo: document.body,
    });
    await flushPromises();
    const buttons = Array.from(document.querySelectorAll('button')) as HTMLButtonElement[];
    const confirmBtn = buttons.find((b) => b.textContent?.trim() === 'X');
    expect(confirmBtn).toBeTruthy();
    expect(confirmBtn!.className).toMatch(/destructive/);
    w.unmount();
  });
});
