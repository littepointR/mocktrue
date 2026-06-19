import type { ModuleFrontend } from '../core/module/types'

export const settingsModule: ModuleFrontend = {
  id: 'settings',
  activity: { icon: 'settings', title: '设置' },
  views: [
    { id: 'settings.global', title: '全局设置', component: 'settings/GlobalSettings' },
    { id: 'settings.serial', title: '串口', component: 'settings/SerialSettings' },
  ],
}

export * from './stores/settingsStore'
