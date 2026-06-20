import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import {
  defaultFecbusWorkspaceState,
  fecbusFunctionOptions,
  formatHexBytes,
  parseHexBytes,
  useFecbusStore,
} from './fecbusStore'
import {
  FrameType,
  FunctionCode,
  SessionRole,
  StatusCode,
} from '../../../bindings/github.com/suyue/mocktrue/internal/modules/serial/fecbus/models.js'

const serialServiceMock = vi.hoisted(() => ({
  listFecbusSessions: vi.fn(async () => []),
  openFecbusSession: vi.fn(async (request: any) => sampleSession(request.ID ?? 'fecbus-1', request.Role ?? 'master')),
  closeFecbusSession: vi.fn(async () => undefined),
  fecbusSendRequest: vi.fn(async (request: any) => ({
    ID: 'tx-1',
    SessionID: request.SessionID,
    RequestFrameHex: '7e 00',
    ResponseFrameHex: '7e 01',
    Response: null,
  })),
  startFecbusSlave: vi.fn(async (request: any) => sampleSession(request.SessionID ?? 'fecbus-1', 'slave')),
  stopFecbusSlave: vi.fn(async () => undefined),
  updateFecbusSlaveState: vi.fn(async () => undefined),
  addFecbusSlaveUnit: vi.fn(async () => undefined),
  removeFecbusSlaveUnit: vi.fn(async () => undefined),
  listFecbusSlaveUnits: vi.fn(async () => []),
  queryFecbusFrames: vi.fn(async () => ({
    Frames: [],
    Offset: 0,
    Limit: 200,
    Total: 0,
    EOF: true,
  })),
  clearFecbusFrames: vi.fn(async () => undefined),
}))

vi.mock('../services/serialService', () => ({
  serialService: serialServiceMock,
}))

describe('fecbusStore helpers', () => {
  it('parses and formats spaced hex bytes', () => {
    expect(parseHexBytes('0f 0A\nff')).toEqual([0x0f, 0x0a, 0xff])
    expect(formatHexBytes([0x0f, 0x0a, 0xff])).toBe('0f 0a ff')
  })

  it('exposes table C.2 named and custom function options without reserved codes', () => {
    expect(fecbusFunctionOptions.some(option => option.value === FunctionCode.FunctionQueryProtocolVersion)).toBe(true)
    expect(fecbusFunctionOptions.some(option => option.value === 16)).toBe(false)
    expect(fecbusFunctionOptions.some(option => option.value === 46 && option.custom)).toBe(true)
  })
})

describe('fecbusStore workspace state', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    setActivePinia(createPinia())
  })

  it('opens a FECbus session with serial defaults', async () => {
    const store = useFecbusStore()
    store.portForm = {
      ...store.portForm,
      sessionId: 'fec-1',
      port: '/tmp/ttyF0',
      role: 'master',
      baudRate: 9600,
    }

    const id = await store.openSession()

    expect(id).toBe('fec-1')
    expect(serialServiceMock.openFecbusSession).toHaveBeenCalledWith(expect.objectContaining({
      ID: 'fec-1',
      Role: SessionRole.SessionRoleMaster,
      Config: expect.objectContaining({
        PortName: '/tmp/ttyF0',
        BaudRate: 9600,
        DataBits: 8,
        StopBits: '1',
        Parity: 'none',
      }),
    }))
  })

  it('builds and sends a request with normalized header fields', async () => {
    const store = useFecbusStore()
    store.portForm = { ...store.portForm, sessionId: 'fec-1', port: '/tmp/ttyF0' }
    await store.openSession()
    store.sendForm = {
      ...store.sendForm,
      targetAddress: 2,
      sourceAddress: 1,
      messageNumber: 9,
      functionCode: FunctionCode.FunctionQueryProtocolVersion,
      payloadHex: '01 02',
      expectAnswer: true,
    }

    await store.sendRequest()

    expect(serialServiceMock.fecbusSendRequest).toHaveBeenCalledWith(expect.objectContaining({
      SessionID: 'fec-1',
      FrameType: FrameType.FrameTypeRequest,
      TargetAddress: 2,
      SourceAddress: 1,
      MessageNumber: 9,
      Function: FunctionCode.FunctionQueryProtocolVersion,
      DataHex: '2c 01 02',
      ExpectAnswer: true,
    }))
    expect(store.history[0].ID).toBe('tx-1')
  })

  it('exports and restores sessions, forms, history, and frame cache', () => {
    const store = useFecbusStore()
    store.restoreState({
      ...defaultFecbusWorkspaceState(),
      activeSessionId: 'fec-1',
      sessions: [sampleSession('fec-1', 'slave')],
      portForm: { ...store.portForm, port: '/tmp/ttyF0', role: 'slave' },
      slaveForm: { address: 2, statusCode: StatusCode.StatusBusy, autoStatusAnswer: true, acceptBroadcast: false },
      slaveUnits: [
        { address: 2, statusCode: StatusCode.StatusBusy, autoStatusAnswer: true, acceptBroadcast: false },
        { address: 3, statusCode: StatusCode.StatusProcessing, autoStatusAnswer: true, acceptBroadcast: true },
      ],
      customFunctions: [{
        Code: 8 as FunctionCode,
        Name: '厂商测试',
        Description: '',
        Direction: 'custom',
        Answer: true,
        Fields: [{ Key: 'value', Label: '测试值', Offset: 1, Length: 2, Type: 'uint16', Endian: 'little', Enum: null, Meaning: '' }],
      }],
      frameFilters: { 'fec-1': { direction: 'rx', search: '0f' } },
      framePages: {
        'fec-1': {
          Frames: [{
            Seq: 1,
            SessionID: 'fec-1',
            Direction: 'rx',
            Frame: null as any,
            Hex: '7e 01',
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
      history: [{
        ID: 'tx-1',
        SessionID: 'fec-1',
        StartedAt: '' as any,
        CompletedAt: '' as any,
        Request: null as any,
        Response: null,
        RequestFrameHex: '7e 00',
        ResponseFrameHex: '',
        BytesWritten: 10,
        Error: '',
      }],
    })

    const snapshot = store.exportState()
    store.resetWorkspace()
    store.restoreState(snapshot)

    expect(store.activeSessionId).toBe('fec-1')
    expect(store.sessionList[0].Role).toBe(SessionRole.SessionRoleSlave)
    expect(store.portForm.port).toBe('/tmp/ttyF0')
    expect(store.slaveForm.statusCode).toBe(StatusCode.StatusBusy)
    expect(store.slaveUnits.map(unit => unit.address)).toEqual([2, 3])
    expect(store.customFunctions[0].Name).toBe('厂商测试')
    expect(store.frameFilters['fec-1'].search).toBe('0f')
    expect(store.framePages['fec-1'].Total).toBe(1)
    expect(store.history[0].ID).toBe('tx-1')
  })

  it('starts a slave with multiple configured addresses', async () => {
    const store = useFecbusStore()
    store.portForm = { ...store.portForm, sessionId: 'fec-1', port: '/tmp/ttyF0', role: 'slave' }
    await store.openSession()
    store.slaveUnits = [
      { address: 2, statusCode: StatusCode.StatusReceivedOK, autoStatusAnswer: true, acceptBroadcast: true },
      { address: 3, statusCode: StatusCode.StatusBusy, autoStatusAnswer: true, acceptBroadcast: false },
    ]

    await store.startSlave()

    expect(serialServiceMock.startFecbusSlave).toHaveBeenCalledWith(expect.objectContaining({
      SessionID: 'fec-1',
      Units: [
        expect.objectContaining({ Address: 2, DefaultStatus: StatusCode.StatusReceivedOK }),
        expect.objectContaining({ Address: 3, DefaultStatus: StatusCode.StatusBusy }),
      ],
    }))
  })

  it('uses restored frame cache for local-only sessions without querying backend', async () => {
    const store = useFecbusStore()
    const cachedPage = {
      Frames: [{
        Seq: 1,
        SessionID: 'fec-1',
        Direction: 'rx',
        Frame: null as any,
        Hex: '7e 01',
        Error: '',
        Timestamp: '' as any,
        Annotated: sampleAnnotation(),
      }],
      Offset: 0,
      Limit: 200,
      Total: 1,
      EOF: true,
    }
    store.restoreState({
      ...defaultFecbusWorkspaceState(),
      activeSessionId: 'fec-1',
      sessions: [sampleSession('fec-1', 'master')],
      framePages: { 'fec-1': cachedPage },
    })

    const page = await store.queryFrames('fec-1')

    expect(page).toEqual(cachedPage)
    expect(serialServiceMock.queryFecbusFrames).not.toHaveBeenCalled()
    expect(store.error).toBeNull()
  })

  it('passes custom definitions when querying annotated frames', async () => {
    const store = useFecbusStore()
    store.portForm = { ...store.portForm, sessionId: 'fec-1', port: '/tmp/ttyF0' }
    await store.openSession()
    store.customFunctions = [{
      Code: 8 as FunctionCode,
      Name: '厂商测试',
      Description: '',
      Direction: 'custom',
      Answer: true,
      Fields: [{ Key: 'value', Label: '测试值', Offset: 1, Length: 2, Type: 'uint16', Endian: 'little', Enum: null, Meaning: '' }],
    }]

    await store.queryFrames('fec-1')

    expect(serialServiceMock.queryFecbusFrames).toHaveBeenCalledWith(expect.objectContaining({
      SessionID: 'fec-1',
      Custom: [expect.objectContaining({ Name: '厂商测试' })],
    }))
  })
})

function sampleSession(id: string, role: string) {
  return {
    ID: id,
    Name: id,
    Role: role === 'slave' ? SessionRole.SessionRoleSlave : SessionRole.SessionRoleMaster,
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
    SourceAddress: role === 'slave' ? 2 : 1,
    TargetAddress: role === 'slave' ? 1 : 2,
    SlaveUnits: role === 'slave' ? [{ Address: 2, DefaultStatus: StatusCode.StatusReceivedOK, AutoStatusAnswer: true, AcceptBroadcast: true }] : [],
    StartedAt: '' as any,
    StoppedAt: '' as any,
    LastError: '',
  }
}

function sampleAnnotation() {
  return {
    Segments: [],
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
