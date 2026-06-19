import { test, expect } from '@playwright/test';
import { openPort } from './fixtures/app.helper';
import { injectWailsMock } from './fixtures/wails-mock.helper';

test.describe('Timed Send and Quick Buttons E2E', () => {
  test.beforeEach(async ({ page }) => {
    await injectWailsMock(page);
  });

  test('should send data at timed intervals and stop when disabled', async ({ page }) => {
    await openPort(page, '/tmp/ttyV0');

    await page.getByPlaceholder('输入要发送的数据').fill('tick');
    await page.getByLabel('发送间隔').fill('10');

    const autoSendSwitch = page.locator('.send-panel__auto-send').getByRole('switch');
    await expect(autoSendSwitch).toBeVisible();
    await autoSendSwitch.click();

    await expect(page.locator('.stats-panel')).toContainText(/TX: ([4-9]\d|[1-9]\d{2,}) 字节/, { timeout: 2000 });

    await autoSendSwitch.click();
    await page.waitForTimeout(100);
    const txText = await page.locator('.stats-panel').textContent();
    await page.waitForTimeout(450);
    await expect(page.locator('.stats-panel')).toHaveText(txText ?? '');
  });

  test('should use quick buttons', async () => {
    test.skip(true, 'quick button UI is not implemented in the current serial panel');
  });
});
