import { test, expect } from '@playwright/test';
import { startVirtualPair, stopVirtualPair, writeToPort, VirtualPair } from './fixtures/vserial.helper';
import { generatePattern, bufferToHex } from './fixtures/patterns';

// These tests require the real Wails backend; skipped under static vite-preview CI mode.
const hasBackend = process.env.MOCKTRUE_E2E_BACKEND === '1';

test.describe('HEX View E2E', () => {
  let virtualPair: VirtualPair;

  test.beforeAll(() => {
    test.skip(!hasBackend, 'skipped: set MOCKTRUE_E2E_BACKEND=1 and run wails3 task dev');
    virtualPair = startVirtualPair();
  });

  test.afterAll(() => {
    if (virtualPair) stopVirtualPair(virtualPair);
  });

  test('should switch to HEX classic view', async ({ page }) => {
    await page.goto('/');
    await page.waitForSelector('.activity-bar');

    const serialIcon = page.locator('.activity-bar__item').first();
    await serialIcon.click();

    const portSelect = page.locator('select').first();
    await portSelect.selectOption(virtualPair.port2);

    const baudSelect = page.locator('select').nth(1);
    await baudSelect.selectOption('115200');

    const openButton = page.locator('button:has-text("打开")');
    await openButton.click();
    await page.waitForTimeout(500);

    const viewModeSelect = page.locator('select').nth(2);
    await viewModeSelect.selectOption('hexClassic');

    const pattern = generatePattern(32);
    writeToPort(virtualPair.port1, pattern.toString('binary'));

    await page.waitForTimeout(1000);

    const hexView = page.locator('.hex-classic');
    await expect(hexView).toBeVisible();

    const hexData = page.locator('.hex-data').first();
    await expect(hexData).toBeVisible();
  });

  test('should display correct HEX format', async ({ page }) => {
    await page.goto('/');
    await page.waitForSelector('.activity-bar');

    const serialIcon = page.locator('.activity-bar__item').first();
    await serialIcon.click();

    const portSelect = page.locator('select').first();
    await portSelect.selectOption(virtualPair.port2);

    const baudSelect = page.locator('select').nth(1);
    await baudSelect.selectOption('115200');

    const openButton = page.locator('button:has-text("打开")');
    await openButton.click();
    await page.waitForTimeout(500);

    const viewModeSelect = page.locator('select').nth(2);
    await viewModeSelect.selectOption('hexClassic');

    const testData = Buffer.from([0xAA, 0x55, 0x01, 0x02, 0x03]);
    writeToPort(virtualPair.port1, testData.toString('binary'));

    await page.waitForTimeout(1000);

    const hexData = page.locator('.hex-data').first();
    const hexText = await hexData.textContent();
    expect(hexText).toContain('aa 55 01 02 03');
  });

  test('should show ASCII representation', async ({ page }) => {
    await page.goto('/');
    await page.waitForSelector('.activity-bar');

    const serialIcon = page.locator('.activity-bar__item').first();
    await serialIcon.click();

    const portSelect = page.locator('select').first();
    await portSelect.selectOption(virtualPair.port2);

    const baudSelect = page.locator('select').nth(1);
    await baudSelect.selectOption('115200');

    const openButton = page.locator('button:has-text("打开")');
    await openButton.click();
    await page.waitForTimeout(500);

    const viewModeSelect = page.locator('select').nth(2);
    await viewModeSelect.selectOption('hexClassic');

    writeToPort(virtualPair.port1, 'Hello');

    await page.waitForTimeout(1000);

    const hexAscii = page.locator('.hex-ascii').first();
    const asciiText = await hexAscii.textContent();
    expect(asciiText).toContain('Hello');
  });
});
