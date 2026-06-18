import type { ModuleFrontend } from '../core/module/types'

/** The serial debugging module's frontend contribution. */
export const serialModule: ModuleFrontend = {
  id: 'serial',
  activity: { icon: 'serial', title: '串口调试' },
  views: [{ id: 'serial.connect', title: '连接', component: 'serial/ConnectView' }],
  async onActivate() {
    // Initialize Store event listeners
    const { useSerialStore } = await import('./stores/serialStore')
    const { useBufferStore } = await import('./stores/bufferStore')
    useSerialStore().initEventListeners()
    useBufferStore().initEventListeners()
  },
  async onDeactivate() {
    // Cleanup
    const { useSerialStore } = await import('./stores/serialStore')
    const { useBufferStore } = await import('./stores/bufferStore')
    useSerialStore().cleanup()
    useBufferStore().cleanup()
  },
}

export * from './stores/serialStore'
export * from './stores/bufferStore'
export * from './services'
