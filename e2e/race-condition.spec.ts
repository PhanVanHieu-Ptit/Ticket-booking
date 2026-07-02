import { test, expect, APIRequestContext, Page, Browser } from '@playwright/test';
import { gotoEvent, resetInventory, restoreSeedInventory } from './helpers';

// This backend has no "ticketTypeId" concept: inventory is split into two
// fixed categories, "VIP" and "Standard" (see apps/backend/db/seeds/seed_tickets.sql).
// We race on "Standard" specifically because spam-click.spec.ts and
// server-down.spec.ts both grab the *first* card on the page, which is
// always "VIP" (categories are returned VIP-then-Standard by
// GetAvailability, see apps/backend/internal/handlers/availability_handler.go).
// Using a different category keeps this file's inventory resets from
// stomping on those specs' inventory. Cross-file isolation is additionally
// guaranteed by running with a single Playwright worker (see
// playwright.config.ts), since two files racing on Standard/VIP truly in
// parallel would still corrupt each other's counts.
const RACE_CATEGORY = 'Standard' as const;
const CATEGORY_HEADING = 'Standard Pass'; // heading text rendered by TicketCategoryCard for "Standard"

const ADMIN_LOGIN_PATH = '/api/v1/admin/login';
const ADMIN_METRICS_PATH = '/api/v1/admin/metrics';
// Seeded in apps/backend/db/seeds/seed_tickets.sql as the bcrypt hash of 'admin123'.
const ADMIN_PASSCODE = 'admin123';

const REPEAT_COUNT = 10;

async function getAdminToken(request: APIRequestContext): Promise<string> {
  const res = await request.post(ADMIN_LOGIN_PATH, { data: { passcode: ADMIN_PASSCODE } });
  if (!res.ok()) {
    throw new Error(`Admin login failed: ${res.status()} ${await res.text()}`);
  }
  const body = await res.json();
  return body.data.token as string;
}

/** Reads how many tickets are currently Holding in the given category via the admin metrics API. */
async function getHeldCount(request: APIRequestContext, token: string, category: 'VIP' | 'Standard'): Promise<number> {
  const res = await request.get(ADMIN_METRICS_PATH, { headers: { Authorization: `Bearer ${token}` } });
  if (!res.ok()) {
    throw new Error(`Admin metrics fetch failed: ${res.status()} ${await res.text()}`);
  }
  const body = await res.json();
  return body.data.held_inventory[category] as number;
}

/**
 * Locates the ticket category card by its rendered heading (e.g. "Standard Pass").
 * Mirrors the "Inventory Remaining" → ancestor::div[.//h3][1] pattern from
 * ./helpers.ts: anchoring the ancestor climb on the h3 itself instead would
 * resolve to an inner wrapper div (heading+badge only) that doesn't contain
 * the Reserve button, since the button lives in a sibling subtree of the card.
 */
function getCategoryCard(page: Page, headingText: string) {
  return page
    .getByText('Inventory Remaining', { exact: true })
    .locator('xpath=ancestor::div[.//h3][1]')
    .filter({ has: page.getByRole('heading', { name: headingText, exact: true }) });
}

type ReserveOutcome = 'held' | 'sold_out';

/**
 * Fires a forced click on the category's Reserve button without awaiting the
 * click's own resolution, so callers can fire two of these in a true
 * Promise.all() race across two independent browser contexts/sessions.
 */
function fireReserveClick(page: Page): Promise<void> {
  const button = getCategoryCard(page, CATEGORY_HEADING).getByRole('button');
  return button.click({ force: true }).catch(() => {});
}

/**
 * Waits for a page's reservation attempt to resolve into one of two outcomes:
 * - 'held': navigated to /checkout (booking.page.tsx navigates there on success
 *   or on ACTIVE_HOLD_EXISTS), where checkout.page.tsx shows the hold's
 *   countdown ("Hold expires in mm:ss").
 * - 'sold_out': stayed on the booking page and shows the inline
 *   "sold out" error (booking.page.tsx's TICKET_UNAVAILABLE branch), with no
 *   broken/blank UI.
 */
async function waitForReserveOutcome(page: Page): Promise<ReserveOutcome> {
  await expect
    .poll(
      async () => {
        if (/\/checkout/.test(page.url())) return 'held';
        const soldOutVisible = await page
          .getByText(/sold out/i)
          .first()
          .isVisible()
          .catch(() => false);
        if (soldOutVisible) return 'sold_out';
        return 'pending';
      },
      { timeout: 10_000, message: `page never resolved to a held/sold-out state; final URL: ${page.url()}` }
    )
    .not.toBe('pending');

  return /\/checkout/.test(page.url()) ? 'held' : 'sold_out';
}

/**
 * Runs one full "two users race for the last ticket" round: resets inventory
 * to a single seat, opens two independent browser contexts (= two independent
 * sessions, since the backend enforces a one-ticket-per-session purchase
 * limit via Redis), fires both Reserve clicks concurrently, and asserts
 * exactly one side wins.
 */
async function runOversellRace(browser: Browser, request: APIRequestContext): Promise<void> {
  await resetInventory(request, RACE_CATEGORY, 1);

  const contextA = await browser.newContext();
  const contextB = await browser.newContext();
  try {
    const pageA = await contextA.newPage();
    const pageB = await contextB.newPage();

    await Promise.all([gotoEvent(pageA), gotoEvent(pageB)]);
    await expect(getCategoryCard(pageA, CATEGORY_HEADING).getByRole('button')).toBeEnabled();
    await expect(getCategoryCard(pageB, CATEGORY_HEADING).getByRole('button')).toBeEnabled();

    // Bấm nút "Chọn vé" (Reserve) ở CẢ HAI page thật sự đồng thời: cả hai
    // click() được gọi song song (không await tuần tự) trước khi chờ kết quả.
    const clicks = Promise.all([fireReserveClick(pageA), fireReserveClick(pageB)]);
    const outcomes = await Promise.all([waitForReserveOutcome(pageA), waitForReserveOutcome(pageB)]);
    await clicks;

    const heldCount = outcomes.filter((o) => o === 'held').length;
    const soldOutCount = outcomes.filter((o) => o === 'sold_out').length;

    expect(heldCount, `expected exactly 1 winner, outcomes were: ${JSON.stringify(outcomes)}`).toBe(1);
    expect(soldOutCount, `expected exactly 1 loser, outcomes were: ${JSON.stringify(outcomes)}`).toBe(1);

    // Assert the winner's page actually shows a real hold with a countdown,
    // not just a bare /checkout URL.
    const winnerPage = outcomes[0] === 'held' ? pageA : pageB;
    await expect(winnerPage.getByText(/Hold expires in/i)).toBeVisible();

    // Data-layer assertion: no more than one ticket in this category can be
    // Holding (or beyond) at once — the whole point of the atomic Redis hold
    // script plus the Postgres row-lock second layer.
    const token = await getAdminToken(request);
    const heldInDb = await getHeldCount(request, token, RACE_CATEGORY);
    expect(heldInDb, 'oversell detected: more than 1 reservation is Holding for a category reset to 1 seat').toBeLessThanOrEqual(1);
  } finally {
    await contextA.close();
    await contextB.close();
  }
}

// Every iteration resets and races over the same shared "Standard" category,
// so two iterations running concurrently would let a third, untracked
// context from another iteration steal the "last" ticket out from under this
// iteration's own two contexts. test.describe.serial() forces this file's
// tests to run one at a time, in order, and aborts the remaining ones if any
// iteration fails, to keep each race isolated and reports readable.
test.describe.serial('oversell protection: two users race for the last ticket', () => {
  // Each iteration leaves "Standard" collapsed down to 1-seat-then-resolved
  // (see resetInventory(RACE_CATEGORY, 1) inside runOversellRace). Restore
  // both categories to their full seed counts after every iteration so
  // nothing here stays pinned near-empty for whatever test runs next. Uses
  // afterEach (a test-scoped hook, so it can use the test-scoped `request`
  // fixture) rather than afterAll so cleanup still happens even if a later
  // iteration in this file fails or times out.
  test.afterEach(async ({ request }) => {
    await restoreSeedInventory(request);
  });

  test('exactly one of two simultaneous reservations wins the last seat', async ({ browser, request }) => {
    await runOversellRace(browser, request);
  });

  test.describe.serial('repeated race (rules out a lucky non-collision)', () => {
    for (let i = 1; i <= REPEAT_COUNT; i++) {
      test(`iteration ${i}/${REPEAT_COUNT}`, async ({ browser, request }) => {
        await runOversellRace(browser, request);
      });
    }
  });
});
