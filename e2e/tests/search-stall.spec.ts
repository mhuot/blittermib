import { test, expect, type Page, type Request } from '@playwright/test';
import { clientTiming } from './helpers';

// Regression coverage for the as-you-type search stall (no42-org#87).
//
// The bug: the typeahead fired one FTS request per keystroke with no
// minimum length and no cancellation, so a burst of expensive
// short-prefix queries piled up on the store's single connection and
// stalled the site. These tests assert the client half of the fix,
// which the Go/API tests can't reach: gating, the OID exemption, and
// cancellation of superseded / dismissed requests.
//
// The debounce and the gate threshold are read from palette.js's own
// constants (see helpers.ts) so a client tuning change can't silently
// invalidate an assertion — queries are sized off the live MIN_QUERY_LEN
// and waits off the live DEBOUNCE_MS.

const SEARCH = /\/api\/v1\/search\?/;
const isSearch = (r: Request) => SEARCH.test(r.url());
const isAbort = (r: Request) =>
  isSearch(r) && !!r.failure()?.errorText?.includes('ERR_ABORTED');

// A seeded symbol (storetest.SeedIFMIB), so any prefix of it matches —
// a gated prefix still returns nothing precisely because it is gated,
// not because it fails to match.
const SEED_SYMBOL = 'ifInOctets';
const belowGate = (min: number) => SEED_SYMBOL.slice(0, Math.max(1, min - 1));
const atGate = (min: number) => SEED_SYMBOL.slice(0, min);

// Delay each search request so it is still in flight when the next
// keystroke (or a dismiss) should cancel it — the localhost harness
// otherwise round-trips in well under a millisecond. The hold is applied
// before route.continue() forwards the request, so the browser sees a
// pending request throughout.
function slowSearch(page: Page, ms: number) {
  return page.route(SEARCH, async (route) => {
    await new Promise((r) => setTimeout(r, ms));
    try {
      await route.continue();
    } catch {
      // These tests deliberately abort in-flight searches; continuing a
      // request the page already cancelled (or one left pending when the
      // context closes) rejects — that's expected, not a test failure.
    }
  });
}

test.describe('search-stall regression', () => {
  test('a below-threshold query is gated and fires no request', async ({ page, request, baseURL }) => {
    const { debounceMs, minQueryLen } = await clientTiming(request, baseURL!);
    const fired: string[] = [];
    page.on('request', (r) => isSearch(r) && fired.push(r.url()));
    await page.goto('/');
    // A prefix one shorter than the gate — must be declined even though
    // it prefixes a real symbol.
    await page.locator('.hero-search-input').pressSequentially(belowGate(minQueryLen));
    // Wait several debounce windows so a regressed gate's request would
    // have fired by now — scaled from the client's own constant.
    await page.waitForTimeout(debounceMs * 4 + 200);
    expect(fired, 'sub-threshold query must be gated client-side').toHaveLength(0);
  });

  test('an at-threshold query does fetch and renders hits', async ({ page, request, baseURL }) => {
    const { minQueryLen } = await clientTiming(request, baseURL!);
    const fired: string[] = [];
    page.on('request', (r) => isSearch(r) && fired.push(r.url()));
    await page.goto('/');
    await page.locator('.hero-search-input').pressSequentially(atGate(minQueryLen));
    await expect.poll(() => fired.length).toBeGreaterThan(0);
    await expect(page.locator('#hero-results')).toHaveAttribute('data-state', 'visible');
  });

  test('a one-character OID query is exempt and fetches', async ({ page }) => {
    const fired: string[] = [];
    page.on('request', (r) => isSearch(r) && fired.push(r.url()));
    await page.goto('/');
    // "1" is a single character but OID-shaped, so it takes the cheap
    // indexed path and must NOT be gated regardless of the text threshold.
    await page.locator('.hero-search-input').pressSequentially('1');
    await expect.poll(() => fired.length).toBeGreaterThan(0);
  });

  test('superseded requests are aborted, not left to complete', async ({ page, request, baseURL }) => {
    const { debounceMs, minQueryLen } = await clientTiming(request, baseURL!);
    const keyDelay = debounceMs + 80; // each keystroke's request fires before the next
    await slowSearch(page, keyDelay * 6); // response outlives several keystrokes
    const aborted: string[] = [];
    page.on('requestfailed', (r) => isAbort(r) && aborted.push(r.url()));
    await page.goto('/');
    // A couple of characters past the gate: an early keystroke fires a
    // request a later one supersedes while it's still in flight.
    await page
      .locator('.hero-search-input')
      .pressSequentially(atGate(minQueryLen + 2), { delay: keyDelay });
    await expect
      .poll(() => aborted.length, {
        message: 'superseding keystrokes must cancel in-flight searches',
      })
      .toBeGreaterThan(0);
  });

  // Both search surfaces must cancel a pending request when dismissed —
  // the modal via reset() (unconditional), the landing hero via clear()
  // once results are showing. Without it a slow response re-opens the
  // dismissed dropdown and the request lingers on the lone connection.
  const surfaces = {
    modal: {
      goto: '/m/IF-MIB/1.3.6.1.2.1.2.2.1.10',
      input: '.palette-input',
      option: '.palette-item',
      open: async (page: Page) => {
        await page.locator('[data-palette-toggle]').click();
        await expect(page.locator('.palette-overlay')).toHaveAttribute('data-state', 'visible');
      },
    },
    hero: {
      goto: '/',
      input: '.hero-search-input',
      option: '.hero-result',
      open: async () => {},
    },
  } as const;

  for (const [name, s] of Object.entries(surfaces)) {
    test(`dismissing the ${name} after results are shown aborts the in-flight request`, async ({ page, request, baseURL }) => {
      const { debounceMs, minQueryLen } = await clientTiming(request, baseURL!);
      const hold = debounceMs * 12;
      await slowSearch(page, hold);
      const aborted: string[] = [];
      page.on('requestfailed', (r) => isAbort(r) && aborted.push(r.url()));

      await page.goto(s.goto);
      await s.open(page);
      const input = page.locator(s.input);

      // Round 1: let a first query complete so a result is showing (the
      // hero only cancels on Escape while its dropdown is visible).
      await input.pressSequentially(atGate(minQueryLen));
      await expect(page.locator(s.option).first()).toBeVisible({ timeout: hold + 5000 });

      // Round 2: subscribe BEFORE the keystroke so the debounced request
      // can't fire in the gap before we listen, then dismiss it in flight.
      const inflight = page.waitForRequest(SEARCH);
      await input.press(SEED_SYMBOL[minQueryLen] ?? 'x'); // extend the query -> a fresh request
      await inflight;
      await page.keyboard.press('Escape'); // dismiss -> cancelInFlight()

      await expect
        .poll(() => aborted.length, {
          message: `dismissing the ${name} must cancel the pending search`,
        })
        .toBeGreaterThan(0);
    });
  }

  // Mid-fetch dismiss (Escape before any result renders) — asserted on the
  // modal only. The hero input is type=search: on Chromium the browser's
  // native Escape-clears-field fires an input event that aborts the request
  // itself, masking whether the app cancelled it. So the hero's mid-fetch
  // cancellation (palette.js Escape -> ctl.clear regardless of visibility)
  // is real but only observable on engines without that native behavior
  // (e.g. Firefox); a Chromium-only assertion here would pass either way.
  // The modal input carries no such native behavior, so this has teeth.
  test('dismissing the modal mid-fetch aborts and never opens the dropdown', async ({ page, request, baseURL }) => {
    const s = surfaces.modal;
    const { debounceMs, minQueryLen } = await clientTiming(request, baseURL!);
    await slowSearch(page, debounceMs * 20); // response never arrives before we dismiss
    const aborted: string[] = [];
    page.on('requestfailed', (r) => isAbort(r) && aborted.push(r.url()));

    await page.goto(s.goto);
    await s.open(page);
    const input = page.locator(s.input);

    const inflight = page.waitForRequest(SEARCH);
    await input.pressSequentially(atGate(minQueryLen)); // fires a request, held; nothing rendered
    await inflight;
    await page.keyboard.press('Escape'); // dismiss mid-fetch -> reset() -> cancelInFlight

    await expect
      .poll(() => aborted.length, {
        message: 'mid-fetch dismiss of the modal must cancel the request',
      })
      .toBeGreaterThan(0);
    // The held response must never populate the dropdown the user dismissed.
    await expect(page.locator(s.option)).toHaveCount(0);
  });
});
