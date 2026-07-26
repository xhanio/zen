import { test, expect } from '@playwright/test';

const API = 'http://127.0.0.1:8080/api/v1';

test.beforeAll(async ({ request }) => {
  const r = await request.get(`${API}/groups`);
  if (r.status() >= 500) {
    throw new Error(`zen-backend at 127.0.0.1:8080 returned ${r.status()} — start it first`);
  }
});

// Mirrors the real flow: a conversation exists, an agent-style API call edits a
// card citing that conversation, and the record must appear in the thread,
// open a diff, and toggle back.
test('a conversation-attributed edit shows as a record and opens a diff', async ({ page, request }) => {
  const stamp = Date.now();

  const group = await (await request.post(`${API}/groups`, {
    data: { name: `snapshot-e2e-${stamp}` },
  })).json();

  const card = await (await request.post(`${API}/cards`, {
    data: { title: `Snapshot target ${stamp}`, content: 'first body', group_id: group.id },
  })).json();

  const conv = await (await request.post(`${API}/conversations`, {
    data: { title: `snapshot thread ${stamp}`, anchor_kind: 'card', anchor_id: card.id },
  })).json();

  // The edit an agent would make: actor header plus the causing conversation.
  const updated = await request.put(`${API}/cards/${card.id}`, {
    headers: { 'X-Zen-Actor': 'agent' },
    data: { content: 'second body', conversation_id: conv.id },
  });
  expect(updated.ok()).toBeTruthy();

  // The record is filed under the conversation, not the card.
  const listed = await (await request.get(`${API}/snapshots?conversation_id=${conv.id}`)).json();
  expect(listed.snapshots).toHaveLength(1);
  expect(listed.snapshots[0].actor).toBe('agent');
  expect(listed.snapshots[0].seq).toBe(2);

  // The card's own chain also holds the baseline written at create time.
  const chain = await (await request.get(`${API}/snapshots?card_id=${card.id}`)).json();
  expect(chain.snapshots).toHaveLength(2);
  expect(chain.snapshots[1].change_kind).toBe('create');

  // The detail carries the previous body and a diff to render.
  const detail = await (await request.get(`${API}/snapshots/${listed.snapshots[0].id}`)).json();
  expect(detail.previous.content).toBe('first body');
  expect(detail.snapshot.diff).not.toBe('');

  // Change view: the card body is replaced by the diff, and leaving it
  // restores the live card.
  await page.goto(`/c/${card.id}?snapshot=${listed.snapshots[0].id}`);
  await expect(page.locator('[data-test="snapshot-banner"]')).toBeVisible();
  await expect(page.locator('[data-test="diff-line-add"]').first()).toContainText('second body');

  await page.locator('[data-test="snapshot-exit"]').click();
  await expect(page.locator('[data-test="snapshot-banner"]')).toHaveCount(0);
  await expect(page.locator('[data-test="diff-line-add"]')).toHaveCount(0);

  // The fallback entry is a chip in the title row, not a standing panel: it
  // costs nothing until opened, and reaches the same records from the card.
  await expect(page.locator('[data-test="snapshot-popover"]')).toHaveCount(0);
  await page.locator('[data-test="snapshot-chip"]').click();
  await expect(page.locator('[data-test="snapshot-list-row"]')).toHaveCount(2);
  await page.locator('[data-test="snapshot-list-row"]').first().click();
  await expect(page.locator('[data-test="snapshot-popover"]')).toHaveCount(0);
  await expect(page.locator('[data-test="snapshot-banner"]')).toBeVisible();
});
