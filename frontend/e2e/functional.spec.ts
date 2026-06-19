import { test, expect } from '@playwright/test';
import { openSerialModule, openSidebarTab } from './fixtures/app.helper';

test.describe('Functional Tests', () => {
  test.beforeEach(async ({ page }) => {
    await openSerialModule(page);
  });

  test('should display port config panel when serial module is active', async ({ page }) => {
    const sidebar = page.locator('.sidebar');
    await expect(sidebar.locator('.sidebar__item').filter({ hasText: '打开串口' })).toBeVisible();
    await expect(sidebar.locator('.sidebar__item').filter({ hasText: '添加虚拟串口' })).toBeVisible();
    await expect(sidebar.locator('.sidebar__item').filter({ hasText: '添加串口桥接' })).toBeVisible();

    await openSidebarTab(page, '串口');
    const portConfigForm = page.locator('.port-config-form');
    await expect(portConfigForm).toBeVisible();

    await expect(portConfigForm).toContainText('端口');
    await expect(portConfigForm).toContainText('波特率');
    await expect(portConfigForm.locator('.n-base-selection')).toHaveCount(6);
    await expect(page.getByRole('button', { name: '刷新' })).toBeVisible();

    const openButton = portConfigForm.getByRole('button', { name: '打开串口' });
    await expect(openButton).toBeVisible();
    await expect(openButton).toBeDisabled();
  });

  test('should keep operation pane and tab content area separate', async ({ page }) => {
    const operationPanel = page.locator('.serial-view__operation-panel');
    const mainArea = page.locator('.serial-view__main');
    await expect(operationPanel).toHaveCount(1);
    await expect(mainArea).toBeVisible();
    await expect(page.locator('.serial-view__empty')).toBeVisible();
    await expect(page.locator('.port-config-form')).toHaveCount(0);

    const sidebar = page.locator('.sidebar');
    await sidebar.locator('.sidebar__item').filter({ hasText: '打开串口' }).click();
    await expect(operationPanel.locator('.port-config-form')).toBeVisible();
    await expect(mainArea.locator('.serial-view__empty')).toBeVisible();
    let operationBox = await operationPanel.boundingBox();
    expect(operationBox?.width).toBeGreaterThan(250);

    await sidebar.locator('.sidebar__item').filter({ hasText: '打开串口' }).click();
    await expect(operationPanel.locator('.port-config-form')).toHaveCount(0);
    await expect(mainArea.locator('.serial-view__empty')).toBeVisible();
    operationBox = await operationPanel.boundingBox();
    expect(operationBox?.width ?? 0).toBeLessThan(2);

    await sidebar.locator('.sidebar__item').filter({ hasText: '添加虚拟串口' }).click();
    await expect(operationPanel.locator('.virtual-pair-panel')).toBeVisible();
    await expect(mainArea.locator('.serial-view__empty')).toBeVisible();

    await sidebar.locator('.sidebar__item').filter({ hasText: '添加虚拟串口' }).click();
    await expect(operationPanel.locator('.virtual-pair-panel')).toHaveCount(0);

    await sidebar.locator('.sidebar__item').filter({ hasText: '添加串口桥接' }).click();
    await expect(operationPanel.locator('.bridge-panel')).toBeVisible();
    await expect(mainArea.locator('.serial-view__empty')).toBeVisible();

    await sidebar.locator('.sidebar__item').filter({ hasText: '添加串口桥接' }).click();
    await expect(operationPanel.locator('.bridge-panel')).toHaveCount(0);
    await expect(mainArea.locator('.serial-view__empty')).toBeVisible();
  });

  test('should have proper layout dimensions', async ({ page }) => {
    await expect(page.locator('.serial-view__sidebar')).toHaveCount(0);

    const mainArea = page.locator('.serial-view__main');
    const mainBox = await mainArea.boundingBox();
    expect(mainBox).toBeTruthy();
    expect(mainBox!.width).toBeGreaterThan(700);
  });

  test('should have correct dark theme colors', async ({ page }) => {
    const sidebar = page.locator('.sidebar');
    const bgColor = await sidebar.evaluate(el => getComputedStyle(el).backgroundColor);
    expect(bgColor).toBe('rgb(37, 37, 38)'); // #252526

    const borderColor = await sidebar.evaluate(el => getComputedStyle(el).borderRightColor);
    expect(borderColor).toBe('rgb(45, 45, 45)'); // #2d2d2d
  });

  test('should have window control button safe area', async ({ page }) => {
    const appShell = page.locator('.app-shell');
    const paddingTop = await appShell.evaluate(el => getComputedStyle(el).paddingTop);
    expect(paddingTop).toBe('50px');
  });

  test('should be responsive to window resize', async ({ page }) => {
    let appShell = await page.locator('.app-shell').boundingBox();
    expect(appShell?.width).toBeCloseTo(1280, 5);

    await page.setViewportSize({ width: 1600, height: 900 });
    await expect(page.locator('.app-shell')).toHaveJSProperty('clientWidth', 1600);

    appShell = await page.locator('.app-shell').boundingBox();
    expect(appShell?.width).toBeCloseTo(1600, 5);

    const sidebar = page.locator('.sidebar');
    const sidebarBox = await sidebar.boundingBox();
    expect(sidebarBox?.width).toBeCloseTo(200, 5);
  });

  test('should display all form fields', async ({ page }) => {
    await openSidebarTab(page, '串口');
    await expect(page.locator('.sidebar__item').filter({ hasText: '打开串口' })).toBeVisible();
    await expect(page.locator('text=端口')).toBeVisible();
    await expect(page.locator('text=波特率')).toBeVisible();
    await expect(page.locator('text=数据位')).toBeVisible();
    await expect(page.locator('text=停止位')).toBeVisible();
    await expect(page.locator('text=校验')).toBeVisible();
    await expect(page.locator('text=流控')).toBeVisible();

    await expect(page.getByRole('button', { name: '刷新' })).toBeVisible();
  });
});
