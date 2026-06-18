import { defineConfig } from '@playwright/test';

export default defineConfig({
  testDir: './e2e',
  timeout: 30000,
  retries: 0,
  use: {
    // Vite preview serves the built frontend on localhost:4173
    baseURL: 'http://localhost:4173',
    trace: 'on-first-retry',
  },
  projects: [
    {
      name: 'chromium',
      use: { browserName: 'chromium' },
    },
  ],
  // Start vite preview to serve the built frontend for UI E2E tests.
  // The Wails @wailsio/runtime calls will fail in this mode (no backend),
  // but UI shell/layout/interaction tests can run against the static build.
  webServer: {
    command: 'pnpm run build && pnpm run preview',
    port: 4173,
    timeout: 60000,
    reuseExistingServer: true,
  },
});
