import { test, expect } from '@playwright/test';

test.beforeAll(async ({ request }) => {
  const r = await request.get('http://127.0.0.1:8080/api/v1/groups');
  if (r.status() >= 500) {
    throw new Error(`zen-backend at 127.0.0.1:8080 returned ${r.status()} — start it first`);
  }
});

test('create group, then create card, then see card in group', async ({ page }) => {
  const groupName = `m7c-smoke-${Date.now()}`;
  const cardTitle = `Card ${Date.now()}`;

  await page.goto('/');
  await page.waitForLoadState('networkidle');
  await expect(page.getByText('Zen')).toBeVisible();

  const aside = page.locator('aside');

  await aside.getByRole('button', { name: '+ New group' }).click();
  const groupInput = aside.getByPlaceholder('Group name');
  await groupInput.fill(groupName);
  await groupInput.press('Enter');
  const groupLink = aside.getByRole('link', { name: groupName });
  await expect(groupLink).toBeVisible();

  await groupLink.click();
  await expect(page.getByRole('heading', { name: groupName })).toBeVisible();
  await page.waitForLoadState('networkidle');

  const postCard = page.waitForResponse((r) => r.url().endsWith('/api/v1/cards') && r.request().method() === 'POST');
  await page.getByRole('button', { name: '+ Card', exact: true }).click();
  await page.getByPlaceholder('Title').fill(cardTitle);
  await page.getByRole('button', { name: 'Create' }).click();
  await postCard;
  // Reload to verify persistence from server (independent of the in-memory store race).
  await page.reload();
  await page.waitForLoadState('networkidle');

  await expect(page.getByRole('link', { name: cardTitle })).toBeVisible({ timeout: 10000 });
});

test('Discuss this group → send message → message appears in sidebar thread', async ({ page }) => {
  const groupName = `m12-${Date.now()}`;

  await page.goto('/');
  await page.waitForLoadState('networkidle');

  // Use the LeftRail aside (first one).
  const leftRail = page.locator('aside').first();
  await leftRail.getByRole('button', { name: '+ New group' }).click();
  const groupInput = leftRail.getByPlaceholder('Group name');
  await groupInput.fill(groupName);
  await groupInput.press('Enter');
  await leftRail.getByRole('link', { name: groupName }).click();
  await expect(page.getByRole('heading', { name: groupName })).toBeVisible();

  // Click "Discuss this group" — the sidebar should slide in.
  await page.getByRole('button', { name: 'Discuss this group' }).click();

  // The chat sidebar is the aside with aria-label="Chat sidebar".
  const sidebar = page.getByRole('complementary', { name: 'Chat sidebar' });
  await expect(sidebar).toBeVisible();

  // Type a message + send.
  const textarea = sidebar.locator('textarea');
  const sendMsg = `hello group ${Date.now()}`;
  await textarea.fill(sendMsg);
  await sidebar.getByRole('button', { name: 'Send' }).click();

  // Message should land in the thread quickly via REST + WS echo.
  await expect(sidebar.getByText(sendMsg).first()).toBeVisible({ timeout: 5000 });
});
