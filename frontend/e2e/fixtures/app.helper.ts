import { expect, type Page } from '@playwright/test';

export async function openSerialModule(page: Page) {
  await page.goto('/');
  await expect(page.locator('.activity-bar')).toBeVisible({ timeout: 10000 });
  await page.locator('.activity-bar__item').first().click();
  await expect(page.locator('.serial-view')).toBeVisible();
}

export async function openSerialGraph(page: Page) {
  await openSerialModule(page);
  await expect(page.locator('[data-testid="sidebar-view-serial.graph"]')).toBeVisible();
  await expect(page.locator('[data-testid="serial-graph-panel"]')).toBeVisible({ timeout: 10000 });
}

export async function addGraphNode(page: Page, type: string, nodeId: string) {
  await page.locator(`[data-testid="serial-graph-provider-${type}"]`).click();
  await expect(page.locator(`[data-testid="serial-graph-node-${nodeId}"]`)).toBeVisible({ timeout: 10000 });
  await expect(page.locator(`[data-testid="serial-graph-node-tab-${nodeId}"]`)).toBeVisible({ timeout: 10000 });
}

export async function connectGraphNodes(
  page: Page,
  sourceNode: string,
  sourcePort: string,
  targetNode: string,
  targetPort: string,
) {
  await page.locator(`[data-testid="serial-graph-output-${sourceNode}-${sourcePort}"]`).click();
  await page.locator(`[data-testid="serial-graph-input-${targetNode}-${targetPort}"]`).click();
  await expect(page.locator('[data-testid^="serial-graph-edge-line-"]')).toHaveCount(1, { timeout: 10000 });
}

export async function createSenderReceiverGraph(page: Page) {
  await openSerialGraph(page);
  await addGraphNode(page, 'serial.script.generator', 'node-1');
  await addGraphNode(page, 'serial.virtual', 'node-2');
  await connectGraphNodes(page, 'node-1', 'out', 'node-2', 'tx');
}

export async function startGraphRuntime(page: Page) {
  await page.locator('[data-testid="serial-graph-start"]').click();
  await expect(page.locator('[data-testid="serial-graph-runtime-status"]')).toContainText('running', { timeout: 10000 });
}

export async function activateNodeTab(page: Page, nodeId: string) {
  const tab = page.locator(`[data-testid="serial-graph-node-tab-${nodeId}"]`);
  await tab.click();
  await expect(tab).toHaveClass(/serial-graph__node-tab--active/);
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
    window.dispatchEvent(new CustomEvent('portweave:serial-data', {
      detail: {
        PortID: portId,
        Data: encoding === 'base64'
          ? btoa(String.fromCharCode(...bytes))
          : bytes,
      },
    }));
  }, { portId, text, encoding });
}
