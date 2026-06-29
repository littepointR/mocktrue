import { test, expect } from '@playwright/test';
import { injectWailsMock } from './fixtures/wails-mock.helper';
import { emitSerialData, openPort } from './fixtures/app.helper';

// Legacy tests for the removed serial tab/operation-panel UI. Replace with graph-node flow tests before re-enabling.
test.describe.skip('Serial Basic E2E legacy serial-tab flow', () => {
  test.beforeEach(async ({ page }) => {
    await injectWailsMock(page);
  });

  test('should open port and send ASCII data', async ({ page }) => {
    await openPort(page, '/tmp/ttyV0');

    const tabConfig = page.locator('.serial-tab-config');
    await expect(tabConfig).toBeVisible();
    await expect(tabConfig).toContainText('/tmp/ttyV0');
    await tabConfig.getByRole('button', { name: /串口配置/ }).click();
    await expect(tabConfig).toContainText('115200');
    await expect(tabConfig).toContainText('数据位');

    await page.getByPlaceholder('输入要发送的数据').fill('hello');
    await page.getByRole('button', { name: '发送' }).click();

    await expect(page.locator('.data-display')).toBeVisible();
    await expect(page.locator('.stats-panel')).toContainText('TX: 5 字节', { timeout: 10000 });
  });

  test('should edit serial parameters inside an opened tab', async ({ page }) => {
    await openPort(page, '/tmp/ttyV0');

    const tabConfig = page.locator('.serial-tab-config');
    await tabConfig.getByRole('button', { name: /串口配置/ }).click();

    await tabConfig.locator('.serial-tab-config__field').filter({ hasText: '波特率' }).locator('.n-base-selection').click();
    await page.locator('.n-base-select-option').filter({ hasText: '9600' }).click();
    await page.getByRole('button', { name: '应用配置' }).click();

    await expect(tabConfig).toContainText('9600 bps', { timeout: 10000 });
    await tabConfig.getByRole('button', { name: /串口配置/ }).click();
    await expect(tabConfig.locator('.serial-tab-config__field').filter({ hasText: '波特率' }).locator('.n-base-selection')).toContainText('9600');
  });

  test('should show received data from mocked runtime events', async ({ page }) => {
    await openPort(page, '/tmp/ttyV0');

    await emitSerialData(page, 'port-1', 'test data');

    await expect(page.locator('.data-display')).toContainText('test data', { timeout: 10000 });
  });

  test('should show received data from Wails base64 byte events', async ({ page }) => {
    await openPort(page, '/tmp/ttyV0');

    await emitSerialData(page, 'port-1', 'base64 data', 'base64');

    await expect(page.locator('.data-display')).toContainText('base64 data', { timeout: 10000 });
  });

  test('should toggle receive timestamps', async ({ page }) => {
    await openPort(page, '/tmp/ttyV0');

    await emitSerialData(page, 'port-1', 'timestamped');

    const display = page.locator('.data-display');
    await expect(display).toContainText('timestamped', { timeout: 10000 });
    await expect(display.locator('.ascii-timestamp')).toBeVisible();

    await page.locator('.data-display__timestamp-control').getByRole('switch').click();
    await expect(display.locator('.ascii-timestamp')).toHaveCount(0);
    await expect(display).toContainText('timestamped');
  });

  test('should toggle receive auto scroll', async ({ page }) => {
    await openPort(page, '/tmp/ttyV0');

    const scrollContainer = page.locator('.data-display__content');
    const autoScrollSwitch = page.locator('.data-display__autoscroll-control').getByRole('switch');
    await expect(autoScrollSwitch).toBeVisible();

    await autoScrollSwitch.click();
    await scrollContainer.evaluate((el) => { el.scrollTop = 0; });
    await emitSerialData(page, 'port-1', `${Array.from({ length: 80 }, (_, i) => `locked-${i}`).join('\n')}\n`);
    await expect(page.locator('.data-display')).toContainText('locked-79', { timeout: 10000 });
    await expect(scrollContainer).toHaveJSProperty('scrollTop', 0);

    await autoScrollSwitch.click();
    await emitSerialData(page, 'port-1', `${Array.from({ length: 80 }, (_, i) => `follow-${i}`).join('\n')}\n`);
    await expect(page.locator('.data-display')).toContainText('follow-79', { timeout: 10000 });
    await expect.poll(async () => scrollContainer.evaluate(el => el.scrollTop)).toBeGreaterThan(0);
  });

  test('should show RX/TX statistics', async ({ page }) => {
    await openPort(page, '/tmp/ttyV0');

    await emitSerialData(page, 'port-1', 'rx');
    await page.getByPlaceholder('输入要发送的数据').fill('hello');
    await page.getByRole('button', { name: '发送' }).click();

    await expect(page.locator('.stats-panel')).toContainText('RX: 2 字节', { timeout: 500 });
    await expect(page.locator('.stats-panel')).toContainText('TX: 5 字节', { timeout: 10000 });
  });

  test('should reset RX and TX counters', async ({ page }) => {
    await openPort(page, '/tmp/ttyV0');

    await emitSerialData(page, 'port-1', 'rx');
    await page.getByPlaceholder('输入要发送的数据').fill('tx');
    await page.getByRole('button', { name: '发送' }).click();

    const statsPanel = page.locator('.stats-panel');
    await expect(statsPanel).toContainText('RX: 2 字节', { timeout: 10000 });
    await expect(statsPanel).toContainText('TX: 2 字节', { timeout: 10000 });

    await page.getByRole('button', { name: '复位计数' }).click();
    await expect(statsPanel).toContainText('RX: 0 字节', { timeout: 10000 });
    await expect(statsPanel).toContainText('TX: 0 字节', { timeout: 10000 });
  });

  test('should show send history queue and resend by clicking history item', async ({ page }) => {
    await openPort(page, '/tmp/ttyV0');

    const editor = page.locator('.send-panel__editor textarea');
    const history = page.locator('.send-panel__history');
    await expect(editor).toBeVisible();

    const editorBox = await editor.boundingBox();
    expect(editorBox).not.toBeNull();
    expect(editorBox!.height).toBeGreaterThan(70);

    await editor.fill('repeat');
    await page.getByRole('button', { name: '发送' }).click();

    const historyItem = history.locator('.send-panel__history-item').filter({ hasText: 'repeat' });
    await expect(historyItem).toBeVisible();
    await expect(editor).toHaveValue('repeat');
    await expect(page.locator('.stats-panel')).toContainText('TX: 6 字节', { timeout: 10000 });

    await page.getByRole('button', { name: '发送' }).click();
    await expect(historyItem).toHaveCount(1);
    await expect(page.locator('.stats-panel')).toContainText('TX: 12 字节', { timeout: 10000 });

    await historyItem.click();
    await expect(historyItem).toHaveCount(1);
    await expect(page.locator('.stats-panel')).toContainText('TX: 18 字节', { timeout: 10000 });
  });

  test('should format HEX input and convert content when switching send modes', async ({ page }) => {
    await openPort(page, '/tmp/ttyV0');

    const editor = page.locator('.send-panel__editor textarea');
    const modeSelect = page.locator('.send-panel__mode .n-base-selection');

    await editor.fill('Hi');
    await modeSelect.click();
    await page.locator('.n-base-select-option').filter({ hasText: 'HEX' }).click();
    await expect(editor).toHaveValue('48 69');

    await editor.fill('48656c6C6f');
    await expect(editor).toHaveValue('48 65 6c 6c 6f');

    await page.getByRole('button', { name: '发送' }).click();
    await expect(page.locator('.stats-panel')).toContainText('TX: 5 字节', { timeout: 10000 });

    await modeSelect.click();
    await page.locator('.n-base-select-option').filter({ hasText: 'ASCII' }).click();
    await expect(editor).toHaveValue('Hello');
  });

  test('should resize receive and send areas by dragging divider', async ({ page }) => {
    await openPort(page, '/tmp/ttyV0');

    const display = page.locator('.serial-tab-content__display');
    const send = page.locator('.serial-tab-content__send');
    const divider = page.locator('.serial-tab-content__resize-handle');

    await expect(divider).toBeVisible();
    const displayBefore = await display.boundingBox();
    const sendBefore = await send.boundingBox();
    const dividerBox = await divider.boundingBox();

    expect(displayBefore).not.toBeNull();
    expect(sendBefore).not.toBeNull();
    expect(dividerBox).not.toBeNull();

    await page.mouse.move(dividerBox!.x + dividerBox!.width / 2, dividerBox!.y + dividerBox!.height / 2);
    await page.mouse.down();
    await page.mouse.move(dividerBox!.x + dividerBox!.width / 2, dividerBox!.y - 80);
    await page.mouse.up();

    const displayAfter = await display.boundingBox();
    const sendAfter = await send.boundingBox();

    expect(displayAfter).not.toBeNull();
    expect(sendAfter).not.toBeNull();
    expect(displayAfter!.height).toBeLessThan(displayBefore!.height - 40);
    expect(sendAfter!.height).toBeGreaterThan(sendBefore!.height + 40);
  });
});
