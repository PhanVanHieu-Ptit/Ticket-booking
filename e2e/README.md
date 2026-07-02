# E2E Test Suite

Playwright specs that exercise the running app (real backend, real Postgres,
real Redis — nothing mocked except the browser's own network layer for
failure-injection scenarios) to prove the system's core concurrency/
reliability guarantees hold in practice, not just in unit tests.

## Running the suite

1. Start infra + backend + frontend once, as usual:
   ```bash
   npm run infra:up
   npm run dev
   ```
2. In another terminal, run the suite:
   ```bash
   npm run test:e2e        # interactive/local run
   npm run test:e2e:ci     # headless, list + HTML reporter (what CI runs)
   ```

If the frontend isn't already running, Playwright's `webServer` config
(`playwright.config.ts`) starts one for you automatically.

The suite mutates real ticket inventory through a test-only endpoint
(`POST /api/v1/test/reset-inventory`, only mounted when `APP_ENV != production`
— see `apps/backend/internal/handlers/test_handler.go`). It runs on a single
Playwright worker (`workers: 1` in `playwright.config.ts`) because several
specs share the same two ticket categories (`VIP`, `Standard`) and reset them
to specific counts mid-test; true parallel execution across files would let
one spec's reset stomp another's. Each spec restores both categories to their
full seed counts (100 VIP / 400 Standard) in an `afterEach`, so re-running the
suite (or any single file) repeatedly never leaves depleted or "dirty"
inventory behind for the next run.

## Reading the HTML report

```bash
npx playwright show-report
```

opens an interactive report with a pass/fail timeline, and for every test:
the full trace (network requests, console logs, DOM snapshots at each step),
screenshots on failure, and — for retried/failed runs — a trace viewer you can
step through frame by frame. `npm run test:e2e:ci` also prints a `list`
summary straight to the terminal. In CI, the same HTML report is uploaded as
the `playwright-report` workflow artifact (`.github/workflows/e2e.yml`) —
download it from the failed run's Actions summary page.

## What each spec proves

| Spec | Claim being tested |
| --- | --- |
| `smoke.spec.ts` | The app boots and renders the event page at all — the baseline sanity check everything else assumes. |
| `spam-click.spec.ts` | Spam-clicking "Reserve" 10 times in a row (on a slow/throttled network) sends **exactly one** reservation request and consumes **exactly one** seat — the frontend's own click-guard prevents duplicate submissions, not just luck/network timing. |
| `race-condition.spec.ts` | The classic oversell scenario: with exactly **1** seat left, two independent browser sessions click "Reserve" at the same instant. Exactly one wins (gets a real hold with a countdown), exactly one loses (sees "sold out", no broken UI) — run once, then repeated 10 more times back-to-back to rule out a lucky non-collision. Also asserts directly against the backend's admin metrics that never more than 1 ticket is left `Holding` for a category reset to 1 seat, i.e. no oversell at the data layer, not just the UI. |
| `realtime-update.spec.ts` | Two independent sessions, no page reload: (1) when session B holds a ticket, session A's visible count updates on its own within 5s via the SSE push — proving live updates work, not just polling; (2) when session A's SSE connection is forcibly dropped mid-session, it falls back to REST polling ("Slow Mode"), and once the connection is restored it reconnects and resyncs to the correct count on its own — proving the reconnect/resync path is robust, not just the happy path. |
| `server-down.spec.ts` | For two failure modes (total network abort, and HTTP 500) and three moments (initial page load, mid-reservation, mid-payment): the app shows a friendly error instead of crashing to a blank screen, stays on a sane page/state, and — once the backend is back, **without a page reload** — either self-recovers (homepage) or lets the user successfully retry the exact same action (reserve, pay). |

## Known flaky/failing tests

None at the time of writing — see git history if a spec starts failing
consistently; check `git log -- apps/backend/internal/handlers/test_handler.go`
and `apps/frontend/src/hooks/useTicketAvailability.ts` for the fixes that
made `server-down.spec.ts` reliable (a stale `orders` row could reject a
retried purchase of a recycled test ticket ID, and the SSE `onopen` handler
could stop REST polling before the first successful fetch ever populated any
data).
