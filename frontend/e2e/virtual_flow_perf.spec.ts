import { test, expect } from '@playwright/test';
import { startVirtualPair, stopVirtualPair, writeToPort, VirtualPair } from './fixtures/vserial.helper';

const hasBackend = process.env.MOCKTRUE_E2E_BACKEND === '1';

test.describe('Virtual Serial and Flow Control E2E', () => {
  let virtualPair: VirtualPair;

  test.beforeAll(() => {
    test.skip(!hasBackend, 'skipped: set MOCKTRUE_E2E_BACKEND=1 and run wails3 task dev');
    virtualPair = startVirtualPair();
  });

  test.afterAll(() => {
    if (virtualPair) stopVirtualPair(virtualPair);
  });

  test('should create virtual pair from UI', async ({ page }) => {
    await page.goto('/');
    await page.waitForSelector('.activity-bar');

    const serialIcon = page.locator('.activity-bar__item').first();
    await serialIcon.click();

    const virtualButton = page.locator('button:has-text("虚拟串口")');
    if (await virtualButton.isVisible()) {
      await virtualButton.click();
      await page.waitForTimeout(1000);

      const portSelect = page.locator('select').first();
      const options = await portSelect.locator('option').allTextContents();
      expect(options.some(opt => opt.includes('ttyV'))).toBeTruthy();
    }
  });

  test('should handle large data throughput', async ({ page }) => {
    await page.goto('/');
    await page.waitForSelector('.activity-bar');

    const serialIcon = page.locator('.activity-bar__item').first();
    await serialIcon.click();

    await page.locator('select').first().selectOption(virtualPair.port2);
    await page.locator('select').nth(1).selectOption('115200');
    await page.locator('button:has-text("打开")').click();
    await page.waitForTimeout(500);

    const largeData = 'A'.repeat(1024 * 1024);
    writeToPort(virtualPair.port1, largeData);

    await page.waitForTimeout(2000);

    const statsPanel = page.locator('.stats-panel');
    const statsText = await statsPanel.textContent();
    expect(statsText).toContain('RX:');

    const rxMatch = statsText?.match(/RX:\s*(\d+)/);
    if (rxMatch) {
      const rxBytes = parseInt(rxMatch[1]);
      expect(rxBytes).toBeGreaterThan(1000000);
    }
  });

  test('should show flow control options', async ({ page }) => {
    await page.goto('/');
    await page.waitForSelector('.activity-bar');

    const serialIcon = page.locator('.activity-bar__item').first();
    await serialIcon.click();

    const flowSelect = page.locator('select').last();
    await expect(flowSelect).toBeVisible();

    const options = await flowSelect.locator('option').allTextContents();
    expect(options.some(opt => opt.includes('无'))).toBeTruthy();
    expect(options.some(opt => opt.includes('硬件'))).toBeTruthy();
    expect(options.some(opt => opt.includes('软件'))).toBeTruthy();
  });
});
