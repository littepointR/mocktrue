import { test, expect } from '@playwright/test';
import { openPorts } from './fixtures/app.helper';
import { injectWailsMock } from './fixtures/wails-mock.helper';

test.describe('Multi-Port Parallel E2E', () => {
  test.beforeEach(async ({ page }) => {
    await injectWailsMock(page);
  });

  test('should open multiple ports in tabs', async ({ page }) => {
    await openPorts(page, ['/tmp/ttyV0', '/tmp/ttyV1']);

    const tabs = page.locator('.n-tabs-tab');
    await expect(tabs.filter({ hasText: '/tmp/ttyV0' })).toBeVisible();
    await expect(tabs.filter({ hasText: '/tmp/ttyV1' })).toBeVisible();

    await page.getByPlaceholder('输入要发送的数据').fill('port2 data');
    await page.getByRole('button', { name: '发送' }).click();
    await expect(page.locator('.stats-panel')).toContainText('TX: 10 字节', { timeout: 10000 });
  });

  test('should switch between tabs', async ({ page }) => {
    await openPorts(page, ['/tmp/ttyV0', '/tmp/ttyV1']);

    await page.locator('.n-tabs-tab').filter({ hasText: '/tmp/ttyV0' }).click();
    await page.getByPlaceholder('输入要发送的数据').fill('port1');
    await page.getByRole('button', { name: '发送' }).click();
    await expect(page.locator('.stats-panel')).toContainText('TX: 5 字节', { timeout: 10000 });
  });

  test('should close tab', async ({ page }) => {
    await openPorts(page, ['/tmp/ttyV0']);

    await page.locator('.n-tabs-tab__close').first().click();

    await expect(page.locator('.n-tabs-tab').filter({ hasText: '/tmp/ttyV0' })).toHaveCount(0);
    await expect(page.locator('.serial-view__empty')).toBeVisible();
    await expect(page.locator('.serial-view__main')).toBeVisible();
  });

  test('should split editor groups by dragging a tab to the right edge', async ({ page }) => {
    await openPorts(page, ['/tmp/ttyV0', '/tmp/ttyV1']);

    const tab = page.locator('.editor-tab').filter({ hasText: '/tmp/ttyV1' });
    const main = page.locator('.serial-view__main');
    await expect(tab).toBeVisible();

    const tabBox = await tab.boundingBox();
    const mainBox = await main.boundingBox();
    expect(tabBox).not.toBeNull();
    expect(mainBox).not.toBeNull();

    await page.mouse.move(tabBox!.x + tabBox!.width / 2, tabBox!.y + tabBox!.height / 2);
    await page.mouse.down();
    await page.mouse.move(mainBox!.x + mainBox!.width - 12, mainBox!.y + mainBox!.height / 2);
    await page.mouse.up();

    await expect(page.locator('.editor-group')).toHaveCount(2);
    await expect(page.locator('.editor-group').nth(0).locator('.editor-tab').filter({ hasText: '/tmp/ttyV0' })).toBeVisible();
    await expect(page.locator('.editor-group').nth(1).locator('.editor-tab').filter({ hasText: '/tmp/ttyV1' })).toBeVisible();
  });

  test('should show skeleton preview while dragging a tab to split', async ({ page }) => {
    await openPorts(page, ['/tmp/ttyV0', '/tmp/ttyV1']);

    const tab = page.locator('.editor-tab').filter({ hasText: '/tmp/ttyV1' });
    const main = page.locator('.serial-view__main');
    await expect(tab).toBeVisible();

    const tabBox = await tab.boundingBox();
    const mainBox = await main.boundingBox();
    expect(tabBox).not.toBeNull();
    expect(mainBox).not.toBeNull();

    await page.mouse.move(tabBox!.x + tabBox!.width / 2, tabBox!.y + tabBox!.height / 2);
    await page.mouse.down();
    await page.mouse.move(mainBox!.x + mainBox!.width - 12, mainBox!.y + mainBox!.height / 2);

    const preview = page.locator('.editor-drop-preview');
    await expect(preview).toBeVisible();
    await expect(preview).toHaveAttribute('data-edge', 'right');

    const previewBox = await preview.boundingBox();
    expect(previewBox).not.toBeNull();
    expect(previewBox!.width).toBeGreaterThan(80);
    expect(previewBox!.height).toBeGreaterThan(120);
    expect(previewBox!.x).toBeGreaterThanOrEqual(mainBox!.x + mainBox!.width / 2);

    await page.mouse.up();
    await expect(preview).toHaveCount(0);
  });

  test('should split an existing editor group in another direction', async ({ page }) => {
    await openPorts(page, ['/tmp/ttyV0', '/tmp/ttyV1', '/tmp/ttyV2']);

    const tab1 = page.locator('.editor-tab').filter({ hasText: '/tmp/ttyV1' });
    const main = page.locator('.serial-view__main');
    const tab1Box = await tab1.boundingBox();
    const mainBox = await main.boundingBox();
    expect(tab1Box).not.toBeNull();
    expect(mainBox).not.toBeNull();

    await page.mouse.move(tab1Box!.x + tab1Box!.width / 2, tab1Box!.y + tab1Box!.height / 2);
    await page.mouse.down();
    await page.mouse.move(mainBox!.x + mainBox!.width - 12, mainBox!.y + mainBox!.height / 2);
    await page.mouse.up();
    await expect(page.locator('.editor-group')).toHaveCount(2);

    const leftGroup = page.locator('.editor-group').filter({ has: page.locator('.editor-tab', { hasText: '/tmp/ttyV2' }) }).first();
    const tab2 = leftGroup.locator('.editor-tab').filter({ hasText: '/tmp/ttyV2' });
    const leftGroupBox = await leftGroup.boundingBox();
    const tab2Box = await tab2.boundingBox();
    expect(leftGroupBox).not.toBeNull();
    expect(tab2Box).not.toBeNull();

    await page.mouse.move(tab2Box!.x + tab2Box!.width / 2, tab2Box!.y + tab2Box!.height / 2);
    await page.mouse.down();
    await page.mouse.move(leftGroupBox!.x + leftGroupBox!.width / 2, leftGroupBox!.y + leftGroupBox!.height - 12);
    await page.mouse.up();

    await expect(page.locator('.editor-group')).toHaveCount(3);
    await expect(page.locator('.editor-group').filter({ has: page.locator('.editor-tab', { hasText: '/tmp/ttyV0' }) })).toBeVisible();
    await expect(page.locator('.editor-group').filter({ has: page.locator('.editor-tab', { hasText: '/tmp/ttyV1' }) })).toBeVisible();
    await expect(page.locator('.editor-group').filter({ has: page.locator('.editor-tab', { hasText: '/tmp/ttyV2' }) })).toBeVisible();
  });
});
