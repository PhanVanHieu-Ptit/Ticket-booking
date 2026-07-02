import { test, expect } from '@playwright/test';
import { gotoEvent } from './helpers';

test('homepage shows the event title', async ({ page }) => {
  await gotoEvent(page);

  const heading = page.getByRole('heading', { level: 1 });
  await expect(heading).toBeVisible();
  await expect(heading).toHaveText(/Neon Symphony:\s*Hyperion Tour 2026/);
});
