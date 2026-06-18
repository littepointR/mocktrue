import { test, expect } from '@playwright/test';
import path from 'path';
import fs from 'fs';

const screenshotsDir = path.join(process.cwd(), 'test-results', 'screenshots');
if (!fs.existsSync(screenshotsDir)) {
  fs.mkdirSync(screenshotsDir, { recursive: true });
}

test.describe('Diagnostic UI Tests', () => {
  test('capture initial render screenshot', async ({ page }) => {
    const consoleMessages: string[] = [];
    const errors: Error[] = [];

    page.on('console', msg => {
      consoleMessages.push(`[${msg.type()}] ${msg.text()}`);
    });
    page.on('pageerror', err => {
      errors.push(err);
    });

    await page.goto('/');
    await page.waitForTimeout(2000); // Let everything settle

    // Take a full-page screenshot
    await page.screenshot({
      path: path.join(screenshotsDir, '01-initial-render.png'),
      fullPage: true,
    });

    console.log('=== Console messages ===');
    consoleMessages.forEach(m => console.log(m));

    console.log('=== Page errors ===');
    errors.forEach(e => console.log(`ERROR: ${e.message}\n${e.stack}`));

    // Capture the full HTML for inspection
    const html = await page.content();
    fs.writeFileSync(path.join(screenshotsDir, '01-initial.html'), html);

    // Print all visible text to diagnose what's actually rendered
    const bodyText = await page.locator('body').innerText();
    console.log('=== Visible text ===');
    console.log(bodyText);

    // Check critical elements
    const appShellExists = await page.locator('.app-shell').count();
    const activityBarExists = await page.locator('.activity-bar').count();
    const sidebarExists = await page.locator('.sidebar').count();
    const editorGroupsExists = await page.locator('.editor-groups').count();
    const panelExists = await page.locator('.panel').count();
    const statusBarExists = await page.locator('.status-bar').count();

    console.log('=== Element counts ===');
    console.log(`.app-shell: ${appShellExists}`);
    console.log(`.activity-bar: ${activityBarExists}`);
    console.log(`.sidebar: ${sidebarExists}`);
    console.log(`.editor-groups: ${editorGroupsExists}`);
    console.log(`.panel: ${panelExists}`);
    console.log(`.status-bar: ${statusBarExists}`);

    // Check activity bar contents
    const iconCount = await page.locator('.activity-bar__item').count();
    console.log(`.activity-bar__item count: ${iconCount}`);

    if (iconCount > 0) {
      const iconText = await page.locator('.activity-bar__item').first().innerText();
      const iconTitle = await page.locator('.activity-bar__item').first().getAttribute('title');
      console.log(`First icon text: "${iconText}", title: "${iconTitle}"`);
    }
  });

  test('click serial icon and capture screenshot', async ({ page }) => {
    await page.goto('/');
    await page.waitForTimeout(1000);

    // Click serial icon
    const serialIcon = page.locator('.activity-bar__item').first();
    await serialIcon.click();
    await page.waitForTimeout(2000); // Let UI settle

    await page.screenshot({
      path: path.join(screenshotsDir, '02-serial-active.png'),
      fullPage: true,
    });

    // Check what's now visible in the editor area (use correct class)
    const editorGroupsExists = await page.locator('.editor-groups').count();
    if (editorGroupsExists > 0) {
      const editorContent = await page.locator('.editor-groups').innerHTML();
      fs.writeFileSync(path.join(screenshotsDir, '02-editor-content.html'), editorContent);
    }

    // Check for serial view
    const serialViewCount = await page.locator('.serial-view').count();
    const portConfigCount = await page.locator('.port-config').count();
    const dataDisplayCount = await page.locator('.data-display').count();
    const sendPanelCount = await page.locator('.send-panel').count();
    const statsPanelCount = await page.locator('.stats-panel').count();

    console.log('=== After clicking serial ===');
    console.log(`.serial-view: ${serialViewCount}`);
    console.log(`.port-config: ${portConfigCount}`);
    console.log(`.data-display: ${dataDisplayCount}`);
    console.log(`.send-panel: ${sendPanelCount}`);
    console.log(`.stats-panel: ${statsPanelCount}`);

    // Capture sidebar contents
    const sidebarText = await page.locator('.sidebar').innerText();
    console.log(`Sidebar text: "${sidebarText}"`);
  });

  test('check for JavaScript errors', async ({ page }) => {
    const errors: string[] = [];
    const resources404: string[] = [];

    page.on('pageerror', err => errors.push(err.message));
    page.on('console', msg => {
      if (msg.type() === 'error') errors.push(`[console.error] ${msg.text()}`);
    });
    page.on('response', response => {
      if (response.status() === 404) {
        resources404.push(response.url());
      }
    });

    await page.goto('/');
    await page.waitForTimeout(2000);

    // Try clicking serial
    await page.locator('.activity-bar__item').first().click();
    await page.waitForTimeout(2000);

    if (resources404.length > 0) {
      console.log('=== 404 resources ===');
      resources404.forEach(u => console.log(u));
    }

    // Filter out non-app errors (like 404 from Wails runtime which is expected in static mode)
    const appErrors = errors.filter(e =>
      !e.includes('Failed to load resource') ||
      // Only fail on truly bad 404s for app assets
      false
    );

    if (appErrors.length > 0) {
      console.log('=== JavaScript errors ===');
      appErrors.forEach(e => console.log(e));
    }

    // Allow Wails-runtime-related 404s but reject real app errors
    const realErrors = errors.filter(e => !e.includes('Failed to load resource'));
    expect(realErrors).toEqual([]);
  });

  test('check window dimensions and viewport', async ({ page }) => {
    await page.goto('/');
    await page.waitForTimeout(1000);

    const viewport = page.viewportSize();
    console.log(`Viewport: ${viewport?.width}x${viewport?.height}`);

    const bodyDims = await page.locator('body').evaluate(el => ({
      width: el.clientWidth,
      height: el.clientHeight,
      scrollWidth: el.scrollWidth,
      scrollHeight: el.scrollHeight,
    }));
    console.log(`Body: ${JSON.stringify(bodyDims)}`);

    const appShellDims = await page.locator('.app-shell').evaluate(el => ({
      width: el.clientWidth,
      height: el.clientHeight,
    }));
    console.log(`App shell: ${JSON.stringify(appShellDims)}`);

    const activityBarDims = await page.locator('.activity-bar').evaluate(el => ({
      width: el.clientWidth,
      height: el.clientHeight,
    }));
    console.log(`Activity bar: ${JSON.stringify(activityBarDims)}`);
  });

  test('verify responsive layout', async ({ page }) => {
    await page.goto('/');
    await page.waitForTimeout(1000);

    // 检查初始尺寸
    const appShell = await page.locator('.app-shell').boundingBox();
    const viewport = page.viewportSize();

    expect(appShell?.width).toBeCloseTo(viewport!.width, 5);
    expect(appShell?.height).toBeCloseTo(viewport!.height, 5);

    // 改变窗口大小
    await page.setViewportSize({ width: 1600, height: 900 });
    await page.waitForTimeout(500);

    const newAppShell = await page.locator('.app-shell').boundingBox();
    expect(newAppShell?.width).toBeCloseTo(1600, 5);
    expect(newAppShell?.height).toBeCloseTo(900, 5);
  });
});
