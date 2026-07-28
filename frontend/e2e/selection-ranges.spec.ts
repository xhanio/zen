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

// The whole feature, through the product: drag on an html card, ask, and the
// dragged span comes back underlined. Must be a real mouse drag — Chrome only
// retargets a shadow-tree selection for genuine input, so setBaseAndExtent
// would pass even against a broken build (the v1.1.2 lesson).
test('a message selection paints an underline on the card', async ({ page, request }) => {
  const stamp = Date.now();

  const group = await (await request.post(`${API}/groups`, {
    data: { name: `msgsel-e2e-${stamp}` },
  })).json();

  const source = await (await request.post(`${API}/cards`, {
    data: {
      title: `Msgsel ${stamp}`,
      content: '<p>Hello <strong>brave</strong> world</p>',
      format: 'html',
      group_id: group.id,
    },
  })).json();

  await page.goto(`/c/${source.id}`);
  await expect(page.locator('.html-body-host')).toBeAttached();

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

  const selected = await page.evaluate(() => window.getSelection()!.toString().trim());
  expect(selected.length).toBeGreaterThan(0);

  await page.getByRole('button', { name: 'Ask' }).click();
  await page.locator('[data-test="composer-input"]').fill('what is this?');
  await page.locator('[data-test="composer-send"]').click();

  // Wait for the message to actually persist before reloading. Reloading
  // straight after the click races the POST, and the reloaded page then has
  // nothing to paint — which looks exactly like a broken feature.
  await expect.poll(async () => {
    const convs = await (await request.get(
      `${API}/conversations?anchor_kind=card&anchor_id=${source.id}`,
    )).json();
    const conv = (convs.conversations ?? [])[0];
    if (!conv) return null;
    const sels = await (await request.get(`${API}/cards/${source.id}/selections`)).json();
    return (sels.selections ?? []).length;
  }, { timeout: 10_000 }).toBe(1);

  // Reload so the card fetches its selections fresh from the endpoint.
  await page.reload();
  const bodyHost = page.locator('.html-body-host').first();
  await expect.poll(async () => {
    return await bodyHost.evaluate((host: HTMLElement) => {
      const marks = host.shadowRoot?.querySelectorAll('mark.zen-sel') ?? [];
      return Array.from(marks).map((m) => m.textContent).join('');
    });
  }, { timeout: 10_000 }).toBe(selected);
});

// The parity this feature exists to establish: the same section shows the same
// mark whether opened alone or read inside its document.
test('a section selection paints in the container view too', async ({ page, request }) => {
  const stamp = Date.now();

  const group = await (await request.post(`${API}/groups`, {
    data: { name: `sectionhl-e2e-${stamp}` },
  })).json();

  // Parent with one child section — the shape decompose produces.
  const parent = await (await request.post(`${API}/cards`, {
    data: { title: `Doc ${stamp}`, content: '', group_id: group.id },
  })).json();

  const section = await (await request.post(`${API}/cards`, {
    data: {
      title: `Section ${stamp}`,
      content: 'hello brave world',
      format: 'markdown',
      group_id: group.id,
      parent_card_id: parent.id,
    },
  })).json();

  const conv = await (await request.post(`${API}/conversations`, {
    data: { title: `sectionhl ${stamp}`, anchor_kind: 'card', anchor_id: section.id },
  })).json();

  // "brave" is [6,11) in the section's OWN rendered text — the section's offset
  // space, not the container's. No translation happens anywhere.
  await request.post(`${API}/conversations/${conv.id}/messages`, {
    data: {
      role: 'user', content: 'what is this?',
      selection_text: 'brave', selection_start: 6, selection_end: 11, selection_seq: 1,
    },
  });

  // Standalone: the baseline that already worked in v1.1.3.
  await page.goto(`/c/${section.id}`);
  await expect(page.locator('mark.zen-sel')).toHaveCount(1);
  expect(await page.locator('mark.zen-sel').textContent()).toBe('brave');

  // Container: the same mark, inside that section's subtree.
  await page.goto(`/c/${parent.id}`);
  const inSection = page.locator(`[data-card-id="${section.id}"] mark.zen-sel`);
  await expect(inSection).toHaveCount(1);
  expect(await inSection.textContent()).toBe('brave');
});

// Clicking a mark must resolve against the card that OWNS it, not the card the
// route names. In a container those differ, and the mismatch made the click a
// silent no-op — the mark painted, nothing happened.
test('clicking a section underline opens its conversation in both views', async ({ page, request }) => {
  const stamp = Date.now();

  const group = await (await request.post(`${API}/groups`, {
    data: { name: `clickroute-e2e-${stamp}` },
  })).json();
  const parent = await (await request.post(`${API}/cards`, {
    data: { title: `Doc ${stamp}`, content: '', group_id: group.id },
  })).json();
  const section = await (await request.post(`${API}/cards`, {
    data: {
      title: `Sec ${stamp}`, content: 'hello brave world', format: 'markdown',
      group_id: group.id, parent_card_id: parent.id,
    },
  })).json();
  const conv = await (await request.post(`${API}/conversations`, {
    data: { title: `clickroute ${stamp}`, anchor_kind: 'card', anchor_id: section.id },
  })).json();
  await request.post(`${API}/conversations/${conv.id}/messages`, {
    data: {
      role: 'user', content: 'what is this?', selection_text: 'brave',
      selection_start: 6, selection_end: 11, selection_seq: 1,
    },
  });

  // Leaf view: the card the route names IS the mark's owner.
  await page.goto(`/c/${section.id}`);
  await page.locator('mark.zen-sel').click();
  await expect(page.locator('[data-test="chat-panel"]')).toBeVisible();

  // Container view: the route names the parent, the mark belongs to the child.
  await page.goto(`/c/${parent.id}`);
  await page.locator(`[data-card-id="${section.id}"] mark.zen-sel`).click();
  await expect(page.locator('[data-test="chat-panel"]')).toBeVisible();
});

// Capture inside a CONTAINER. A section renders its title through HtmlBody
// too, so the section holds two shadow hosts; measuring against the first one
// (the title) produced a null range and the selection was stored with no
// offsets — visible to the user as "I selected it but no underline appeared".
test('a drag inside a container section records the offsets', async ({ page, request }) => {
  const stamp = Date.now();

  const group = await (await request.post(`${API}/groups`, {
    data: { name: `containercapture-e2e-${stamp}` },
  })).json();
  const parent = await (await request.post(`${API}/cards`, {
    data: { title: `Doc ${stamp}`, content: '', group_id: group.id },
  })).json();
  const section = await (await request.post(`${API}/cards`, {
    data: {
      title: `Sec ${stamp}`,
      content: '<p>Hello <strong>brave</strong> world</p>',
      format: 'html',
      group_id: group.id,
      parent_card_id: parent.id,
    },
  })).json();

  // Open the CONTAINER, not the section.
  await page.goto(`/c/${parent.id}`);
  const sectionSel = `[data-card-id="${section.id}"]`;
  await expect(page.locator(`${sectionSel} .html-body-host`).first()).toBeAttached();

  // Two hosts live in this section: the title's and the body's.
  const hostCount = await page.locator(`${sectionSel} .html-body-host`).count();
  expect(hostCount).toBeGreaterThan(1);

  const box = await page.evaluate((sel) => {
    const host = document.querySelector(
      `${sel} .html-body-host:not(.zen-title-html)`,
    ) as HTMLElement;
    const r = host.shadowRoot!.querySelector('p')!.getBoundingClientRect();
    return { x: r.x, y: r.y, w: r.width, h: r.height };
  }, sectionSel);

  await page.mouse.move(box.x + 2, box.y + box.h / 2);
  await page.mouse.down();
  await page.mouse.move(box.x + box.w - 2, box.y + box.h / 2, { steps: 15 });
  await page.mouse.up();

  const selected = await page.evaluate(() => window.getSelection()!.toString().trim());
  expect(selected.length).toBeGreaterThan(0);

  await page.getByRole('button', { name: 'Ask' }).click();
  await page.locator('[data-test="composer-input"]').fill('what is this?');
  await page.locator('[data-test="composer-send"]').click();

  // The offsets must land on the SECTION, measured in the section's own text.
  await expect.poll(async () => {
    const sels = await (await request.get(`${API}/cards/${section.id}/selections`)).json();
    const s = (sels.selections ?? [])[0];
    if (!s) return null;
    return 'Hello brave world'.slice(s.selection_start, s.selection_end);
  }, { timeout: 10_000 }).toBe(selected);
});

// The first-run experience, with NO reload. Every earlier test reloaded before
// asserting, which quietly hid the fact that sending a message never told the
// card anything had changed — you selected, asked, and saw nothing until you
// navigated away and back.
test('the underline appears on send, without a reload', async ({ page, request }) => {
  const stamp = Date.now();

  const group = await (await request.post(`${API}/groups`, {
    data: { name: `instant-e2e-${stamp}` },
  })).json();
  const source = await (await request.post(`${API}/cards`, {
    data: {
      title: `Instant ${stamp}`,
      content: '<p>Hello <strong>brave</strong> world</p>',
      format: 'html',
      group_id: group.id,
    },
  })).json();

  await page.goto(`/c/${source.id}`);
  await expect(page.locator('.html-body-host')).toBeAttached();

  const box = await page.evaluate((cardId) => {
    const host = document
      .querySelector(`[data-card-id="${cardId}"]`)!
      .querySelector('.html-body-host:not(.zen-title-html)') as HTMLElement;
    const r = host.shadowRoot!.querySelector('p')!.getBoundingClientRect();
    return { x: r.x, y: r.y, w: r.width, h: r.height };
  }, source.id);

  await page.mouse.move(box.x + 2, box.y + box.h / 2);
  await page.mouse.down();
  await page.mouse.move(box.x + box.w - 2, box.y + box.h / 2, { steps: 15 });
  await page.mouse.up();

  const selected = await page.evaluate(() => window.getSelection()!.toString().trim());
  expect(selected.length).toBeGreaterThan(0);

  await page.getByRole('button', { name: 'Ask' }).click();
  await page.locator('[data-test="composer-input"]').fill('what is this?');
  await page.locator('[data-test="composer-send"]').click();

  // No reload, no navigation: the mark must show up on its own.
  await expect.poll(async () => {
    return await page.locator('.html-body-host:not(.zen-title-html)').first().evaluate(
      (host: HTMLElement) =>
        Array.from(host.shadowRoot?.querySelectorAll('mark.zen-sel') ?? [])
          .map((m) => m.textContent)
          .join(''),
    );
  }, { timeout: 8_000 }).toBe(selected);
});
