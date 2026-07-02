import { defineConfig, devices } from '@playwright/test';

const baseURL = process.env.BASE_URL || 'http://localhost:3000';

export default defineConfig({
  testDir: './e2e',
  timeout: 15_000,
  // These specs drive one shared backend/Postgres/Redis instance with real,
  // non-mocked ticket inventory (only two fixed categories, VIP/Standard --
  // see apps/backend/db/seeds/seed_tickets.sql) instead of a per-test
  // sandbox. Two tests -- even from different files -- resetting/holding
  // the same category truly concurrently would corrupt each other's counts,
  // so run everything on a single worker. test.describe.serial() inside the
  // individual spec files is an extra, explicit guard for tests within one
  // file that share a category.
  fullyParallel: false,
  workers: 1,
  retries: process.env.CI ? 1 : 0,
  reporter: 'html',
  use: {
    baseURL,
    trace: 'on-first-retry',
    headless: true,
  },
  projects: [
    { name: 'chromium', use: { ...devices['Desktop Chrome'] } },
  ],
  // Reuses an already-running `npm run dev:frontend` if present, otherwise starts one.
  webServer: {
    command: 'npm run dev:frontend',
    url: baseURL,
    reuseExistingServer: !process.env.CI,
    timeout: 60_000,
  },
});
