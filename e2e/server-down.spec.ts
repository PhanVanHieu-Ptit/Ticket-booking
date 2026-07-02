import { test, expect, Page } from '@playwright/test';
import { gotoEvent, waitForTicketAvailable, resetInventory, restoreSeedInventory } from './helpers';

const API_GLOB = '**/api/**';

// The "first card" on the page (see reserveButtonForFirstCard below) is
// always the "VIP" category (GetAvailability returns VIP-then-Standard; see
// apps/backend/internal/handlers/availability_handler.go). spam-click.spec.ts
// also uses the first card, so this file resets VIP to a known count before
// every test and restores the full seed count afterwards to keep both files
// from leaking depleted/sold VIP stock into each other or into later runs.
// Cross-file isolation while both specs touch VIP is guaranteed by running
// with a single Playwright worker (see playwright.config.ts).
const SERVER_DOWN_CATEGORY = 'VIP' as const;
const STARTING_INVENTORY = 5;

// The app's own error copy is English ("Failed to...", "...error...", "try
// again"), but we match loosely enough to also pass against a
// Vietnamese-localized build ("không thể kết nối", "đã có lỗi xảy ra")
// without hardcoding either exact wording.
const FRIENDLY_ERROR_TEXT =
  /không\s*th[ểe]|đã\s*có\s*l[ỗo]i|l[ỗo]i\s*x[ảa]y\s*ra|error|fail(ed)?|unable|cannot|try again|something went wrong/i;

// Rendered by the outer <Layout> in App.tsx, independent of whatever the
// routed page below it is doing — proves the app didn't hard-crash to a
// blank/white screen.
async function expectHeaderAlive(page: Page) {
  await expect(page.getByText('TICKET RUSH', { exact: true })).toBeVisible();
}

async function reserveButtonForFirstCard(page: Page) {
  await expect(page.getByText('Inventory Remaining').first()).toBeVisible();
  await waitForTicketAvailable(page);
  const card = page
    .getByText('Inventory Remaining', { exact: true })
    .first()
    .locator('xpath=ancestor::div[.//h3][1]');
  return card.getByRole('button');
}

async function createActiveHold(page: Page) {
  const reserveButton = await reserveButtonForFirstCard(page);
  await reserveButton.click();
  await expect(page).toHaveURL(/\/checkout/, { timeout: 10_000 });
}

async function fillValidPaymentForm(page: Page) {
  await page.getByPlaceholder('you@example.com').fill('buyer@example.com');
  await page.getByPlaceholder('John Doe').fill('Jane Doe');
  await page.getByPlaceholder('4111 1111 1111 1111').fill('4111111111111111');
  await page.getByPlaceholder('MM/YY').fill('12/29');
  await page.getByPlaceholder('123').fill('123');
}

const scenarios: { name: string; install: (page: Page) => Promise<void> }[] = [
  {
    name: 'total network failure (aborted requests)',
    install: async (page) => {
      await page.route(API_GLOB, (route) => route.abort('failed'));
    },
  },
  {
    name: 'server responds with HTTP 500',
    install: async (page) => {
      await page.route(API_GLOB, (route) =>
        route.fulfill({
          status: 500,
          contentType: 'application/json',
          body: JSON.stringify({
            success: false,
            error: { code: 'INTERNAL_ERROR', message: 'Internal server error. Please try again later.' },
          }),
        })
      );
    },
  },
];

// Every test below reserves/holds/sells against the same shared "VIP"
// category, so wrap the whole file (both scenarios) in serial mode -- nested
// describe blocks inherit it -- to keep resets/holds from one test from
// racing another's.
test.describe.serial('backend outage handling', () => {
  test.beforeEach(async ({ request }) => {
    await resetInventory(request, SERVER_DOWN_CATEGORY, STARTING_INVENTORY);
  });

  test.afterEach(async ({ request }) => {
    await restoreSeedInventory(request);
  });

  for (const scenario of scenarios) {
    test.describe(`backend outage: ${scenario.name}`, () => {
      test('loading the homepage shows a friendly error and self-recovers once the backend is back (no reload)', async ({
        page,
      }) => {
        test.setTimeout(60_000);

        await scenario.install(page);
        await gotoEvent(page);

        // Friendly error, not a raw crash / white screen.
        await expect(page.getByText(FRIENDLY_ERROR_TEXT).first()).toBeVisible({ timeout: 10_000 });
        await expectHeaderAlive(page);

        // Restore the network and let the app's own SSE-reconnect/poll loop
        // recover on its own — no page.reload() involved.
        await page.unroute(API_GLOB);
        await expect(page.getByText('Inventory Remaining').first()).toBeVisible({ timeout: 45_000 });
        await waitForTicketAvailable(page);
      });

      test('clicking Reserve mid-session shows a friendly error and can be retried successfully (no reload)', async ({
        page,
      }) => {
        test.setTimeout(45_000);

        // Load normally first so the ticket cards (and the Reserve button)
        // actually exist. If the backend were already down on first load there
        // would be no cards to click at all — that path is covered by the
        // "loading the homepage" test above.
        await gotoEvent(page);
        const reserveButton = await reserveButtonForFirstCard(page);

        await scenario.install(page);
        await reserveButton.click();

        await expect(page.getByText(FRIENDLY_ERROR_TEXT).first()).toBeVisible({ timeout: 10_000 });
        await expectHeaderAlive(page);
        await expect(page).not.toHaveURL(/\/checkout/);

        // Restore the network and retry the exact same action — no reload.
        await page.unroute(API_GLOB);
        await expect(reserveButton).toBeEnabled();
        await reserveButton.click();
        await expect(page).toHaveURL(/\/checkout/, { timeout: 10_000 });
      });

      test('clicking Pay Now mid-session shows a friendly error and can be retried successfully (no reload)', async ({
        page,
      }) => {
        test.setTimeout(45_000);

        await gotoEvent(page);
        await createActiveHold(page);
        await fillValidPaymentForm(page);

        await scenario.install(page);
        await page.getByRole('button', { name: /Pay Now/i }).click();

        await expect(page.getByText(FRIENDLY_ERROR_TEXT).first()).toBeVisible({ timeout: 10_000 });
        await expectHeaderAlive(page);
        // Still parked on checkout, not bounced anywhere odd.
        await expect(page).toHaveURL(/\/checkout/);

        // Restore the network and retry the exact same submit — no reload.
        await page.unroute(API_GLOB);
        await page.getByRole('button', { name: /Pay Now/i }).click();
        await expect(page).toHaveURL(/\/confirmation/, { timeout: 10_000 });
      });
    });
  }
});
