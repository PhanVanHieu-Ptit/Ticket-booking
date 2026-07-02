import { APIRequestContext, Page, expect } from '@playwright/test';

export const RESET_INVENTORY_PATH = '/api/v1/test/reset-inventory';

// The two fixed ticket categories seeded by apps/backend/db/seeds/seed_tickets.sql,
// and their original seed counts. Every spec that mutates inventory via
// resetInventory() should restore both categories to these counts afterwards
// (see restoreSeedInventory) so it doesn't leave depleted stock behind for
// whichever spec file/run happens to execute next.
export const SEED_INVENTORY = { VIP: 100, Standard: 400 } as const;
export type TicketCategory = keyof typeof SEED_INVENTORY;

/**
 * Resets a ticket category down to exactly `available` Available tickets via
 * the test-only backend endpoint (mounted only when the API server is not
 * running with APP_ENV=production; see cmd/api/main.go). Used both to set up
 * a deterministic starting count for a scenario and, called with the seed
 * totals, to clean a category back up afterwards.
 */
export async function resetInventory(
  request: APIRequestContext,
  category: TicketCategory,
  available: number
): Promise<void> {
  const res = await request.post(RESET_INVENTORY_PATH, { data: { category, available } });
  if (!res.ok()) {
    throw new Error(
      `Failed to reset inventory for "${category}" to ${available}: ${res.status()} ${await res.text()}\n` +
        'Is the backend running with APP_ENV != production so /api/v1/test/reset-inventory is mounted?'
    );
  }
}

/**
 * Restores both ticket categories to their original seed counts. Call this in
 * an afterEach/afterAll so a spec's own reservations/holds/sales never leak
 * into the next test (in this file or another) as depleted inventory.
 */
export async function restoreSeedInventory(request: APIRequestContext): Promise<void> {
  await resetInventory(request, 'VIP', SEED_INVENTORY.VIP);
  await resetInventory(request, 'Standard', SEED_INVENTORY.Standard);
}

/**
 * Navigate to an event's booking page.
 * The app currently only exposes a single hardcoded event at the root route,
 * so eventId is accepted (and ignored) for forward-compatibility once
 * per-event routes (e.g. /events/:id) exist.
 */
export async function gotoEvent(page: Page, eventId?: string): Promise<void> {
  void eventId;
  await page.goto('/');
}

/**
 * Reads the "X / Y Available" count shown on a ticket category card.
 * categoryLabel matches against the card's visible heading (e.g. "VIP Experience",
 * "Standard Pass"); omit it to read the first card on the page.
 */
export async function getAvailableCount(page: Page, categoryLabel?: string): Promise<number> {
  const inventoryLabels = page.getByText('Inventory Remaining', { exact: true });
  const cardCount = await inventoryLabels.count();

  for (let i = 0; i < cardCount; i++) {
    const card = inventoryLabels.nth(i).locator('xpath=ancestor::div[.//h3][1]');

    if (categoryLabel) {
      const heading = await card.locator('h3').innerText();
      if (!heading.toLowerCase().includes(categoryLabel.toLowerCase())) continue;
    }

    const availabilityText = await card.getByText(/\d+\s*\/\s*\d+\s*Available/).innerText();
    const match = availabilityText.match(/(\d+)\s*\/\s*(\d+)\s*Available/);
    if (!match) {
      throw new Error(`Could not parse availability text: "${availabilityText}"`);
    }
    return parseInt(match[1], 10);
  }

  throw new Error(
    categoryLabel
      ? `No ticket category card found matching "${categoryLabel}"`
      : 'No ticket category cards found on page'
  );
}

/** Waits until a ticket category (or any, if omitted) has at least one seat available. */
export async function waitForTicketAvailable(
  page: Page,
  categoryLabel?: string,
  timeout = 15_000
): Promise<void> {
  await expect
    .poll(() => getAvailableCount(page, categoryLabel), { timeout })
    .toBeGreaterThan(0);
}
