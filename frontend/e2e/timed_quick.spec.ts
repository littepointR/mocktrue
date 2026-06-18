import { test, expect } from '@playwright/test';
import { startVirtualPair, stopVirtualPair, writeToPort, VirtualPair } from './fixtures/vserial.helper';


// These tests require the real Wails backend; skipped under static vite-preview CI mode.
const hasBackend = process.env.MOCKTRUE_E2E_BACKEND === '1';

test.describe('Timed Send and Quick Buttons E2E', () => {
  let virtualPair: VirtualPair;

  test.beforeAll(() => {
    test.skip(!hasBackend, 'skipped: set MOCKTRUE_E2E_BACKEND=1 and run wails3 task dev');
    virtualPair = startVirtualPair();
  });

  test.afterAll(() => {
    if (virtualPair) stopVirtualPair(virtualPair);
  });

  test('should send data at timed intervals', async ({ page }) => {
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

    // Switch to timed send mode
    const timedButton = page.locator('button:has-text("定时发送")');
    await timedButton.click();

    // Set interval
    const intervalInput = page.locator('input[placeholder="间隔(ms)"]');
    await intervalInput.fill('100');

    // Set content
    const contentInput = page.locator('input[placeholder="发送内容"]');
    await contentInput.fill('timed');

    // Start timed send
    const startButton = page.locator('button:has-text("开始")');
    await startButton.click();

    // Wait for multiple sends
    await page.waitForTimeout(1500);

    // Stop timed send
    const stopButton = page.locator('button:has-text("停止")');
    await stopButton.click();

    // Verify send count
    const sendCount = page.locator('.send-count');
    const countText = await sendCount.textContent();
    const count = parseInt(countText?.match(/\d+/)?.[0] || '0');
    expect(count).toBeGreaterThanOrEqual(10);
  });

  test('should use quick buttons', async ({ page }) => {
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

    // Add quick button
    const addButton = page.locator('button:has-text("添加快捷按钮")');
    await addButton.click();

    // Fill quick button form
    const labelInput = page.locator('input[placeholder="标签"]');
    await labelInput.fill('Test Button');

    const contentInput = page.locator('input[placeholder="内容"]');
    await contentInput.fill('quick test');

    const saveButton = page.locator('button:has-text("保存")');
    await saveButton.click();

    // Click quick button
    const quickButton = page.locator('button:has-text("Test Button")');
    await quickButton.click();

    await page.waitForTimeout(500);

    // Verify data was sent
    const dataDisplay = page.locator('.data-display');
    const content = await dataDisplay.textContent();
    expect(content).toContain('quick test');
  });
});
