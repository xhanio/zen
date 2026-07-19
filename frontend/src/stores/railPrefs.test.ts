import { describe, it, expect, beforeEach } from 'vitest';
import { setActivePinia, createPinia } from 'pinia';
import { useRailPrefsStore } from './railPrefs';

beforeEach(() => {
  setActivePinia(createPinia());
  localStorage.clear();
});

describe('railPrefs', () => {
  it('defaults to 256px', () => {
    expect(useRailPrefsStore().width).toBe(256);
  });

  it('clamps width to [200, 480] and persists', () => {
    const s = useRailPrefsStore();
    s.setWidth(50);
    expect(s.width).toBe(200);
    s.setWidth(9999);
    expect(s.width).toBe(480);
    s.setWidth(320);
    expect(s.width).toBe(320);
    expect(localStorage.getItem('zen:railWidth')).toBe('320');
  });

  it('reset returns to the default width', () => {
    const s = useRailPrefsStore();
    s.setWidth(400);
    s.reset();
    expect(s.width).toBe(256);
  });

  it('loads a persisted width on init', () => {
    localStorage.setItem('zen:railWidth', '300');
    setActivePinia(createPinia());
    expect(useRailPrefsStore().width).toBe(300);
  });
});
