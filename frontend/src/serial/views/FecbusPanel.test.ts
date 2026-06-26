import { flushPromises, mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import FecbusPanel from './FecbusPanel.vue'
import { useFecbusStore } from '../stores/fecbusStore'
import { serialService } from '../services/serialService'
import { FunctionCode, SessionRole, StatusCode } from '../../../bindings/github.com/littepointR/portweave/internal/modules/serial/fecbus/models.js'

vi.mock('../services/serialService', () => ({
  serialService: {
    listFecbusSessions: vi.fn(async () => []),
    openFecbusSession: vi.fn(async () => null),
    fecbusSendRequest: vi.fn(async () => ({
      ID: 'tx-1',
      SessionID: 'fec-1',
      RequestFrameHex: '7e 10',
      ResponseFrameHex: '7e 11',
      Response: null,
      Error: '',
    })),
    startFecbusSlave: vi.fn(async (request: any) => ({
      ID: request.SessionID,
      Name: 'FECbus Slave',
      Role: 'slave',
      Config: { PortName: '/tmp/ttyF0', BaudRate: 9600, DataBits: 8, StopBits: '1', Parity: 'none', FlowMode: 'none', ReadBufKB: 32 },
      Status: 'open',
      RxBytes: 0,
      TxBytes: 0,
      SlaveRunning: true,
      SourceAddress: 1,
      TargetAddress: 2,
      SlaveUnits: [],
      StartedAt: '',
      StoppedAt: '',
      LastError: '',
    })),
    stopFecbusSlave: vi.fn(async () => undefined),
    addFecbusSlaveUnit: vi.fn(async () => undefined),
    removeFecbusSlaveUnit: vi.fn(async () => undefined),
    queryFecbusFrames: vi.fn(async () => ({ Frames: [], Offset: 0, Limit: 200, Total: 0, EOF: true })),
    clearFecbusFrames: vi.fn(async () => undefined),
    listPorts: vi.fn(async () => []),
    enumeratePorts: vi.fn(async () => []),
  },
}))

describe('FecbusPanel', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    setActivePinia(createPinia())
  })

  it('opens and refreshes creation sessions through the header controls', async () => {
    const store = useFecbusStore()
    store.portForm.port = '/tmp/ttyF0'
    vi.mocked(serialService.openFecbusSession).mockResolvedValueOnce(sampleSession('fec-open') as any)

    const wrapper = mount(FecbusPanel, { global: { stubs } })

    await findButton(wrapper, '刷新').trigger('click')
    await flushPromises()
    expect(serialService.listFecbusSessions).toHaveBeenCalled()

    await findButton(wrapper, '打开').trigger('click')
    await flushPromises()
    expect(serialService.openFecbusSession).toHaveBeenCalledWith(expect.objectContaining({
      Config: expect.objectContaining({ PortName: '/tmp/ttyF0' }),
      Role: SessionRole.SessionRoleMaster,
    }))
    expect(wrapper.emitted('opened')?.[0]).toEqual(['fec-open'])
    expect(store.activeSessionId).toBe('fec-open')
  })

  it('only renders session creation controls in the operation panel', () => {
    const wrapper = mount(FecbusPanel, { global: { stubs } })
    const text = wrapper.text()

    expect(text).toContain('FECbus 调试')
    expect(text).toContain('角色')
    expect(text).toContain('打开')
    expect(text).not.toContain('发送帧')
    expect(text).not.toContain('帧历史')
  })

  it('renders a tab workbench with send, slave, history, and function catalog sections', () => {
    const store = useFecbusStore()
    store.restoreState({
      ...store.exportState(),
      activeSessionId: 'fec-1',
      sessions: [sampleSession('fec-1')],
      framePages: {
        'fec-1': {
          Frames: [{
            Seq: 1,
            SessionID: 'fec-1',
            Direction: 'tx',
            Frame: null as any,
            Hex: '7e 00',
            Error: '',
            Timestamp: '' as any,
            Annotated: sampleAnnotation(),
          }],
          Offset: 0,
          Limit: 200,
          Total: 1,
          EOF: true,
        },
      },
    })

    const wrapper = mount(FecbusPanel, {
      props: { variant: 'tab', sessionId: 'fec-1' },
      global: { stubs },
    })
    const text = wrapper.text()

    expect(text).toContain('FECbus Master')
    expect(text).toContain('发送帧')
    expect(text).toContain('设备从站')
    expect(text).toContain('帧历史')
    expect(text).toContain('功能码')
    expect(text).toContain('查协议版本')
    expect(text).toContain('7e 00')
    expect(wrapper.find('[data-testid="fecbus-send-form"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="fecbus-frame-history"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="fecbus-frame-history"] [data-testid="resize-handle-hex"]').exists()).toBe(true)
  })

  it('renders built-in and custom structured send fields in tab sessions', async () => {
    const store = useFecbusStore()
    store.sessions.set('fec-1', sampleSession('fec-1'))
    store.setActiveSession('fec-1')
    store.sendForm.inputMode = 'structured'
    store.sendForm.functionCode = FunctionCode.FunctionQueryDeviceStatus
    const wrapper = mount(FecbusPanel, {
      props: { variant: 'tab', sessionId: 'fec-1' },
      global: { stubs },
    })

    expect(wrapper.text()).toContain('控制器编号')
    expect(wrapper.text()).toContain('单元编号')
    expect(wrapper.text()).toContain('通道编号')

    store.sendForm.functionCode = FunctionCode.FunctionQueryProtocolVersion
    await wrapper.vm.$nextTick()
    expect(wrapper.text()).toContain('控制器编号')

    store.upsertCustomFunction({
      Code: 8 as FunctionCode,
      Name: '厂商扩展',
      Description: '',
      Direction: 'custom',
      Answer: true,
      Fields: [{ Key: 'vendor', Label: '厂商值', Offset: 1, Length: 1, Type: 'uint8', Endian: 'little', Enum: null, Meaning: '' }],
    })
    store.sendForm.functionCode = 8
    await wrapper.vm.$nextTick()

    expect(wrapper.text()).toContain('厂商值')
  })

  it('renders the empty tab message when the target session is missing', () => {
    const wrapper = mount(FecbusPanel, {
      props: { variant: 'tab', sessionId: 'missing-fecbus' },
      global: { stubs },
    })

    expect(wrapper.text()).toContain('选择或打开 FECbus 会话')
  })

  it('drives send, slave, frame history, and custom function actions', async () => {
    const store = useFecbusStore()
    store.sessions.set('fec-1', sampleSession('fec-1'))
    store.setActiveSession('fec-1')
    const wrapper = mount(FecbusPanel, {
      props: { variant: 'tab', sessionId: 'fec-1' },
      global: { stubs },
    })
    await flushPromises()

    await findButton(wrapper, '发送').trigger('click')
    await flushPromises()
    expect(serialService.fecbusSendRequest).toHaveBeenCalledWith(expect.objectContaining({
      SessionID: 'fec-1',
      Function: FunctionCode.FunctionQueryDeviceStatus,
    }))
    expect(wrapper.text()).toContain('7e 10')

    store.slaveForm = { address: 4, statusCode: StatusCode.StatusBusy, autoStatusAnswer: false, acceptBroadcast: false }
    await findButton(wrapper, '添加/更新').trigger('click')
    await flushPromises()
    expect(serialService.addFecbusSlaveUnit).toHaveBeenCalledWith('fec-1', expect.objectContaining({ Address: 4 }))

    store.sessions.set('fec-1', sampleSession('fec-1'))
    store.setActiveSession('fec-1')
    await findButton(wrapper, '启动').trigger('click')
    await flushPromises()
    expect(serialService.startFecbusSlave).toHaveBeenCalledWith(expect.objectContaining({ SessionID: 'fec-1' }))

    await findButton(wrapper, '停止').trigger('click')
    await flushPromises()
    expect(serialService.stopFecbusSlave).toHaveBeenCalledWith('fec-1')

    await wrapper.find('.fecbus-panel__unit-list button button').trigger('click')
    await flushPromises()
    expect(serialService.removeFecbusSlaveUnit).toHaveBeenCalledWith('fec-1', 2)

    vi.mocked(serialService.queryFecbusFrames).mockResolvedValueOnce({
      Frames: [sampleFrame(3)],
      Offset: 0,
      Limit: 200,
      Total: 1,
      EOF: true,
    } as any)
    store.setFrameFilter('fec-1', { direction: 'rx', search: '7e' })
    await wrapper.vm.$nextTick()
    await findButton(wrapper, '查询').trigger('click')
    await flushPromises()
    expect(store.frameFilters['fec-1']).toEqual({ direction: 'rx', search: '7e' })
    expect(serialService.queryFecbusFrames).toHaveBeenLastCalledWith(expect.objectContaining({
      Direction: 'rx',
      Search: '7e',
    }))

    await wrapper.find('.fecbus-panel__frame-row').trigger('click')
    expect(wrapper.text()).toContain('控制器编号')
    await findButton(wrapper, '清空').trigger('click')
    await flushPromises()
    expect(serialService.clearFecbusFrames).toHaveBeenCalledWith('fec-1')

    store.sendForm.functionCode = 8
    await findButton(wrapper, '添加当前功能码定义').trigger('click')
    await wrapper.vm.$nextTick()
    expect(store.customFunctions[0]).toMatchObject({ Code: 8, Name: '用户自定义' })
    await findButton(wrapper, '添加字段').trigger('click')
    expect(store.customFunctions[0]?.Fields?.[0]).toMatchObject({ Key: 'value', Type: 'uint8' })
    await wrapper.findAll('.fecbus-panel__custom-function button').at(1)?.trigger('click')
    expect(store.customFunctions).toHaveLength(0)
  })
})

const stubs = {
  NButton: { props: ['disabled'], emits: ['click'], template: '<button v-bind="$attrs" :disabled="disabled" @click="$emit(\'click\', $event)"><slot /></button>' },
  NInput: { props: ['value'], emits: ['update:value'], template: '<input v-bind="$attrs" :value="value" @input="$emit(\'update:value\', $event.target.value)" />' },
  NInputNumber: { props: ['value'], emits: ['update:value'], template: '<input v-bind="$attrs" type="number" :value="value" @input="$emit(\'update:value\', Number($event.target.value))" />' },
  NSelect: { props: ['value', 'options'], emits: ['update:value'], template: '<select v-bind="$attrs" :value="value" @change="$emit(\'update:value\', $event.target.value)"><option v-for="option in options" :key="option.value" :value="option.value">{{ option.label }}</option></select>' },
  NSwitch: { props: ['value'], emits: ['update:value'], template: '<input v-bind="$attrs" type="checkbox" :checked="value" @change="$emit(\'update:value\', $event.target.checked)" />' },
  NForm: { template: '<form><slot /></form>' },
  NFormItem: { template: '<label><slot /></label>' },
  NSpace: { template: '<div><slot /></div>' },
  NAlert: { template: '<div role="alert"><slot /></div>' },
  NTag: { template: '<span><slot /></span>' },
  NTable: { template: '<table><slot /></table>' },
}

function sampleSession(id: string) {
  return {
    ID: id,
    Name: id,
    Role: SessionRole.SessionRoleMaster,
    Config: {
      PortName: '/tmp/ttyF0',
      BaudRate: 9600,
      DataBits: 8,
      StopBits: '1',
      Parity: 'none',
      FlowMode: 'none',
      ReadBufKB: 32,
    },
    Status: 'open',
    RxBytes: 0,
    TxBytes: 0,
    SlaveRunning: false,
    SourceAddress: 1,
    TargetAddress: 2,
    SlaveUnits: [],
    StartedAt: '' as any,
    StoppedAt: '' as any,
    LastError: '',
  }
}

function sampleFrame(seq: number) {
  return {
    Seq: seq,
    SessionID: 'fec-1',
    Direction: 'rx',
    Frame: { Type: 1, TargetAddress: 2, SourceAddress: 1, MessageNumber: 3, Data: '2c' } as any,
    Hex: '7e 03',
    Error: '',
    Timestamp: '' as any,
    Annotated: {
      ...sampleAnnotation(),
      GroupColorIndex: 3,
      DataFields: [{ Key: 'controller_id', Label: '控制器编号', Start: 1, End: 1, Hex: '01', Value: 1, ValueText: '1', Meaning: '主机' }],
    },
  }
}

function findButton(wrapper: ReturnType<typeof mount>, label: string) {
  const button = wrapper.findAll('button').find(item => item.text() === label)
  if (!button) throw new Error(`Button not found: ${label}`)
  return button
}

function sampleAnnotation() {
  return {
    Segments: [{ Key: 'frame_head', Label: '帧头', Start: 0, End: 1, Hex: '7e', Value: 0x7e, ValueText: '126', Meaning: '0x7E' }],
    DataFields: [],
    Function: {
      Code: FunctionCode.FunctionQueryProtocolVersion,
      Hex: '2CH',
      Name: '查协议版本',
      Description: '',
      Direction: 'controller_to_device',
      Answer: true,
      Custom: false,
      Reserved: false,
    },
    GroupKey: '',
    GroupColorIndex: -1,
    Summary: '查协议版本',
    Warnings: [],
  }
}
