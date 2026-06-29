import { test, expect } from '@playwright/test';
import { addGraphNode, openSerialGraph } from './fixtures/app.helper';

const graphPanel = '[data-testid="serial-graph-panel"]';

test.describe('Functional Tests', () => {
  test.beforeEach(async ({ page }) => {
    await openSerialGraph(page);
  });

  test('should display serial graph panel when serial module is active', async ({ page }) => {
    await expect(page.locator(graphPanel)).toBeVisible();
    await expect(page.locator('[data-testid="serial-graph-node-palette"]')).toBeVisible();
    await expect(page.locator('[data-testid="serial-graph-provider-serial.physical"]')).toBeVisible();
    await expect(page.locator('[data-testid="serial-graph-provider-serial.virtual"]')).toBeVisible();
    await expect(page.locator('[data-testid="serial-graph-provider-serial.script.generator"]')).toBeVisible();
    await expect(page.locator('[data-testid="serial-graph-provider-serial.monitor"]')).toBeVisible();
  });

  test('should keep graph canvas and node workbench separate', async ({ page }) => {
    const canvas = page.locator('[data-testid="serial-graph-canvas"]');
    const workbench = page.locator('[data-testid="serial-graph-node-workbench"]');
    await expect(canvas).toBeVisible();
    await expect(workbench).toBeVisible();
    await expect(page.locator('[data-testid="serial-graph-node-content"]')).toHaveCount(0);

    await addGraphNode(page, 'serial.script.generator', 'node-1');

    await expect(canvas).toBeVisible();
    await expect(workbench).toBeVisible();
    await expect(page.locator('[data-testid="serial-graph-node-content"]')).toBeVisible();

    const canvasBox = await canvas.boundingBox();
    const workbenchBox = await workbench.boundingBox();
    expect(canvasBox).not.toBeNull();
    expect(workbenchBox).not.toBeNull();
    expect(canvasBox!.height).toBeGreaterThanOrEqual(120);
    expect(workbenchBox!.height).toBeGreaterThan(100);
  });

  test('should have proper layout dimensions', async ({ page }) => {
    await expect(page.locator('.serial-view__sidebar')).toHaveCount(0);

    const mainArea = page.locator('.serial-view__main');
    await expect(mainArea).toBeVisible();
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

  test('should render titlebar without obsolete macOS padding shim', async ({ page }) => {
    await expect(page.locator('.app-shell__titlebar')).toBeVisible();
    const paddingTop = await page.locator('.app-shell').evaluate(el => getComputedStyle(el).paddingTop);
    expect(paddingTop).toBe('0px');
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

  test('should display serial node configuration fields after adding a serial node', async ({ page }) => {
    await addGraphNode(page, 'serial.physical', 'node-1');
    await expect(page.locator('[data-testid="serial-graph-node-content"]')).toBeVisible();
    await expect(page.locator('[data-testid="serial-graph-config-portName"]')).toBeVisible();
    await expect(page.locator('[data-testid="serial-graph-config-baudRate"]')).toBeVisible();
    await expect(page.locator('[data-testid="serial-graph-config-dataBits"]')).toBeVisible();
    await expect(page.locator('[data-testid="serial-graph-config-stopBits"]')).toBeVisible();
    await expect(page.locator('[data-testid="serial-graph-config-parity"]')).toBeVisible();
    await expect(page.locator('[data-testid="serial-graph-config-flowMode"]')).toBeVisible();
  });
});
