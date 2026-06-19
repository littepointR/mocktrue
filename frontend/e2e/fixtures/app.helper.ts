import { expect, type Page } from '@playwright/test';

export async function openSerialModule(page: Page) {
  await page.goto('/');
  await expect(page.locator('.activity-bar')).toBeVisible({ timeout: 10000 });
  await page.locator('.activity-bar__item').first().click();
  await expect(page.locator('.serial-view')).toBeVisible();
}

export async function openSidebarTab(page: Page, tabText: string) {
  const optionText = tabText === '串口'
    ? '打开串口'
    : tabText === '虚拟'
      ? '添加虚拟串口'
      : tabText === '桥接'
        ? '添加串口桥接'
        : tabText;
  const panelSelector = tabText === '串口'
    ? '.port-config-form'
    : tabText === '虚拟'
      ? '.virtual-pair-panel'
      : tabText === '桥接'
        ? '.bridge-panel'
        : null;
  if (panelSelector && await page.locator(panelSelector).isVisible().catch(() => false)) {
    return;
  }
  const item = page.locator('.sidebar__item').filter({ hasText: optionText });
  await expect(item).toBeVisible();
  await item.click();
}

export async function selectNaiveOption(page: Page, container: string, value: string, index = 0) {
  const select = page.locator(container).locator('.n-base-selection').nth(index);
  await expect(select).toBeVisible();
  await select.click();
  const option = page.locator('.n-base-select-option').filter({ hasText: value }).first();
  await expect(option).toBeVisible();
  await option.click();
}

export async function openPort(page: Page, portName: string) {
  await openSerialModule(page);
  await openSidebarTab(page, '串口');
  await selectNaiveOption(page, '.port-config-form', portName);
  await page.locator('.port-config-form').getByRole('button', { name: '打开串口' }).click();
  await expect(page.locator('.serial-tab-content')).toBeVisible({ timeout: 10000 });
}

export async function openPorts(page: Page, portNames: string[]) {
  await openSerialModule(page);
  for (const portName of portNames) {
    await openSidebarTab(page, '串口');
    await selectNaiveOption(page, '.port-config-form', portName);
    await page.locator('.port-config-form').getByRole('button', { name: '打开串口' }).click();
    await expect(page.locator('.n-tabs-tab').filter({ hasText: portName })).toBeVisible({ timeout: 10000 });
  }
}

export async function createVirtualPort(page: Page, id: string, portName: string) {
  await openSerialModule(page);
  await openSidebarTab(page, '虚拟');
  await page.getByPlaceholder('例: vport-1').fill(id);
  await page.getByPlaceholder('例: ttyV0').fill(portName);
  await page.getByRole('button', { name: '创建' }).click();
  await expect(page.locator('.virtual-pair-panel')).toContainText(id, { timeout: 10000 });
  await expect(page.locator('.virtual-pair-panel')).toContainText(`/tmp/${portName}`);
}

export async function switchDataView(page: Page, viewLabel: string) {
  await selectNaiveOption(page, '.data-display__controls', viewLabel);
}

export async function waitForDisplayText(page: Page, text: string) {
  await expect(page.locator('.data-display')).toContainText(text, { timeout: 10000 });
}

export async function emitSerialData(page: Page, portId: string, text: string, encoding: 'bytes' | 'base64' = 'bytes') {
  await page.evaluate(({ portId, text, encoding }) => {
    const bytes = Array.from(new TextEncoder().encode(text));
    window.dispatchEvent(new CustomEvent('mocktrue:serial-data', {
      detail: {
        PortID: portId,
        Data: encoding === 'base64'
          ? btoa(String.fromCharCode(...bytes))
          : bytes,
      },
    }));
  }, { portId, text, encoding });
}
