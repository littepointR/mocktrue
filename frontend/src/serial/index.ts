import type { ModuleFrontend } from '../core/module/types'

/** The serial debugging module's frontend contribution. */
export const serialModule: ModuleFrontend = {
  id: 'serial',
  activity: { icon: 'serial', title: '串口调试' },
  views: [
    { id: 'serial.open', title: '打开串口', component: 'serial/OpenPort' },
    { id: 'serial.virtual', title: '添加虚拟串口', component: 'serial/VirtualPort' },
    { id: 'serial.bridge', title: '添加串口桥接', component: 'serial/Bridge' },
    { id: 'serial.monitor', title: '串口监控', component: 'serial/Monitor' },
    { id: 'serial.modbus', title: 'Modbus 调试', component: 'serial/Modbus' },
  ],
  async onActivate() {
    // Initialize Store event listeners
    const { useSerialStore } = await import('./stores/serialStore')
    const { useBufferStore } = await import('./stores/bufferStore')
    const { useMonitorStore } = await import('./stores/monitorStore')
    const { useModbusStore } = await import('./stores/modbusStore')
    useSerialStore().initEventListeners()
    useBufferStore().initEventListeners()
    useMonitorStore().refreshSessions()
    useModbusStore().refreshSessions()
  },
  async onDeactivate() {
    // Cleanup
    const { useSerialStore } = await import('./stores/serialStore')
    const { useBufferStore } = await import('./stores/bufferStore')
    const { useMonitorStore } = await import('./stores/monitorStore')
    const { useModbusStore } = await import('./stores/modbusStore')
    useSerialStore().cleanup()
    useBufferStore().cleanup()
    useMonitorStore().cleanup()
    useModbusStore().cleanup()
  },
}

export * from './stores/serialStore'
export * from './stores/bufferStore'
export * from './stores/monitorStore'
export * from './stores/modbusStore'
export * from './services'
