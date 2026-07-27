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

// The capture half, in a real browser. The test above injects offsets over the
// API, so it never exercised a drag — and html-format cards render into a
// shadow root, where Chrome retargets the document selection to the host. That
// made every html card record no range at all.
test('a drag inside an html card body records the offsets', async ({ page, request }) => {
  const stamp = Date.now();

  const group = await (await request.post(`${API}/groups`, {
    data: { name: `selrange-html-e2e-${stamp}` },
  })).json();

  const source = await (await request.post(`${API}/cards`, {
    data: {
      title: `Html source ${stamp}`,
      content: '<p>Hello <strong>brave</strong> world</p>',
      format: 'html',
      group_id: group.id,
    },
  })).json();

  await page.goto(`/c/${source.id}`);
  await expect(page.locator('.html-body-host')).toBeAttached();

  // It must be a real mouse drag. setBaseAndExtent does NOT reproduce this:
  // only a genuine drag makes Chrome retarget the document selection out of
  // the shadow tree, which is the whole condition under test.
  const box = await page.evaluate((cardId) => {
    const host = document
      .querySelector(`[data-card-id="${cardId}"]`)!
      .querySelector('.html-body-host') as HTMLElement;
    const r = host.shadowRoot!.querySelector('p')!.getBoundingClientRect();
    return { x: r.x, y: r.y, w: r.width, h: r.height };
  }, source.id);

  await page.mouse.move(box.x + 2, box.y + box.h / 2);
  await page.mouse.down();
  await page.mouse.move(box.x + box.w - 2, box.y + box.h / 2, { steps: 15 });
  await page.mouse.up();

  // Ground truth from the user's side: whatever Chrome says is selected. The
  // stored offsets must slice the body's rendered text to exactly this, which
  // holds regardless of where the drag landed pixel-wise.
  const selected = (await page.evaluate(() => window.getSelection()!.toString().trim()))!;
  expect(selected.length).toBeGreaterThan(0);

  const ask = page.getByRole('button', { name: 'Ask' });
  await expect(ask).toBeVisible(); // proves the bubble saw the shadow selection
  await ask.click();

  await page.locator('[data-test="composer-input"]').fill('what is this?');
  await page.locator('[data-test="composer-send"]').click();

  const stored = await expect.poll(async () => {
    const convs = await (await request.get(
      `${API}/conversations?anchor_kind=card&anchor_id=${source.id}`,
    )).json();
    const conv = (convs.conversations ?? [])[0];
    if (!conv) return null;
    const msgs = await (await request.get(`${API}/conversations/${conv.id}/messages`)).json();
    const user = (msgs.messages ?? []).find((m: { role: string }) => m.role === 'user');
    if (!user || user.selection_start == null) return null;
    return { start: user.selection_start, end: user.selection_end, text: user.selection_text };
  }, { timeout: 10_000 }).not.toBeNull();

  const msgs = await (await request.get(
    `${API}/conversations?anchor_kind=card&anchor_id=${source.id}`,
  )).json();
  const conv = msgs.conversations[0];
  const all = await (await request.get(`${API}/conversations/${conv.id}/messages`)).json();
  const user = all.messages.find((m: { role: string }) => m.role === 'user');

  // Rendered text of the html body, with the sanitizer's tags stripped.
  const rendered = 'Hello brave world';
  expect(user.selection_text).toBe(selected);
  expect(rendered.slice(user.selection_start, user.selection_end)).toBe(selected);
});
