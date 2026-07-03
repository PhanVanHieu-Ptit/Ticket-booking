import { test, expect, Page } from '@playwright/test';
import { gotoEvent, waitForTicketAvailable, resetInventory, restoreSeedInventory } from './helpers';

// The "first card" on the page is always "VIP" (GetAvailability returns
// VIP-then-Standard; see apps/backend/internal/handlers/availability_handler.go).
// Other files (server-down.spec.ts, spam-click.spec.ts) also reset VIP before
// each test and restore the full seed count after, so cross-file isolation
// while sharing VIP relies on running with a single Playwright worker (see
// playwright.config.ts).
const CLOCK_SKEW_CATEGORY = 'VIP' as const;
const STARTING_INVENTORY = 5;

test.beforeEach(async ({ request }) => {
  await resetInventory(request, CLOCK_SKEW_CATEGORY, STARTING_INVENTORY);
});

test.afterEach(async ({ request }) => {
  await restoreSeedInventory(request);
});

async function reserveButtonForFirstCard(page: Page) {
  await expect(page.getByText('Inventory Remaining').first()).toBeVisible();
  await waitForTicketAvailable(page);
  const card = page
    .getByText('Inventory Remaining', { exact: true })
    .first()
    .locator('xpath=ancestor::div[.//h2][1]');
  return card.getByRole('button');
}

async function createActiveHold(page: Page) {
  const reserveButton = await reserveButtonForFirstCard(page);
  await reserveButton.click();
  await expect(page).toHaveURL(/\/checkout/, { timeout: 10_000 });
}

/** Parses the "Hold expires in mm:ss" countdown into total seconds. */
async function readCountdownSeconds(page: Page): Promise<number> {
  const text = await page.getByText(/Hold expires in/i).innerText();
  const match = text.match(/(\d{2}):(\d{2})/);
  if (!match) {
    throw new Error(`Could not parse countdown text: "${text}"`);
  }
  return parseInt(match[1], 10) * 60 + parseInt(match[2], 10);
}

test('countdown keeps counting down smoothly when the OS clock is turned back mid-session', async ({ page }) => {
  await gotoEvent(page);
  await createActiveHold(page);
  await expect(page.getByText(/Hold expires in/i)).toBeVisible();

  const before = await readCountdownSeconds(page);
  expect(before).toBeGreaterThan(0);

  // Simulate a user turning their OS clock back 10 minutes mid-session.
  // Real clock changes only ever affect Date/Date.now() -- never the
  // monotonic performance.now() the countdown is now built on -- so this
  // patches exactly (and only) what a real clock change would touch.
  await page.evaluate(() => {
    const skewMs = -10 * 60 * 1000;
    const realNow = Date.now.bind(Date);
    Date.now = () => realNow() + skewMs;
  });

  await page.waitForTimeout(3000);
  const after = await readCountdownSeconds(page);

  // Must have kept counting down by roughly the real elapsed time -- never
  // jumped upward (the ~10 minute jump a Date.now()-based countdown would
  // have shown), never gone negative, never produced a nonsensical value.
  expect(after).toBeGreaterThan(0);
  expect(after).toBeLessThan(before);
  expect(before - after).toBeLessThanOrEqual(6); // generous tolerance for CI scheduling jitter
});

test('countdown reaching zero waits for a real server round trip before showing "Hold Expired"', async ({ page }) => {
  const VERIFY_DELAY_MS = 2000;
  let getHoldCallCount = 0;

  // Intercept every GET to the active-hold endpoint. The real backend hold
  // and its actual 5-minute TTL are left completely untouched -- only the
  // *displayed* seconds_remaining is faked down so the countdown reaches
  // zero in a couple of seconds instead of five real minutes.
  await page.route('**/api/v1/tickets/hold', async (route) => {
    if (route.request().method() !== 'GET') {
      await route.continue();
      return;
    }
    getHoldCallCount += 1;

    if (getHoldCallCount === 1) {
      // Initial mount sync: shrink the baseline the countdown starts from.
      const response = await route.fetch();
      const body = await response.json();
      if (body?.data) {
        body.data.seconds_remaining = 2;
      }
      await route.fulfill({ response, json: body });
      return;
    }

    // Every later GET is the app's own zero-hit verification call. Delay it
    // and report the hold as genuinely gone (404 NO_ACTIVE_HOLD), simulating
    // server-side truth that the client clock alone could never know.
    await new Promise((resolve) => setTimeout(resolve, VERIFY_DELAY_MS));
    await route.fulfill({
      status: 404,
      contentType: 'application/json',
      body: JSON.stringify({ success: false, error: { code: 'NO_ACTIVE_HOLD', message: 'No active hold found' } }),
    });
  });

  await gotoEvent(page);
  await createActiveHold(page);
  await expect(page.getByText(/Hold expires in/i)).toBeVisible();

  // Countdown ticks down to the last positive second and then holds there
  // (by design: the UI doesn't flash "expired" purely off the local timer;
  // it freezes on the last known-good reading while it verifies).
  await expect.poll(() => readCountdownSeconds(page), { timeout: 6_000 }).toBeLessThanOrEqual(1);

  // The verification request is deliberately still in flight -- the modal
  // (its heading, distinct from the small "Hold Expired" status badge that
  // can render elsewhere once `expired` flips) must not have appeared yet
  // from the local clock alone.
  const expirationModalHeading = page.getByRole('heading', { name: 'Hold Expired' });
  await expect(expirationModalHeading).not.toBeVisible();

  // Once the delayed, real verification response confirms expiry, the modal appears.
  await expect(expirationModalHeading).toBeVisible({ timeout: VERIFY_DELAY_MS + 5_000 });
  expect(getHoldCallCount).toBeGreaterThanOrEqual(2);
});
