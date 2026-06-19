import { test, expect } from '@playwright/test';
import { emitSerialData, openPort, switchDataView } from './fixtures/app.helper';
import { injectWailsMock } from './fixtures/wails-mock.helper';

test.describe('HEX View E2E', () => {
  test.beforeEach(async ({ page }) => {
    await injectWailsMock(page);
    await openPort(page, '/tmp/ttyV0');
  });

  test('should switch to HEX classic view', async ({ page }) => {
    await switchDataView(page, 'HEX 经典');
    await emitSerialData(page, 'port-1', 'Hello');

    const hexView = page.locator('.hex-classic');
    await expect(hexView).toBeVisible();

    const hexData = page.locator('.hex-data').first();
    await expect(hexData).toBeVisible();
  });

  test('should display correct HEX format', async ({ page }) => {
    await switchDataView(page, 'HEX 经典');
    await page.evaluate(() => {
      window.dispatchEvent(new CustomEvent('mocktrue:serial-data', {
        detail: { PortID: 'port-1', Data: [0xaa, 0x55, 0x01, 0x02, 0x03] },
      }));
    });

    const hexData = page.locator('.hex-data').first();
    const hexText = await hexData.textContent();
    expect(hexText).toContain('aa 55 01 02 03');
  });

  test('should show ASCII representation', async ({ page }) => {
    await switchDataView(page, 'HEX 经典');
    await emitSerialData(page, 'port-1', 'Hello');

    const hexAscii = page.locator('.hex-ascii').first();
    const asciiText = await hexAscii.textContent();
    expect(asciiText).toContain('Hello');
  });
});
