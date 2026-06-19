import { describe, expect, it } from 'vitest'
import { settingsModule } from './index'

describe('settingsModule', () => {
  it('contributes global and serial settings views', () => {
    expect(settingsModule.id).toBe('settings')
    expect(settingsModule.activity).toEqual({ icon: 'settings', title: '设置' })
    expect(settingsModule.views.map(view => [view.id, view.title])).toEqual([
      ['settings.global', '全局设置'],
      ['settings.serial', '串口'],
    ])
  })
})
