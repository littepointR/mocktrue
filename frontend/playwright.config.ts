import { defineConfig } from '@playwright/test';

const isWailsMode = process.env.PORTWEAVE_E2E_MODE === 'wails';
const vitePort = Number(process.env.WAILS_VITE_PORT) || 9245;
const baseURL = isWailsMode ? `http://localhost:${vitePort}` : 'http://localhost:4173';
const wailsDevCommand = process.platform === 'win32'
  ? 'set "PATH=%PATH%;%USERPROFILE%\\go\\bin" && wails3 task dev'
  : 'PATH="$PATH:$HOME/go/bin" wails3 task dev';

export default defineConfig({
  testDir: './e2e',
  timeout: 30000,
  retries: 0,
  fullyParallel: false,
  workers: isWailsMode ? 1 : undefined,
  use: {
    baseURL,
    trace: 'on-first-retry',
    screenshot: 'only-on-failure',
  },
  projects: [
    {
      name: 'chromium',
      use: { browserName: 'chromium' },
    },
  ],
  webServer: {
    command: isWailsMode
      ? wailsDevCommand
      : 'pnpm run build && pnpm run preview',
    port: isWailsMode ? vitePort : 4173,
    timeout: isWailsMode ? 120000 : 60000,
    reuseExistingServer: !isWailsMode,
  },
});
