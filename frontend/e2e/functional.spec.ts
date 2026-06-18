import { test, expect } from '@playwright/test';

test.describe('Functional Tests', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/');
    await page.waitForTimeout(1000);
  });

  test('should display port config panel when serial icon is clicked', async ({ page }) => {
    // Click serial icon
    const serialIcon = page.locator('.activity-bar__item').first();
    await serialIcon.click();
    await page.waitForTimeout(500);

    // Check port config form is visible
    const portConfigForm = page.locator('.port-config-form');
    await expect(portConfigForm).toBeVisible();

    // Check form elements
    const portSelect = page.locator('text=端口').locator('..').locator('.n-base-selection');
    await expect(portSelect).toBeVisible();

    const baudRateSelect = page.locator('text=波特率').locator('..').locator('.n-base-selection');
    await expect(baudRateSelect).toBeVisible();

    const openButton = page.locator('button:has-text("打开串口")');
    await expect(openButton).toBeVisible();
    await expect(openButton).toBeDisabled(); // Disabled until port is selected
  });

  test('should show empty state message initially', async ({ page }) => {
    const serialIcon = page.locator('.activity-bar__item').first();
    await serialIcon.click();
    await page.waitForTimeout(500);

    const emptyMessage = page.locator('text=在左侧配置并打开串口');
    await expect(emptyMessage).toBeVisible();
  });

  test('should have proper layout dimensions', async ({ page }) => {
    const serialIcon = page.locator('.activity-bar__item').first();
    await serialIcon.click();
    await page.waitForTimeout(500);

    // Check sidebar width
    const sidebar = page.locator('.serial-view__sidebar');
    const sidebarBox = await sidebar.boundingBox();
    expect(sidebarBox?.width).toBeCloseTo(280, 5);

    // Check main area takes remaining space
    const mainArea = page.locator('.serial-view__main');
    const mainBox = await mainArea.boundingBox();
    expect(mainBox).toBeTruthy();
    expect(mainBox!.width).toBeGreaterThan(500);
  });

  test('should have correct dark theme colors', async ({ page }) => {
    const serialIcon = page.locator('.activity-bar__item').first();
    await serialIcon.click();
    await page.waitForTimeout(500);

    // Check sidebar background
    const sidebar = page.locator('.serial-view__sidebar');
    const bgColor = await sidebar.evaluate(el => getComputedStyle(el).backgroundColor);
    expect(bgColor).toBe('rgb(37, 37, 38)'); // #252526

    // Check border color
    const borderColor = await sidebar.evaluate(el => getComputedStyle(el).borderRightColor);
    expect(borderColor).toBe('rgb(45, 45, 45)'); // #2d2d2d
  });

  test('should have window control button safe area', async ({ page }) => {
    const appShell = page.locator('.app-shell');
    const paddingTop = await appShell.evaluate(el => getComputedStyle(el).paddingTop);
    expect(paddingTop).toBe('50px');
  });

  test('should be responsive to window resize', async ({ page }) => {
    const serialIcon = page.locator('.activity-bar__item').first();
    await serialIcon.click();
    await page.waitForTimeout(500);

    // Initial size
    let appShell = await page.locator('.app-shell').boundingBox();
    expect(appShell?.width).toBeCloseTo(1280, 5);

    // Resize
    await page.setViewportSize({ width: 1600, height: 900 });
    await page.waitForTimeout(500);

    // New size
    appShell = await page.locator('.app-shell').boundingBox();
    expect(appShell?.width).toBeCloseTo(1600, 5);

    // Sidebar should maintain fixed width
    const sidebar = page.locator('.serial-view__sidebar');
    const sidebarBox = await sidebar.boundingBox();
    expect(sidebarBox?.width).toBeCloseTo(280, 5);
  });

  test('should display all form fields', async ({ page }) => {
    const serialIcon = page.locator('.activity-bar__item').first();
    await serialIcon.click();
    await page.waitForTimeout(500);

    // Check all labels
    await expect(page.locator('text=端口')).toBeVisible();
    await expect(page.locator('text=波特率')).toBeVisible();
    await expect(page.locator('text=数据位')).toBeVisible();
    await expect(page.locator('text=停止位')).toBeVisible();
    await expect(page.locator('text=校验')).toBeVisible();
    await expect(page.locator('text=流控')).toBeVisible();

    // Check refresh button
    await expect(page.locator('button:has-text("刷新")')).toBeVisible();
  });
});
