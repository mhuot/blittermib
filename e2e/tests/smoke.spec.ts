import { test, expect } from '@playwright/test';

// Baseline browser coverage: the landing page renders, the typeahead
// returns a seeded hit, and selecting it navigates. These don't depend
// on the search-stall fix — they guard the search surface as a whole.
test.describe('smoke', () => {
  test('landing page renders the search UI', async ({ page }) => {
    await page.goto('/');
    await expect(page.locator('.hero-search-input')).toBeVisible();
    await expect(page.locator('.hero-brand')).toContainText('blittermib');
  });

  test('typeahead shows a matching symbol', async ({ page }) => {
    await page.goto('/');
    await page.locator('.hero-search-input').fill('ifInOctets');
    const results = page.locator('#hero-results');
    await expect(results).toHaveAttribute('data-state', 'visible');
    await expect(results.locator('.hero-result').first()).toContainText('ifInOctets');
  });

  test('selecting a hit navigates to its symbol', async ({ page }) => {
    await page.goto('/');
    await page.locator('.hero-search-input').fill('ifInOctets');
    const first = page.locator('#hero-results .hero-result').first();
    await expect(first).toBeVisible();
    await first.click();
    await expect(page).toHaveURL(/\/m\/IF-MIB\//);
  });
});
