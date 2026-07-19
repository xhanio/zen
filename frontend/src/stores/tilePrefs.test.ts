import { describe, it, expect, beforeEach } from 'vitest';
import { setActivePinia, createPinia } from 'pinia';
import { useTilePrefsStore } from './tilePrefs';

describe('tilePrefs.showTrashedSections', () => {
  beforeEach(() => {
    localStorage.clear();
    setActivePinia(createPinia());
  });

  it('defaults to false', () => {
    const s = useTilePrefsStore();
    expect(s.showTrashedSections).toBe(false);
  });

  it('toggles and persists', () => {
    const s = useTilePrefsStore();
    s.toggleShowTrashedSections();
    expect(s.showTrashedSections).toBe(true);
    // A fresh pinia instance re-loads from localStorage:
    setActivePinia(createPinia());
    const s2 = useTilePrefsStore();
    expect(s2.showTrashedSections).toBe(true);
  });

  it('coexists with hideSummaries', () => {
    const s = useTilePrefsStore();
    s.toggleSummaries();
    s.toggleShowTrashedSections();
    expect(s.hideSummaries).toBe(true);
    expect(s.showTrashedSections).toBe(true);
  });
});

describe('tilePrefs.hideSections', () => {
  beforeEach(() => {
    localStorage.clear();
    setActivePinia(createPinia());
  });

  it('defaults to true', () => {
    const s = useTilePrefsStore();
    expect(s.hideSections).toBe(true);
  });

  it('toggles and persists', () => {
    const s = useTilePrefsStore();
    s.toggleSections();
    expect(s.hideSections).toBe(false);
    setActivePinia(createPinia());
    const s2 = useTilePrefsStore();
    expect(s2.hideSections).toBe(false);
  });
});

describe('tilePrefs.documentsWidth', () => {
  beforeEach(() => {
    localStorage.clear();
    setActivePinia(createPinia());
  });

  it('defaults to 240', () => {
    expect(useTilePrefsStore().documentsWidth).toBe(240);
  });

  it('clamps and persists on set', () => {
    const s = useTilePrefsStore();
    s.setDocumentsWidth(9999);
    expect(s.documentsWidth).toBe(560);
    s.setDocumentsWidth(10);
    expect(s.documentsWidth).toBe(160);
    setActivePinia(createPinia());
    expect(useTilePrefsStore().documentsWidth).toBe(160);
  });

  it('resets to the default', () => {
    const s = useTilePrefsStore();
    s.setDocumentsWidth(400);
    s.resetDocumentsWidth();
    expect(s.documentsWidth).toBe(240);
  });

  it('falls back to the default on a non-numeric stored value', () => {
    localStorage.setItem('zen:tilePrefs', '{"documentsWidth":"not-a-number"}');
    setActivePinia(createPinia());
    expect(useTilePrefsStore().documentsWidth).toBe(240);
  });
});

describe('tilePrefs.chatListWidth', () => {
  beforeEach(() => {
    localStorage.clear();
    setActivePinia(createPinia());
  });

  it('defaults to 256', () => {
    expect(useTilePrefsStore().chatListWidth).toBe(256);
  });

  it('clamps and persists on set', () => {
    const s = useTilePrefsStore();
    s.setChatListWidth(9999);
    expect(s.chatListWidth).toBe(480);
    s.setChatListWidth(10);
    expect(s.chatListWidth).toBe(180);
    setActivePinia(createPinia());
    expect(useTilePrefsStore().chatListWidth).toBe(180);
  });

  it('resets to the default', () => {
    const s = useTilePrefsStore();
    s.setChatListWidth(400);
    s.resetChatListWidth();
    expect(s.chatListWidth).toBe(256);
  });

  it('falls back to the default on a non-numeric stored value', () => {
    localStorage.setItem('zen:tilePrefs', '{"chatListWidth":"nope"}');
    setActivePinia(createPinia());
    expect(useTilePrefsStore().chatListWidth).toBe(256);
  });
});
