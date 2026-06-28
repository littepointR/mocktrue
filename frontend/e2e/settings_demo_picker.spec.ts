import { test, expect } from '@playwright/test'

test.describe('Settings demo picker', () => {
  test('searches, selects, and loads the remote serial demo from settings', async ({ page }) => {
    await page.goto('/')
    await expect(page.locator('.app-shell')).toBeVisible({ timeout: 10000 })

    await page.getByTestId('activity-settings').click()
    await expect(page.getByRole('heading', { name: '全局设置' })).toBeVisible()

    const demoSelect = page.locator('.settings-panel__demo-select .n-base-selection')
    await expect(demoSelect).toBeVisible()
    await demoSelect.click()
    await page.keyboard.type('远端')

    const remoteOption = page.locator('.n-base-select-option').filter({ hasText: '远端串口演示' }).first()
    await expect(remoteOption).toBeVisible()
    await remoteOption.click()
    await expect(page.locator('.settings-panel__demo-select')).toContainText('远端串口演示')

    await page.getByTestId('load-demo').click()

    await expect(page.getByTestId('activity-serial')).toHaveClass(/is-active/)
    await expect(page.getByTestId('serial-graph-name')).toHaveValue('远端串口演示')

    const canvas = page.getByTestId('serial-graph-canvas')
    await expect(canvas).toContainText('发送器')
    await expect(canvas).toContainText('远端串口')
    await expect(canvas).toContainText('接收器')
    await expect(canvas).toContainText('串口监控')

    const selectedNodeContent = page.getByTestId('serial-graph-node-content')
    await expect(selectedNodeContent).toContainText('远端串口')
    await expect(page.getByTestId('serial-graph-config-protocol')).toHaveValue('raw-tcp')
    await expect(page.getByTestId('serial-graph-config-host')).toHaveValue('127.0.0.1')
    await expect(page.getByTestId('serial-graph-config-port')).toHaveValue('3001')
  })
})
