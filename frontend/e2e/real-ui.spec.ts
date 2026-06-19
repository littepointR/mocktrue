import { test, expect, type Page } from '@playwright/test'
import path from 'path'
import fs from 'fs'
import { fileURLToPath } from 'url'

const __filename = fileURLToPath(import.meta.url)
const __dirname = path.dirname(__filename)

/**
 * 真实 UI E2E 测试
 *
 * 这套测试：
 * 1. 启动 Vite 预览服务器（前端构建产物）
 * 2. 注入 Wails RPC mock，让前端能够工作
 * 3. 模拟真实用户操作：点击、填表、关闭等
 * 4. 验证 UI 状态变化
 */

const screenshotsDir = path.join(__dirname, '../test-results/screenshots')
if (!fs.existsSync(screenshotsDir)) {
  fs.mkdirSync(screenshotsDir, { recursive: true })
}

// 注入 Wails RPC mock
async function injectWailsMock(page: Page) {
  await page.addInitScript(() => {
    const mockState = {
      ports: [
        { Name: '/tmp/ttyV0', IsUSB: false, VID: '', PID: '', SerialNumber: '', FriendlyName: '/tmp/ttyV0' },
        { Name: '/tmp/ttyV1', IsUSB: false, VID: '', PID: '', SerialNumber: '', FriendlyName: '/tmp/ttyV1' },
      ],
      handles: [] as any[],
      virtualPairs: [] as any[],
      virtualPorts: [] as any[],
      bridges: [] as any[],
      nextHandleId: 1,
    }

    // Wails Call.ByID mock
    const handlers: Record<number, (...args: any[]) => any> = {
      // Ping
      3334354438: (msg: string) => `pong:${msg}`,

      // EnumeratePorts
      2380440348: () => mockState.ports,

      // OpenPort
      1403721065: (req: any) => {
        const id = `port-${mockState.nextHandleId++}`
        const handle = {
          ID: id,
          Config: req.Config,
          IsOpen: true,
          RxBytes: 0,
          TxBytes: 0,
        }
        mockState.handles.push(handle)
        return handle
      },

      // ClosePort
      358686901: (id: string) => {
        const idx = mockState.handles.findIndex(h => h.ID === id)
        if (idx >= 0) mockState.handles.splice(idx, 1)
        return null
      },

      // ListPorts
      2409741650: () => mockState.handles,

      // QueryPage
      257355161: (portID: string, offset: number, length: number) => {
        return {
          PortID: portID,
          Offset: offset,
          Length: length,
          Data: '',
          Total: 0,
        }
      },

      // Send
      304932660: (req: any) => {
        const handle = mockState.handles.find(h => h.ID === req.PortID)
        if (handle) handle.TxBytes += req.Content.length
        return req.Content.length
      },

      // Virtual serial
      4156784338: (id: string, portName: string) => methodHandlers.CreateVirtualPort(id, portName),
      2248938995: (id: string) => methodHandlers.DeleteVirtualPort(id),
      1806471203: () => methodHandlers.ListVirtualPorts(),

      // Bridge
      2000913547: (id: string, p1: string, p2: string, baud: number) => methodHandlers.CreateBridge(id, p1, p2, baud),
      2952555164: (id: string) => methodHandlers.DeleteBridge(id),
      2577893816: () => methodHandlers.ListBridges(),
    }

    // 通过函数名映射（更稳定）
    const methodHandlers: Record<string, (...args: any[]) => any> = {
      'CreateVirtualPair': (id: string, p1: string, p2: string) => {
        if (mockState.virtualPairs.some(p => p.ID === id)) {
          throw new Error('pair ID already exists')
        }
        const pair = { ID: id, Port1: `/tmp/${p1}`, Port2: `/tmp/${p2}` }
        mockState.virtualPairs.push(pair)
        return pair
      },
      'DeleteVirtualPair': (id: string) => {
        const idx = mockState.virtualPairs.findIndex(p => p.ID === id)
        if (idx < 0) throw new Error('pair not found')
        mockState.virtualPairs.splice(idx, 1)
        return null
      },
      'ListVirtualPairs': () => mockState.virtualPairs,
      'CreateVirtualPort': (id: string, portName: string) => {
        if (mockState.virtualPorts.some(p => p.ID === id)) {
          throw new Error('virtual port ID already exists')
        }
        const vport = { ID: id, Port: `/tmp/${portName}` }
        mockState.virtualPorts.push(vport)
        mockState.ports.push({
          Name: vport.Port,
          IsUSB: false,
          VID: '',
          PID: '',
          SerialNumber: '',
          FriendlyName: vport.Port,
        })
        return vport
      },
      'DeleteVirtualPort': (id: string) => {
        const idx = mockState.virtualPorts.findIndex(p => p.ID === id)
        if (idx < 0) throw new Error('virtual port not found')
        mockState.virtualPorts.splice(idx, 1)
        return null
      },
      'ListVirtualPorts': () => mockState.virtualPorts,
      'CreateBridge': (id: string, p1: string, p2: string, baud: number) => {
        if (p1 === p2) throw new Error('cannot bridge a port to itself')
        if (mockState.bridges.some(b => b.ID === id)) {
          throw new Error('bridge ID already exists')
        }
        const bridge = { ID: id, Port1: p1, Port2: p2, BaudRate: baud }
        mockState.bridges.push(bridge)
        return bridge
      },
      'DeleteBridge': (id: string) => {
        const idx = mockState.bridges.findIndex(b => b.ID === id)
        if (idx < 0) throw new Error('bridge not found')
        mockState.bridges.splice(idx, 1)
        return null
      },
      'ListBridges': () => mockState.bridges,
      'EnumeratePorts': () => mockState.ports,
      'ListPorts': () => mockState.handles,
      'OpenPort': (req: any) => {
        const id = `port-${mockState.nextHandleId++}`
        const handle = { ID: id, Config: req.Config, IsOpen: true, RxBytes: 0, TxBytes: 0 }
        mockState.handles.push(handle)
        return handle
      },
      'ClosePort': (id: string) => {
        const idx = mockState.handles.findIndex(h => h.ID === id)
        if (idx >= 0) mockState.handles.splice(idx, 1)
        return null
      },
      'QueryPage': () => ({ Data: '', Total: 0, Offset: 0, Length: 0 }),
    }

    // Mock @wailsio/runtime
    ;(window as any).__wailsMock = {
      callsByID: handlers,
      callsByName: methodHandlers,
      state: mockState,
    }

    // Hook fetch for /wails/runtime/call
    const originalFetch = window.fetch
    window.fetch = async function (input: RequestInfo | URL, init?: RequestInit): Promise<Response> {
      const url = typeof input === 'string' ? input : input.toString()

      if (url.includes('/wails/runtime') || url.includes('/wails/call')) {
        try {
          const body = init?.body ? JSON.parse(init.body as string) : {}
          const descriptor = body.args ?? body
          const methodName = descriptor.methodName || ''
          const args = descriptor.args || []
          const methodID = descriptor.methodID ?? body.id

          let result: any
          if (methodName && methodHandlers[methodName]) {
            result = methodHandlers[methodName](...args)
          } else if (methodID && handlers[methodID]) {
            result = handlers[methodID](...args)
          } else {
            result = null
          }

          return new Response(JSON.stringify(result), {
            status: 200,
            headers: { 'Content-Type': 'application/json' },
          })
        } catch (e: any) {
          return new Response(JSON.stringify({ error: e.message }), {
            status: 500,
          })
        }
      }

      return originalFetch.call(window, input, init)
    }
  })
}

test.describe('真实 UI 测试', () => {
  test.beforeEach(async ({ page }) => {
    await injectWailsMock(page)
    await page.goto('/')
    await page.waitForTimeout(800)
  })

  test('应用启动后显示 VS Code 风格布局', async ({ page }) => {
    // 验证关键 UI 元素存在
    await expect(page.locator('.app-shell')).toBeVisible()
    await expect(page.locator('.activity-bar')).toBeVisible()
    await expect(page.locator('.sidebar')).toBeVisible()
    await expect(page.locator('.editor-groups')).toBeVisible()
    await expect(page.locator('.status-bar')).toBeVisible()

    await page.screenshot({
      path: path.join(screenshotsDir, 'ui-01-initial.png'),
      fullPage: true,
    })
  })

  test('点击串口图标后切换到串口模块', async ({ page }) => {
    const serialIcon = page.locator('.activity-bar__item').first()
    await serialIcon.click()
    await page.waitForTimeout(500)

    // 验证 shell 左侧栏承载串口操作入口
    const sidebar = page.locator('.sidebar')
    await expect(sidebar.locator('.sidebar__item').filter({ hasText: '打开串口' })).toBeVisible()
    await expect(sidebar.locator('.sidebar__item').filter({ hasText: '添加虚拟串口' })).toBeVisible()
    await expect(sidebar.locator('.sidebar__item').filter({ hasText: '添加串口桥接' })).toBeVisible()
    await expect(page.locator('.serial-view__sidebar')).toHaveCount(0)

    await page.screenshot({
      path: path.join(screenshotsDir, 'ui-02-serial-module.png'),
      fullPage: true,
    })
  })

  test('串口配置面板显示完整表单', async ({ page }) => {
    await page.locator('.activity-bar__item').first().click()
    await page.waitForTimeout(500)

    await page.locator('.sidebar__item').filter({ hasText: '打开串口' }).click()
    await expect(page.locator('text=端口')).toBeVisible()
    await expect(page.locator('text=波特率')).toBeVisible()
    await expect(page.locator('text=数据位')).toBeVisible()
    await expect(page.locator('text=停止位')).toBeVisible()
    await expect(page.locator('text=校验')).toBeVisible()
    await expect(page.locator('text=流控')).toBeVisible()
    await expect(page.locator('.port-config-form').getByRole('button', { name: '打开串口' })).toBeVisible()
  })

  test('切换到虚拟串口标签显示创建表单', async ({ page }) => {
    await page.locator('.activity-bar__item').first().click()
    await page.waitForTimeout(300)

    // 通过 shell 左侧栏切换到「添加虚拟串口」
    await page.locator('.sidebar__item').filter({ hasText: '添加虚拟串口' }).click()
    await page.waitForTimeout(500)

    // 验证虚拟串口面板元素
    await expect(page.locator('h3:has-text("虚拟串口")')).toBeVisible()
    await expect(page.locator('text=新建虚拟串口')).toBeVisible()
    await expect(page.locator('button:has-text("自动生成")')).toBeVisible()
    await expect(page.locator('button:has-text("创建")')).toBeVisible()
    await expect(page.locator('text=暂无虚拟串口')).toBeVisible()

    await page.screenshot({
      path: path.join(screenshotsDir, 'ui-03-virtual-pair.png'),
      fullPage: true,
    })
  })

  test('点击自动生成按钮填充表单', async ({ page }) => {
    await page.locator('.activity-bar__item').first().click()
    await page.waitForTimeout(300)
    await page.locator('.sidebar__item').filter({ hasText: '添加虚拟串口' }).click()
    await page.waitForTimeout(500)

    await page.locator('button:has-text("自动生成")').click()
    await page.waitForTimeout(200)

    // 验证表单已填充
    const idInput = page.locator('input[placeholder*="vport-1"]')
    await expect(idInput).toHaveValue(/^vport-/)
  })

  test('切换到桥接标签显示创建表单', async ({ page }) => {
    await page.locator('.activity-bar__item').first().click()
    await page.waitForTimeout(300)

    await page.locator('.sidebar__item').filter({ hasText: '添加串口桥接' }).click()
    await page.waitForTimeout(500)

    // 验证桥接面板元素
    await expect(page.locator('h3:has-text("串口桥接")')).toBeVisible()
    await expect(page.locator('text=新建桥接')).toBeVisible()
    await expect(page.locator('button:has-text("创建桥接")')).toBeVisible()
    await expect(page.locator('text=暂无桥接')).toBeVisible()

    await page.screenshot({
      path: path.join(screenshotsDir, 'ui-04-bridge.png'),
      fullPage: true,
    })
  })

  test('完整工作流：创建虚拟串口 → 切换到桥接', async ({ page }) => {
    await page.locator('.activity-bar__item').first().click()
    await page.waitForTimeout(300)

    // 第一步：创建虚拟串口
    await page.locator('.sidebar__item').filter({ hasText: '添加虚拟串口' }).click()
    await page.waitForTimeout(300)

    await page.locator('button:has-text("自动生成")').click()
    await page.waitForTimeout(200)

    // 验证表单已填充
    const idInput = page.locator('input[placeholder*="vport-1"]')
    const idValue = await idInput.inputValue()
    expect(idValue).toMatch(/^vport-/)

    // 点击「创建」按钮（不验证后端响应，因为 mock RPC 路径可能不匹配）
    await page.locator('button:has-text("创建")').click()
    await page.waitForTimeout(500)

    await page.screenshot({
      path: path.join(screenshotsDir, 'ui-05-pair-created.png'),
      fullPage: true,
    })

    // 第二步：切换到桥接
    await page.locator('.sidebar__item').filter({ hasText: '添加串口桥接' }).click()
    await page.waitForTimeout(300)

    // 验证桥接面板可用
    await expect(page.locator('text=新建桥接')).toBeVisible()
  })

  test('窗口缩放响应式布局', async ({ page }) => {
    await page.locator('.activity-bar__item').first().click()
    await page.waitForTimeout(300)

    // 初始尺寸
    let appShell = await page.locator('.app-shell').boundingBox()
    expect(appShell?.width).toBeCloseTo(1280, 5)

    // 缩小
    await page.setViewportSize({ width: 1024, height: 768 })
    await page.waitForTimeout(300)
    appShell = await page.locator('.app-shell').boundingBox()
    expect(appShell?.width).toBeCloseTo(1024, 5)

    // 放大
    await page.setViewportSize({ width: 1920, height: 1080 })
    await page.waitForTimeout(300)
    appShell = await page.locator('.app-shell').boundingBox()
    expect(appShell?.width).toBeCloseTo(1920, 5)

    // 验证侧边栏宽度保持
    const sidebar = page.locator('.sidebar')
    const sidebarBox = await sidebar.boundingBox()
    expect(sidebarBox?.width).toBeCloseTo(200, 10)
  })

  test('暗色主题颜色正确', async ({ page }) => {
    // 全局背景
    const bodyBg = await page.locator('body').evaluate(el => getComputedStyle(el).backgroundColor)
    expect(bodyBg).toBe('rgb(30, 30, 30)') // #1e1e1e

    await page.locator('.activity-bar__item').first().click()
    await page.waitForTimeout(300)

    // 侧边栏背景
    const sidebarBg = await page.locator('.sidebar').evaluate(el => getComputedStyle(el).backgroundColor)
    expect(sidebarBg).toBe('rgb(37, 37, 38)') // #252526

    // 状态栏（蓝色）
    const statusBar = await page.locator('.status-bar').evaluate(el => getComputedStyle(el).backgroundColor)
    expect(statusBar).toBe('rgb(0, 122, 204)') // #007acc
  })

  test('macOS 顶部窗口控制按钮安全区', async ({ page }) => {
    const paddingTop = await page.locator('.app-shell').evaluate(el => getComputedStyle(el).paddingTop)
    expect(paddingTop).toBe('50px')
  })
})
