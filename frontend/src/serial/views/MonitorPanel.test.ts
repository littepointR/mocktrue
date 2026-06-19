import { flushPromises, mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import MonitorPanel from './MonitorPanel.vue'
import { useMonitorStore } from '../stores/monitorStore'

vi.mock('naive-ui', () => ({
  NAlert: { props: ['type', 'closable'], template: '<div><slot /></div>' },
  NButton: {
    props: ['disabled', 'loading'],
    emits: ['click'],
    template: '<button type="button" :disabled="disabled" @click="$emit(\'click\', $event)"><slot /></button>',
  },
  NForm: { template: '<form><slot /></form>' },
  NFormItem: { props: ['label'], template: '<label><span>{{ label }}</span><slot /></label>' },
  NInput: {
    props: ['value', 'placeholder', 'disabled'],
    emits: ['update:value'],
    template: '<input :value="value" :placeholder="placeholder" :disabled="disabled" @input="$emit(\'update:value\', $event.target.value)" />',
  },
  NSelect: {
    props: ['value', 'options', 'disabled'],
    emits: ['update:value'],
    template: `
      <select :value="value" :disabled="disabled" @change="$emit('update:value', $event.target.value)">
        <option value=""></option>
        <option v-for="option in options" :key="option.value" :value="option.value">{{ option.label }}</option>
      </select>
    `,
  },
  NSpace: { template: '<div><slot /></div>' },
}))

vi.mock('../services/serialService', () => ({
  serialService: {
    enumeratePorts: vi.fn(async () => [
      { Name: '/dev/tty.usbserial', FriendlyName: 'USB Serial' },
    ]),
  },
}))

vi.mock('../../../bindings/github.com/suyue/mocktrue/internal/modules/serial/service.js', () => ({
  StartMonitor: vi.fn(),
  StartAutoVirtualMonitor: vi.fn(),
  ListMonitors: vi.fn(async () => []),
  QueryMonitorFrames: vi.fn(async () => ({ Frames: [], Total: 0, NextOffset: 0 })),
  StopMonitor: vi.fn(async () => undefined),
  DeleteMonitor: vi.fn(async () => undefined),
  ClearMonitorFrames: vi.fn(async () => undefined),
}))

describe('MonitorPanel auto virtual monitor', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('starts monitoring from one selected real port', async () => {
    const monitor = useMonitorStore()
    const start = vi.spyOn(monitor, 'startAutoVirtualMonitor').mockResolvedValue('mon-1')
    const wrapper = mount(MonitorPanel)
    await flushPromises()

    expect(wrapper.text()).toContain('被监听端口')
    expect(wrapper.text()).not.toContain('端口 B')

    await wrapper.findAll('select')[0].setValue('/dev/tty.usbserial')
    await wrapper.findAll('button').find(button => button.text() === '开始监控')?.trigger('click')

    expect(start).toHaveBeenCalledWith(expect.objectContaining({
      sourcePort: '/dev/tty.usbserial',
      baudRate: 115200,
    }))
    expect(wrapper.emitted('started')?.[0]).toEqual(['mon-1'])
  })
})
