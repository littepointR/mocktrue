import { test, expect } from '@playwright/test';
import {
  activateNodeTab,
  addGraphNode,
  connectGraphNodes,
  openSerialGraph,
  startGraphRuntime,
} from './fixtures/app.helper';
import { injectWailsMock } from './fixtures/wails-mock.helper';

/**
 * 真实 UI E2E 测试
 *
 * 这套测试运行前端构建产物并注入共享 Wails RPC mock，覆盖当前串口拓扑 UI。
 */
test.describe('真实 UI 测试', () => {
  test.beforeEach(async ({ page }) => {
    await injectWailsMock(page);
  });

  test('应用启动后显示 VS Code 风格布局', async ({ page }) => {
    await page.goto('/');

    await expect(page.locator('.app-shell')).toBeVisible();
    await expect(page.locator('.activity-bar')).toBeVisible();
    await expect(page.locator('.sidebar')).toBeVisible();
    await expect(page.locator('.editor-groups')).toBeVisible();
    await expect(page.locator('.status-bar')).toBeVisible();
  });

  test('点击串口图标后切换到串口拓扑模块', async ({ page }) => {
    await openSerialGraph(page);

    await expect(page.locator('.sidebar__title')).toContainText('串口调试');
    await expect(page.locator('[data-testid="sidebar-view-serial.graph"]')).toContainText('串口拓扑');
    await expect(page.locator('[data-testid="serial-graph-panel"]')).toBeVisible();
    await expect(page.locator('[data-testid="serial-graph-node-palette"]')).toBeVisible();
    await expect(page.locator('[data-testid="serial-graph-canvas"]')).toBeVisible();
  });

  test('串口拓扑节点显示完整配置表单', async ({ page }) => {
    await openSerialGraph(page);
    await addGraphNode(page, 'serial.physical', 'node-1');

    await expect(page.locator('[data-testid="serial-graph-node-content"]')).toBeVisible();
    await expect(page.locator('[data-testid="serial-graph-config-portName"]')).toBeVisible();
    await expect(page.locator('[data-testid="serial-graph-config-baudRate"]')).toBeVisible();
    await expect(page.locator('[data-testid="serial-graph-config-dataBits"]')).toBeVisible();
    await expect(page.locator('[data-testid="serial-graph-config-stopBits"]')).toBeVisible();
    await expect(page.locator('[data-testid="serial-graph-config-parity"]')).toBeVisible();
    await expect(page.locator('[data-testid="serial-graph-config-flowMode"]')).toBeVisible();
  });

  test('脚本生成到虚拟串口的拓扑流可启动并刷新接收缓冲区', async ({ page }) => {
    await openSerialGraph(page);
    await page.evaluate(() => {
      (window as any).__mockState.graphBufferText = 'real-ui-rx';
    });
    await addGraphNode(page, 'serial.script.generator', 'node-1');
    await addGraphNode(page, 'serial.virtual', 'node-2');
    await connectGraphNodes(page, 'node-1', 'out', 'node-2', 'tx');
    await activateNodeTab(page, 'node-2');

    await startGraphRuntime(page);
    await expect(page.locator('[data-testid="serial-graph-content-refresh-buffer"]')).toBeEnabled();
    await page.locator('[data-testid="serial-graph-content-refresh-buffer"]').click();

    await expect(page.locator('[data-testid="serial-graph-content-node-buffer"]')).toContainText('real-ui-rx');
  });

  test('窗口标题栏不再依赖旧 macOS 顶部 padding shim', async ({ page }) => {
    await page.goto('/');

    await expect(page.locator('.app-shell__titlebar')).toBeVisible();
    const paddingTop = await page.locator('.app-shell').evaluate(el => getComputedStyle(el).paddingTop);
    expect(paddingTop).toBe('0px');
  });
});
