import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import FecbusPanel from './FecbusPanel.vue'
import { useFecbusStore } from '../stores/fecbusStore'
import { FunctionCode, SessionRole, StatusCode } from '../../../bindings/github.com/littepointR/mocktrue/internal/modules/serial/fecbus/models.js'

vi.mock('../services/serialService', () => ({
  serialService: {
    listFecbusSessions: vi.fn(async () => []),
    openFecbusSession: vi.fn(async () => null),
    listPorts: vi.fn(async () => []),
    enumeratePorts: vi.fn(async () => []),
  },
}))

describe('FecbusPanel', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
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
})

const stubs = {
  NButton: { template: '<button><slot /></button>' },
  NInput: { template: '<input />' },
  NInputNumber: { template: '<input />' },
  NSelect: { template: '<select />' },
  NSwitch: { template: '<input type="checkbox" />' },
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
