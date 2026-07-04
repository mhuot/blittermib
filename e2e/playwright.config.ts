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
    reuseExistingServer: !process.env.CI,
    // `go run` compiles on first launch; give a cold CI build room.
    timeout: 120_000,
    stdout: 'pipe',
    stderr: 'pipe',
  },
});
