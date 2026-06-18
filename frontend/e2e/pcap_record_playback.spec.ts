import { test, expect } from '@playwright/test';
import { startVirtualPair, stopVirtualPair, writeToPort, VirtualPair } from './fixtures/vserial.helper';


// These tests require the real Wails backend; skipped under static vite-preview CI mode.
const hasBackend = process.env.MOCKTRUE_E2E_BACKEND === '1';

test.describe('PCAP Record and Playback E2E', () => {
  let virtualPair: VirtualPair;

  test.beforeAll(() => {
    test.skip(!hasBackend, 'skipped: set MOCKTRUE_E2E_BACKEND=1 and run wails3 task dev');
    virtualPair = startVirtualPair();
  });

  test.afterAll(() => {
    if (virtualPair) stopVirtualPair(virtualPair);
  });

  test('should record data to pcap file', async ({ page }) => {
    await page.goto('/');
    await page.waitForSelector('.activity-bar');

    // Open port
    const serialIcon = page.locator('.activity-bar__item').first();
    await serialIcon.click();

    const portSelect = page.locator('select').first();
    await portSelect.selectOption(virtualPair.port2);

    const baudSelect = page.locator('select').nth(1);
    await baudSelect.selectOption('115200');

    const openButton = page.locator('button:has-text("打开")');
    await openButton.click();
    await page.waitForTimeout(500);

    // Start recording
    const recordButton = page.locator('button:has-text("记录")');
    await recordButton.click();

    // Write data
    writeToPort(virtualPair.port1, 'test recording');

    await page.waitForTimeout(1000);

    // Stop recording
    const stopRecordButton = page.locator('button:has-text("停止记录")');
    await stopRecordButton.click();

    // Verify recording indicator
    const recordStatus = page.locator('.record-status');
    await expect(recordStatus).toContainText('已停止');
  });

  test('should playback recorded data', async ({ page }) => {
    await page.goto('/');
    await page.waitForSelector('.activity-bar');

    // Open port
    const serialIcon = page.locator('.activity-bar__item').first();
    await serialIcon.click();

    const portSelect = page.locator('select').first();
    await portSelect.selectOption(virtualPair.port2);

    const baudSelect = page.locator('select').nth(1);
    await baudSelect.selectOption('115200');

    const openButton = page.locator('button:has-text("打开")');
    await openButton.click();
    await page.waitForTimeout(500);

    // Start recording
    const recordButton = page.locator('button:has-text("记录")');
    await recordButton.click();

    // Write data
    writeToPort(virtualPair.port1, 'playback test');

    await page.waitForTimeout(500);

    // Stop recording
    const stopRecordButton = page.locator('button:has-text("停止记录")');
    await stopRecordButton.click();

    // Start playback
    const playbackButton = page.locator('button:has-text("回放")');
    await playbackButton.click();

    // Wait for playback
    await page.waitForTimeout(1000);

    // Verify playback status
    const playbackStatus = page.locator('.playback-status');
    await expect(playbackStatus).toContainText('回放完成');
  });
});
