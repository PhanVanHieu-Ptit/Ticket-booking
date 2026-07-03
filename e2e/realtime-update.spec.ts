import { test, expect, Page, Browser, BrowserContext } from '@playwright/test';
import { gotoEvent, getAvailableCount, waitForTicketAvailable, resetInventory, restoreSeedInventory } from './helpers';

// The app exposes exactly one hardcoded event on the root route with two fixed
// ticket categories, "VIP" and "Standard" (see apps/backend/db/seeds/seed_tickets.sql).
// GetAvailability always returns VIP before Standard (see
// apps/backend/internal/handlers/availability_handler.go), so "the first card"
// on the page is reliably the VIP category across independent page loads.
const REALTIME_CATEGORY = 'VIP' as const;
const STARTING_INVENTORY = 8; // just needs to be > 1 so a single hold never sells the category out

/** Locates the first ticket category card on the page (always VIP, see REALTIME_CATEGORY above). */
function firstCategoryCard(page: Page) {
  return page.getByText('Inventory Remaining', { exact: true }).first().locator('xpath=ancestor::div[.//h2][1]');
}

/** Opens a fresh independent browser session (own cookies/session) on the event page. */
async function openEventPage(browser: Browser): Promise<{ context: BrowserContext; page: Page }> {
  const context = await browser.newContext();
  const page = await context.newPage();
  await gotoEvent(page);
  // waitForTicketAvailable's poll doesn't retry a thrown error from
  // getAvailableCount, so wait for a real card before polling (see spam-click.spec.ts).
  await expect(page.getByText('Inventory Remaining').first()).toBeVisible();
  await waitForTicketAvailable(page, REALTIME_CATEGORY);
  return { context, page };
}

/**
 * Clicks Reserve on the first category card and waits for the hold to
 * succeed. Success is confirmed the same way spam-click/race-condition specs
 * do it: booking.page.tsx's handleReserve() navigates to /checkout only once
 * bookingApi.reserveTicket() actually resolves (i.e. the backend confirmed the hold).
 */
async function holdOneTicket(page: Page): Promise<void> {
  const button = firstCategoryCard(page).getByRole('button');
  await expect(button).toBeEnabled();
  await button.click();
  await expect(page).toHaveURL(/\/checkout/, { timeout: 10_000 });
}

// Both tests below reset and consume the same shared "VIP" category, so run
// them one at a time within this file (same reasoning as race-condition.spec.ts).
test.describe.serial('ticket availability updates live without a page reload', () => {
  // Restore VIP to its full seed count after each test so the one seat this
  // file consumes per test doesn't leak into spam-click.spec.ts,
  // server-down.spec.ts, or the next run of this file.
  test.afterEach(async ({ request }) => {
    await restoreSeedInventory(request);
  });

  test('another session holding a ticket updates the count within 5s, no reload', async ({ browser, request }) => {
    await resetInventory(request, REALTIME_CATEGORY, STARTING_INVENTORY);

    // Page A: the "spectator" watching the homepage/booking page.
    const { context: contextA, page: pageA } = await openEventPage(browser);
    // Page B: an independent session/context that will grab a ticket.
    const { context: contextB, page: pageB } = await openEventPage(browser);

    try {
      const initialCount = await getAvailableCount(pageA);

      await holdOneTicket(pageB);

      // Deliberately no page.reload() here -- the count on page A must update
      // itself via the "inventory_update" SSE event pushed by
      // apps/backend/internal/sse/broker.go through useTicketAvailability.ts.
      await expect
        .poll(() => getAvailableCount(pageA), {
          timeout: 5_000,
          message: 'available count on page A did not auto-update after page B held a ticket',
        })
        .toBe(initialCount - 1);
    } finally {
      await contextA.close();
      await contextB.close();
    }
  });

  // The project's live-update transport is SSE (EventSource), not a raw
  // WebSocket -- see apps/frontend/src/hooks/useTicketAvailability.ts. It has
  // the same "reconnect and resync" contract a WebSocket implementation would,
  // so we test it the same way the prompt describes for a WebSocket.
  //
  // NOTE on approach: contextA.setOffline(true) (the API the prompt suggests)
  // was tried first, but verified empirically to NOT interrupt an
  // already-open EventSource/SSE stream on Chromium+localhost -- the page
  // kept reporting "Syncing Live" for 40+ seconds after going offline (CDP's
  // offline emulation blocks *new* requests but doesn't tear down an
  // in-flight streaming response). Aborting the specific SSE request via
  // page.route reliably drives the same onerror/reconnect code path instead.
  test('SSE stream reconnects and resyncs to the latest count after a connection drop', async ({ browser, request }) => {
    await resetInventory(request, REALTIME_CATEGORY, STARTING_INVENTORY);

    const contextA = await browser.newContext();
    const pageA = await contextA.newPage();

    // Block the stream before page A's first connection attempt, so it fails
    // immediately and falls into the degraded/reconnect path. The trailing
    // "**" is required: connectSSE() now authenticates the stream via a
    // "?session_token=..." query param (see useTicketAvailability.ts), and a
    // glob with no wildcard after "stream" does not match a URL that has a
    // query string appended.
    let streamBlocked = true;
    await pageA.route('**/tickets/availability/stream**', (route) => {
      if (streamBlocked) {
        route.abort();
      } else {
        route.continue();
      }
    });

    const { context: contextB, page: pageB } = await openEventPage(browser);

    try {
      await gotoEvent(pageA);
      await expect(pageA.getByText('Inventory Remaining').first()).toBeVisible();
      await waitForTicketAvailable(pageA, REALTIME_CATEGORY);
      const initialCount = await getAvailableCount(pageA);

      // useTicketAvailability's es.onerror sets isConnected=false AND calls
      // startPolling() (isDegraded=true) in the same tick, so the badge goes
      // straight to "Slow Mode · Polling", not an intermediate "Reconnecting...".
      await expect(pageA.getByText('Slow Mode · Polling')).toBeVisible({ timeout: 10_000 });

      // While page A's stream is down, page B holds a ticket. Page A
      // necessarily misses the live SSE push for this and must resync once
      // its stream reconnects.
      await holdOneTicket(pageB);

      // Unblock the stream; the next scheduled reconnect attempt (exponential
      // backoff capped at 30s, see connectSSE() in useTicketAvailability.ts)
      // should succeed and deliver a fresh "initial_state" event. The 8s REST
      // polling fallback would also independently correct the count.
      streamBlocked = false;

      // Deliberately no page.reload() -- must resync live.
      await expect
        .poll(() => getAvailableCount(pageA), {
          timeout: 40_000,
          message: 'page A never resynced to the latest count after reconnecting',
        })
        .toBe(initialCount - 1);

      await expect(pageA.getByText('Syncing Live')).toBeVisible({ timeout: 10_000 });
    } finally {
      await contextA.close();
      await contextB.close();
    }
  });
});
