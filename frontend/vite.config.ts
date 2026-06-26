import { defineConfig } from "vitest/config";
import vue from "@vitejs/plugin-vue";
import wails from "@wailsio/runtime/plugins/vite";

const coverageThreshold = (name: string, fallback: number) => {
  const value = Number(process.env[name]);
  return Number.isFinite(value) ? value : fallback;
};

// https://vitejs.dev/config/
export default defineConfig({
  server: {
    host: "127.0.0.1",
    port: Number(process.env.WAILS_VITE_PORT) || 9245,
    strictPort: true,
  },
  plugins: [vue(), wails("./bindings")],
  test: {
    environment: "jsdom",
    globals: true,
    include: ["src/**/*.{test,spec}.ts"],
    setupFiles: ["src/test/setup.ts"],
    coverage: {
      provider: "v8",
      all: true,
      include: ["src/**/*.{ts,vue}"],
      exclude: [
        "src/**/*.test.ts",
        "src/test/**",
        "src/main.ts",
        "src/vite-env.d.ts",
      ],
      reporter: ["text", "json-summary", "lcov"],
      thresholds: {
        // Baseline ratchet for the current all-source Vitest coverage report.
        // Raise these values as tests are added; target is 90%+.
        statements: coverageThreshold("VITEST_COVERAGE_STATEMENTS", 67),
        branches: coverageThreshold("VITEST_COVERAGE_BRANCHES", 59),
        functions: coverageThreshold("VITEST_COVERAGE_FUNCTIONS", 67),
        lines: coverageThreshold("VITEST_COVERAGE_LINES", 70),
      },
    },
  },
});
