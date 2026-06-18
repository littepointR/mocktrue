import { test, expect } from '@playwright/test';

// Shell smoke tests that run against the static frontend build (vite preview).
// These verify the VS Code-style layout renders without needing the Wails backend.

test.describe('App Shell Smoke', () => {
  test('should render the dark VS Code-style shell', async ({ page }) => {
    await page.goto('/');

    // The app shell should be visible
    await page.waitForSelector('.app-shell', { timeout: 10000 });

    // Activity bar (left icon column)
    await expect(page.locator('.activity-bar')).toBeVisible();

    // Sidebar
    await expect(page.locator('.sidebar')).toBeVisible();

    // Panel (bottom)
    await expect(page.locator('.panel')).toBeVisible();

    // Status bar (very bottom)
    await expect(page.locator('.status-bar')).toBeVisible();
  });

  test('should show the serial module icon in the activity bar', async ({ page }) => {
    await page.goto('/');
    await page.waitForSelector('.activity-bar', { timeout: 10000 });

    // The serial module icon should be present
    const serialIcon = page.locator('.activity-bar__item').first();
    await expect(serialIcon).toBeVisible();

    // Title attribute should mention serial debugging
    await expect(serialIcon).toHaveAttribute('title', '串口调试');
  });

  test('should display MockTrue version in the status bar', async ({ page }) => {
    await page.goto('/');
    await page.waitForSelector('.status-bar', { timeout: 10000 });

    const statusBar = page.locator('.status-bar');
    await expect(statusBar).toContainText('MockTrue v0.1.0');
  });

  test('should show sidebar views when activating the serial module', async ({ page }) => {
    await page.goto('/');
    await page.waitForSelector('.activity-bar', { timeout: 10000 });

    // Click the serial module icon
    const serialIcon = page.locator('.activity-bar__item').first();
    await serialIcon.click();

    // The sidebar should now show the serial module's views
    await expect(page.locator('.sidebar__title')).toContainText('串口调试');
    await expect(page.locator('.sidebar__item')).toContainText('连接');
  });

  test('should update status bar with active module id', async ({ page }) => {
    await page.goto('/');
    await page.waitForSelector('.activity-bar', { timeout: 10000 });

    // Initially no active module
    const statusBar = page.locator('.status-bar');
    await expect(statusBar.locator('.status-bar__right')).toContainText('—');

    // Activate serial module
    const serialIcon = page.locator('.activity-bar__item').first();
    await serialIcon.click();

    // Status bar should now show the active module id
    await expect(statusBar.locator('.status-bar__right')).toContainText('serial');
  });

  test('should use a dark background color', async ({ page }) => {
    await page.goto('/');
    await page.waitForSelector('body', { timeout: 10000 });

    const body = page.locator('body');
    const backgroundColor = await body.evaluate(
      el => getComputedStyle(el).backgroundColor
    );
    // #1e1e1e = rgb(30, 30, 30)
    expect(backgroundColor).toBe('rgb(30, 30, 30)');
  });
});
