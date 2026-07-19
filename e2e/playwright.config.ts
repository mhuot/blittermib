import { defineConfig, devices } from '@playwright/test';
import path from 'node:path';

// The app-under-test is the hermetic harness (cmd/e2e-harness): an
// in-process server seeded with a fixed slice of IF-MIB, so no smidump,
// no on-disk corpus, and no Docker are needed. Static assets are
// embedded in the binary, so these tests exercise the real palette.js.
const PORT = process.env.PORT ?? '8081';
const baseURL = `http://127.0.0.1:${PORT}`;

export default defineConfig({
  testDir: './tests',
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 0,
  reporter: process.env.CI ? [['list'], ['html', { open: 'never' }]] : 'list',
  use: {
    baseURL,
    trace: 'on-first-retry',
  },
  projects: [{ name: 'chromium', use: { ...devices['Desktop Chrome'] } }],
  webServer: {
    command: 'go run ./cmd/e2e-harness',
    cwd: path.resolve(__dirname, '..'),
    url: `${baseURL}/readyz`,
    env: { PORT },
    // Never reuse: the harness embeds its static assets, so a server left
    // running from an earlier run serves the palette.js/styles.css it was
    // built with. That silently voids the freshness invariant `make e2e`
    // sets up via prepare-assets + generate — and because helpers.ts reads
    // DEBOUNCE_MS/MIN_QUERY_LEN from that same stale server, the timing
    // envelope stays self-consistent while testing the wrong build. Paying
    // one `go run` per run is cheaper than chasing that.
    reuseExistingServer: false,
    // `go run` compiles on first launch; give a cold CI build room.
    timeout: 120_000,
    stdout: 'pipe',
    stderr: 'pipe',
  },
});
