import { test, expect } from '@playwright/test';
import { startVirtualPair, stopVirtualPair, writeToPort, VirtualPair } from './fixtures/vserial.helper';
import { generateModbusFrame, generateAA55Frame } from './fixtures/patterns';


// These tests require the real Wails backend; skipped under static vite-preview CI mode.
const hasBackend = process.env.MOCKTRUE_E2E_BACKEND === '1';

test.describe('Protocol Parse E2E', () => {
  let virtualPair: VirtualPair;

  test.beforeAll(() => {
    test.skip(!hasBackend, 'skipped: set MOCKTRUE_E2E_BACKEND=1 and run wails3 task dev');
    virtualPair = startVirtualPair();
  });

  test.afterAll(() => {
    if (virtualPair) stopVirtualPair(virtualPair);
  });

  test('should parse Modbus RTU frame', async ({ page }) => {
    await page.goto('/');
    await page.waitForSelector('.activity-bar');

    // Open port
    const serialIcon = page.locator('.activity-bar__item').first();
    await serialIcon.click();

    const portSelect = page.locator('select').first();
    await portSelect.selectOption(virtualPair.port2);

    const baudSelect = page.locator('select').nth(1);
    await baudSelect.selectOption('115200');

    const openButton = page.locator('button:has-text("打开")');
    await openButton.click();
    await page.waitForTimeout(500);

    // Switch to protocol config
    const protocolButton = page.locator('button:has-text("协议")');
    await protocolButton.click();

    // Select Modbus RTU template
    const templateSelect = page.locator('select').last();
    await templateSelect.selectOption('Modbus RTU');

    // Apply template
    const applyButton = page.locator('button:has-text("应用")');
    await applyButton.click();

    // Write Modbus frame
    const modbusFrame = generateModbusFrame(1, 3, Buffer.from([0x00, 0x00, 0x00, 0x0A]));
    writeToPort(virtualPair.port1, modbusFrame.toString('binary'));

    await page.waitForTimeout(1000);

    // Verify parsed fields
    const parsedFields = page.locator('.parsed-fields');
    await expect(parsedFields).toBeVisible();

    const fieldsText = await parsedFields.textContent();
    expect(fieldsText).toContain('SlaveAddr');
    expect(fieldsText).toContain('FuncCode');
  });

  test('should parse AA55 custom frame', async ({ page }) => {
    await page.goto('/');
    await page.waitForSelector('.activity-bar');

    // Open port
    const serialIcon = page.locator('.activity-bar__item').first();
    await serialIcon.click();

    const portSelect = page.locator('select').first();
    await portSelect.selectOption(virtualPair.port2);

    const baudSelect = page.locator('select').nth(1);
    await baudSelect.selectOption('115200');

    const openButton = page.locator('button:has-text("打开")');
    await openButton.click();
    await page.waitForTimeout(500);

    // Switch to protocol config
    const protocolButton = page.locator('button:has-text("协议")');
    await protocolButton.click();

    // Select AA55 template
    const templateSelect = page.locator('select').last();
    await templateSelect.selectOption('AA55 自定义帧');

    // Apply template
    const applyButton = page.locator('button:has-text("应用")');
    await applyButton.click();

    // Write AA55 frame
    const aa55Frame = generateAA55Frame(0x01, Buffer.from([0x02, 0x03]));
    writeToPort(virtualPair.port1, aa55Frame.toString('binary'));

    await page.waitForTimeout(1000);

    // Verify parsed fields
    const parsedFields = page.locator('.parsed-fields');
    await expect(parsedFields).toBeVisible();

    const fieldsText = await parsedFields.textContent();
    expect(fieldsText).toContain('cmd');
  });
});
