import { test, expect } from '@playwright/test';

const API = 'http://127.0.0.1:8080/api/v1';

test.beforeAll(async ({ request }) => {
  const r = await request.get(`${API}/groups`);
  if (r.status() >= 500) {
    throw new Error(`zen-backend at 127.0.0.1:8080 returned ${r.status()} — start it first`);
  }
});

// The span crosses a bold run AND the surrounding phrase repeats, so both
// defects the feature fixes are exercised at once: the old indexOf path could
// neither match across elements nor pick the right occurrence.
test('a message-anchored reference marks the exact dragged span', async ({ page, request }) => {
  const stamp = Date.now();

  const group = await (await request.post(`${API}/groups`, {
    data: { name: `selrange-e2e-${stamp}` },
  })).json();

  const source = await (await request.post(`${API}/cards`, {
    data: {
      title: `Source ${stamp}`,
      content: 'alpha **beta** gamma and alpha again',
      format: 'markdown',
      group_id: group.id,
    },
  })).json();

  const derived = await (await request.post(`${API}/cards`, {
    data: { title: `Derived ${stamp}`, content: 'spun off', group_id: group.id },
  })).json();

  const conv = await (await request.post(`${API}/conversations`, {
    data: { title: `selrange ${stamp}`, anchor_kind: 'card', anchor_id: source.id },
  })).json();

  // Rendered text is "alpha beta gamma and alpha again"; [6,16) is
  // "beta gamma", which straddles the <strong> boundary.
  const msg = await (await request.post(`${API}/conversations/${conv.id}/messages`, {
    data: {
      role: 'user',
      content: 'what does this mean?',
      selection_text: 'beta gamma',
      selection_start: 6,
      selection_end: 16,
      selection_seq: 1,
    },
  })).json();
  expect(msg.selection_start).toBe(6);

  // The agent passes only the message id — no excerpt, no offsets.
  const ref = await (await request.post(`${API}/references`, {
    data: {
      source_card_id: source.id,
      derived_card_id: derived.id,
      conversation_id: conv.id,
      message_id: msg.id,
    },
  })).json();
  expect(ref.selection_text).toBe('beta gamma');
  expect(ref.selection_start).toBe(6);
  expect(ref.selection_end).toBe(16);
  expect(ref.selection_seq).toBe(1);

  await page.goto(`/c/${source.id}`);
  const marks = page.locator(`mark.zen-ref[data-ref-id="${ref.id}"]`);
  await expect(marks.first()).toBeVisible();

  // Several marks share the id because the span crosses the bold boundary;
  // together they cover exactly the dragged text.
  expect(await marks.evaluateAll((ns) => ns.map((n) => n.textContent).join(''))).toBe('beta gamma');
  expect(await marks.count()).toBeGreaterThan(1);

  // "alpha" appears twice and belongs to no reference — nothing else marked.
  await expect(page.locator('mark.zen-ref')).toHaveCount(await marks.count());
});
