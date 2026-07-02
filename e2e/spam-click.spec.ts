import { test, expect } from '@playwright/test';
import { gotoEvent, waitForTicketAvailable, resetInventory, restoreSeedInventory } from './helpers';

// NOTE: the reservation endpoint is actually `/api/v1/tickets/reserve`
// (see apps/frontend/src/modules/booking/booking.api.ts), not `/reservations`.
const RESERVE_ENDPOINT_GLOB = '**/api/v1/tickets/reserve';
const RESERVE_ENDPOINT_PATH = '/api/v1/tickets/reserve';
const NETWORK_DELAY_MS = 1500;
const SPAM_CLICKS = 10;

// The "first card" on the page (see gotoEvent/getAvailableCount in
// ./helpers.ts) is always the "VIP" category (GetAvailability returns
// VIP-then-Standard; see apps/backend/internal/handlers/availability_handler.go).
// server-down.spec.ts also uses the first card, so this file resets VIP to a
// known count before running and restores the full seed count afterwards to
// keep both files from leaking depleted VIP stock into each other or into
// later runs. Cross-file isolation while both specs touch VIP is guaranteed
// by running with a single Playwright worker (see playwright.config.ts).
const SPAM_CLICK_CATEGORY = 'VIP' as const;
const STARTING_INVENTORY = 5;

test.beforeEach(async ({ request }) => {
  await resetInventory(request, SPAM_CLICK_CATEGORY, STARTING_INVENTORY);
});

test.afterEach(async ({ request }) => {
  await restoreSeedInventory(request);
});

test('spam-clicking Reserve sends exactly one reservation request', async ({ page }) => {
  // 1. Throttle the reservation POST to emulate the slow-network manual repro.
  await page.route(RESERVE_ENDPOINT_GLOB, async (route) => {
    if (route.request().method() === 'POST') {
      await new Promise((resolve) => setTimeout(resolve, NETWORK_DELAY_MS));
    }
    await route.continue();
  });

  // 2. Land on the booking page for a category that still has stock.
  await gotoEvent(page);
  // waitForTicketAvailable's expect.poll doesn't retry a thrown error from
  // getAvailableCount (only a failed matcher on a returned value), so calling
  // it while the loading skeleton is still showing fails instead of retrying.
  // Wait for a real card first so the poll always finds one.
  await expect(page.getByText('Inventory Remaining').first()).toBeVisible();
  await waitForTicketAvailable(page);

  // Scope the button to its card container (structurally stable) rather than
  // its accessible name. The button's text flips to "Reserving..." as soon as
  // the first click lands, so a live name-based locator (e.g. getByRole('button',
  // { name: /^Reserve /i }).first()) would silently re-target a *different*
  // card's button on the remaining spam clicks instead of re-clicking this one.
  const card = page.getByText('Inventory Remaining', { exact: true }).first().locator('xpath=ancestor::div[.//h3][1]');
  const reserveButton = card.getByRole('button');
  await expect(reserveButton).toBeEnabled();

  const buttonText = await reserveButton.innerText();
  const categoryName = buttonText.match(/^Reserve\s+(.+?)\s+Ticket$/i)?.[1];
  if (!categoryName) {
    throw new Error(`Could not parse category name from button text: "${buttonText}"`);
  }

  const availabilityBefore = await page.request.get(RESERVE_ENDPOINT_PATH.replace('reserve', 'availability'));
  const categoryBefore = (await availabilityBefore.json()).data.categories.find(
    (c: { name: string }) => c.name === categoryName
  );

  // 3. Record every POST actually sent to the reservation endpoint for the rest of the test.
  const reservePostUrls: string[] = [];
  page.on('request', (request) => {
    if (request.method() === 'POST' && request.url().includes(RESERVE_ENDPOINT_PATH)) {
      reservePostUrls.push(request.url());
    }
  });

  // 4. Fire 10 clicks back-to-back without awaiting in between (spam click).
  //    force:true skips Playwright's "must be enabled" pre-click wait, otherwise
  //    it would just queue clicks 2-10 until the button re-enables ~1.5s later,
  //    which is not a "very short time" spam click anymore.
  const clicks = Array.from({ length: SPAM_CLICKS }, () =>
    reserveButton.click({ force: true }).catch(() => {})
  );

  // 6. The very first click should flip the button into disabled state almost immediately.
  await expect(reserveButton).toBeDisabled({ timeout: 500 });

  await Promise.all(clicks);

  // Wait for the throttled reservation to resolve and the app to navigate to /checkout.
  await expect(page).toHaveURL(/\/checkout/, { timeout: NETWORK_DELAY_MS + 5_000 });

  // 5. Exactly one POST should have reached the network layer, not ten.
  expect(reservePostUrls).toHaveLength(1);

  // Bonus check requested in the task: GET /api/v1/admin/metrics does report sold-ticket
  // counts, but it sits behind AdminAuthMiddleware (apps/backend/cmd/api/main.go:170-174)
  // requiring a login passcode this test doesn't have. Cross-check via the public
  // availability endpoint instead: exactly one seat of this category should be consumed.
  const availabilityAfter = await page.request.get(RESERVE_ENDPOINT_PATH.replace('reserve', 'availability'));
  const categoryAfter = (await availabilityAfter.json()).data.categories.find(
    (c: { name: string }) => c.name === categoryName
  );
  expect(categoryBefore.available - categoryAfter.available).toBe(1);
});
