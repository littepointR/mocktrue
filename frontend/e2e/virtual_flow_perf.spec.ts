import { test, expect } from '@playwright/test';
import {
  createVirtualPort,
  emitSerialData,
  openPort,
  openSerialModule,
  openSidebarTab,
  selectNaiveOption,
} from './fixtures/app.helper';
import { injectWailsMock } from './fixtures/wails-mock.helper';

test.describe('Virtual Serial and Flow Control E2E', () => {
  test.beforeEach(async ({ page }) => {
    await injectWailsMock(page);
  });

  test('should create virtual port from UI', async ({ page }) => {
    await createVirtualPort(page, 'vport-e2e', 'ttyE2E');

    await openSidebarTab(page, '串口');
    await page.getByRole('button', { name: '刷新' }).click();
    await selectNaiveOption(page, '.port-config-form', '/tmp/ttyE2E');
    await expect(page.locator('.port-config-form').getByRole('button', { name: '打开串口' })).toBeEnabled();
  });

  test('should reuse existing tab when opening the same virtual port again', async ({ page }) => {
    await createVirtualPort(page, 'vport-reopen', 'ttyReopen');

    await openSidebarTab(page, '串口');
    await page.getByRole('button', { name: '刷新' }).click();
    await selectNaiveOption(page, '.port-config-form', '/tmp/ttyReopen');
    await page.locator('.port-config-form').getByRole('button', { name: '打开串口' }).click();
    await expect(page.locator('.n-tabs-tab').filter({ hasText: '/tmp/ttyReopen' })).toHaveCount(1);

    await openSidebarTab(page, '串口');
    await selectNaiveOption(page, '.port-config-form', '/tmp/ttyReopen');
    await page.locator('.port-config-form').getByRole('button', { name: '打开串口' }).click();
    await expect(page.locator('.n-tabs-tab').filter({ hasText: '/tmp/ttyReopen' })).toHaveCount(1);
    await expect(page.locator('.n-alert').filter({ hasText: 'port already open' })).toHaveCount(0);
  });

  test('should handle large data throughput', async ({ page }) => {
    await openPort(page, '/tmp/ttyV0');

    const largeData = 'A'.repeat(32 * 1024);
    await emitSerialData(page, 'port-1', largeData);

    await expect(page.locator('.stats-panel')).toContainText('RX: 32768 字节', { timeout: 10000 });
    await expect(page.locator('.ascii-content')).toContainText('A'.repeat(200));
  });

  test('should show flow control options', async ({ page }) => {
    await openSerialModule(page);
    await openSidebarTab(page, '串口');

    await selectNaiveOption(page, '.port-config-form', '硬件', 5);
    await selectNaiveOption(page, '.port-config-form', '软件', 5);
    await selectNaiveOption(page, '.port-config-form', '无', 5);
  });
});
