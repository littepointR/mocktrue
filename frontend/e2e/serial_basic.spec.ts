import { test, expect } from '@playwright/test';
import { startVirtualPair, stopVirtualPair, writeToPort, VirtualPair } from './fixtures/vserial.helper';

// These tests require the real Wails backend (wails3 task dev with a GUI
// environment). They are skipped under the static vite-preview CI mode,
// where @wailsio/runtime service calls are unavailable. Run locally with
// `wails3 task dev` + a display to exercise the full stack.

const hasBackend = process.env.MOCKTRUE_E2E_BACKEND === '1';

test.describe('Serial Basic E2E (requires Wails backend)', () => {
  test.beforeAll(() => {
    test.skip(!hasBackend, 'skipped: set MOCKTRUE_E2E_BACKEND=1 and run wails3 task dev');
  });

  let virtualPair: VirtualPair;

  test.beforeAll(() => {
    virtualPair = startVirtualPair();
  });

  test.afterAll(() => {
    if (virtualPair) stopVirtualPair(virtualPair);
  });

  test('should open port and send ASCII data', async ({ page }) => {
    await page.goto('/');
    await page.waitForSelector('.activity-bar');

    const serialIcon = page.locator('.activity-bar__item').first();
    await serialIcon.click();

    await page.waitForSelector('.port-config');
    await page.locator('select').first().selectOption(virtualPair.port1);
    await page.locator('select').nth(1).selectOption('115200');
    await page.locator('button:has-text("打开")').click();
    await page.waitForTimeout(500);

    await page.locator('input[placeholder="输入要发送的数据"]').fill('hello');
    await page.locator('button:has-text("发送")').click();
    await page.waitForTimeout(200);

    await expect(page.locator('.data-display')).toBeVisible();
  });

  test('should receive data and display it', async ({ page }) => {
    await page.goto('/');
    await page.waitForSelector('.activity-bar');

    const serialIcon = page.locator('.activity-bar__item').first();
    await serialIcon.click();

    await page.locator('select').first().selectOption(virtualPair.port2);
    await page.locator('select').nth(1).selectOption('115200');
    await page.locator('button:has-text("打开")').click();
    await page.waitForTimeout(500);

    writeToPort(virtualPair.port1, 'test data');
    await page.waitForTimeout(1000);

    const content = await page.locator('.data-display').textContent();
    expect(content).toContain('test data');
  });

  test('should show RX/TX statistics', async ({ page }) => {
    await page.goto('/');
    await page.waitForSelector('.activity-bar');

    const serialIcon = page.locator('.activity-bar__item').first();
    await serialIcon.click();

    await page.locator('select').first().selectOption(virtualPair.port1);
    await page.locator('select').nth(1).selectOption('115200');
    await page.locator('button:has-text("打开")').click();
    await page.waitForTimeout(500);

    await page.locator('input[placeholder="输入要发送的数据"]').fill('hello');
    await page.locator('button:has-text("发送")').click();
    await page.waitForTimeout(500);

    const statsText = await page.locator('.stats-panel').textContent();
    expect(statsText).toContain('RX:');
    expect(statsText).toContain('TX:');
  });
});
