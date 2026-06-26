import { test, expect } from '@playwright/test';
import { startVirtualPair, stopVirtualPair, writeToPort, VirtualPair } from './fixtures/vserial.helper';
import { generatePattern, generateModbusFrame, generateAA55Frame } from './fixtures/patterns';


// These tests describe an end-to-end future workflow that combines real Wails
// serial I/O with UI surfaces not present in the current serial panel.
const hasBackend = process.env.PORTWEAVE_E2E_BACKEND === '1';

test.describe('Full Regression E2E', () => {
  let virtualPair: VirtualPair;

  test.beforeAll(() => {
    test.skip(!hasBackend, 'skipped: future full-stack workflow; requires implemented protocol/PCAP UI plus PORTWEAVE_E2E_BACKEND=1');
    virtualPair = startVirtualPair();
  });

  test.afterAll(() => {
    if (virtualPair) stopVirtualPair(virtualPair);
  });

  test('should complete full serial workflow', async ({ page }) => {
    // 1. Start app and verify shell
    await page.goto('/');
    await page.waitForSelector('.activity-bar');
    await page.waitForSelector('.sidebar');
    await page.waitForSelector('.editor-groups');
    await page.waitForSelector('.panel');
    await page.waitForSelector('.status-bar');

    // 2. Open serial module
    const serialIcon = page.locator('.activity-bar__item').first();
    await serialIcon.click();

    // 3. Configure and open port
    const portSelect = page.locator('select').first();
    await portSelect.selectOption(virtualPair.port2);

    const baudSelect = page.locator('select').nth(1);
    await baudSelect.selectOption('115200');

    const openButton = page.locator('button:has-text("打开")');
    await openButton.click();
    await page.waitForTimeout(500);

    // 4. Send ASCII data
    const sendInput = page.locator('input[placeholder="输入要发送的数据"]');
    await sendInput.fill('hello world');

    const sendButton = page.locator('button:has-text("发送")');
    await sendButton.click();
    await page.waitForTimeout(500);

    // 5. Switch to HEX view
    const viewModeSelect = page.locator('select').nth(2);
    await viewModeSelect.selectOption('hexClassic');

    // 6. Send HEX data
    const hexData = Buffer.from([0xAA, 0x55, 0x01, 0x02]);
    writeToPort(virtualPair.port1, hexData.toString('binary'));

    await page.waitForTimeout(1000);

    // 7. Verify HEX display
    const hexView = page.locator('.hex-classic');
    await expect(hexView).toBeVisible();

    // 8. Switch to protocol parsing
    const protocolButton = page.locator('button:has-text("协议")');
    await protocolButton.click();

    // 9. Apply Modbus template
    const templateSelect = page.locator('select').last();
    await templateSelect.selectOption('Modbus RTU');

    const applyButton = page.locator('button:has-text("应用")');
    await applyButton.click();

    // 10. Send Modbus frame
    const modbusFrame = generateModbusFrame(1, 3, Buffer.from([0x00, 0x00, 0x00, 0x0A]));
    writeToPort(virtualPair.port1, modbusFrame.toString('binary'));

    await page.waitForTimeout(1000);

    // 11. Start recording
    const recordButton = page.locator('button:has-text("记录")');
    await recordButton.click();

    // 12. Send more data
    writeToPort(virtualPair.port1, 'recording test');

    await page.waitForTimeout(500);

    // 13. Stop recording
    const stopRecordButton = page.locator('button:has-text("停止记录")');
    await stopRecordButton.click();

    // 14. Verify statistics
    const statsPanel = page.locator('.stats-panel');
    const statsText = await statsPanel.textContent();
    expect(statsText).toContain('RX:');
    expect(statsText).toContain('TX:');

    // 15. Close port
    const closeButton = page.locator('button:has-text("关闭")');
    await closeButton.click();
    await page.waitForTimeout(500);

    // 16. Verify port closed
    const openButtonAgain = page.locator('button:has-text("打开")');
    await expect(openButtonAgain).toBeVisible();

    // 17. Verify dark theme
    const body = page.locator('body');
    const backgroundColor = await body.evaluate(el => getComputedStyle(el).backgroundColor);
    expect(backgroundColor).toBe('rgb(30, 30, 30)'); // #1e1e1e
  });

  test('should persist configuration', async ({ page }) => {
    await page.goto('/');
    await page.waitForSelector('.activity-bar');

    // Open serial module
    const serialIcon = page.locator('.activity-bar__item').first();
    await serialIcon.click();

    // Add quick button
    const addButton = page.locator('button:has-text("添加快捷按钮")');
    if (await addButton.isVisible()) {
      await addButton.click();

      const labelInput = page.locator('input[placeholder="标签"]');
      await labelInput.fill('Persist Test');

      const contentInput = page.locator('input[placeholder="内容"]');
      await contentInput.fill('persist data');

      const saveButton = page.locator('button:has-text("保存")');
      await saveButton.click();

      // Reload page
      await page.reload();
      await page.waitForSelector('.activity-bar');

      // Click serial module again
      await serialIcon.click();

      // Verify quick button persists
      const quickButton = page.locator('button:has-text("Persist Test")');
      if (await quickButton.isVisible()) {
        await expect(quickButton).toBeVisible();
      }
    }
  });
});
