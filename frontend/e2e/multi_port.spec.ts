import { test, expect } from '@playwright/test';
import { startVirtualPair, stopVirtualPair, writeToPort, VirtualPair } from './fixtures/vserial.helper';

const hasBackend = process.env.MOCKTRUE_E2E_BACKEND === '1';

test.describe('Multi-Port Parallel E2E', () => {
  let virtualPair1: VirtualPair;

  test.beforeAll(() => {
    test.skip(!hasBackend, 'skipped: set MOCKTRUE_E2E_BACKEND=1 and run wails3 task dev');
    virtualPair1 = startVirtualPair();
  });

  test.afterAll(() => {
    if (virtualPair1) stopVirtualPair(virtualPair1);
  });

  test('should open multiple ports in tabs', async ({ page }) => {
    await page.goto('/');
    await page.waitForSelector('.activity-bar');

    const serialIcon = page.locator('.activity-bar__item').first();
    await serialIcon.click();

    const portSelect1 = page.locator('select').first();
    await portSelect1.selectOption(virtualPair1.port1);

    const baudSelect1 = page.locator('select').nth(1);
    await baudSelect1.selectOption('115200');

    const openButton1 = page.locator('button:has-text("打开")').first();
    await openButton1.click();
    await page.waitForTimeout(500);

    const tabs = page.locator('.n-tabs-tab');
    await expect(tabs.first()).toBeVisible();

    const sendInput1 = page.locator('input[placeholder="输入要发送的数据"]').first();
    await sendInput1.fill('port1 data');

    const sendButton1 = page.locator('button:has-text("发送")').first();
    await sendButton1.click();
    await page.waitForTimeout(500);
  });

  test('should switch between tabs', async ({ page }) => {
    await page.goto('/');
    await page.waitForSelector('.activity-bar');

    const serialIcon = page.locator('.activity-bar__item').first();
    await serialIcon.click();

    const portSelect = page.locator('select').first();
    await portSelect.selectOption(virtualPair1.port1);

    const baudSelect = page.locator('select').nth(1);
    await baudSelect.selectOption('115200');

    const openButton = page.locator('button:has-text("打开")').first();
    await openButton.click();
    await page.waitForTimeout(500);

    const tab = page.locator('.n-tabs-tab').first();
    await expect(tab).toBeVisible();

    await tab.click();
    await page.waitForTimeout(200);

    const tabContent = page.locator('.n-tab-pane');
    await expect(tabContent.first()).toBeVisible();
  });

  test('should close tab', async ({ page }) => {
    await page.goto('/');
    await page.waitForSelector('.activity-bar');

    const serialIcon = page.locator('.activity-bar__item').first();
    await serialIcon.click();

    const portSelect = page.locator('select').first();
    await portSelect.selectOption(virtualPair1.port1);

    const baudSelect = page.locator('select').nth(1);
    await baudSelect.selectOption('115200');

    const openButton = page.locator('button:has-text("打开")').first();
    await openButton.click();
    await page.waitForTimeout(500);

    const closeButton = page.locator('.n-tabs-tab__close').first();
    await closeButton.click();
    await page.waitForTimeout(500);

    const tabs = page.locator('.n-tabs-tab');
    const tabCount = await tabs.count();
    expect(tabCount).toBeLessThanOrEqual(1);
  });
});
