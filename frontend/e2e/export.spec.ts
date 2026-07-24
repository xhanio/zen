import { test, expect } from '@playwright/test';

test.beforeAll(async ({ request }) => {
  const r = await request.get('http://127.0.0.1:8080/api/v1/groups');
  if (r.status() >= 500) {
    throw new Error(`zen-backend at 127.0.0.1:8080 returned ${r.status()} — start it first`);
  }
});

test('Export downloads a .md file named after the card title', async ({ page }) => {
  const groupName = `export-e2e-${Date.now()}`;
  const cardTitle = `Card ${Date.now()}`;

  // Create a group.
  await page.goto('/');
  await page.waitForLoadState('networkidle');
  const aside = page.locator('aside').first();
  await aside.getByRole('button', { name: '+ New group' }).click();
  const groupInput = aside.getByPlaceholder('Group name');
  await groupInput.fill(groupName);
  await groupInput.press('Enter');
  await aside.getByRole('link', { name: groupName }).click();
  await expect(page.getByRole('heading', { name: groupName })).toBeVisible();

  // Create a card in it.
  const postCard = page.waitForResponse(
    (r) => r.url().endsWith('/api/v1/cards') && r.request().method() === 'POST',
  );
  await page.getByRole('button', { name: '+ Card', exact: true }).click();
  await page.getByPlaceholder('Title').fill(cardTitle);
  await page.getByRole('button', { name: 'Create' }).click();
  await postCard;

  // Open the card view.
  await page.getByRole('link', { name: cardTitle }).click();
  await expect(page.getByRole('heading', { name: cardTitle })).toBeVisible();

  // Click Export and assert the download.
  const [download] = await Promise.all([
    page.waitForEvent('download'),
    page.getByRole('button', { name: 'Export', exact: true }).click(),
  ]);
  expect(download.suggestedFilename()).toBe(`${cardTitle}.md`);
});
