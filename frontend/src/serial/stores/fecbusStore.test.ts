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
} from '../../../bindings/github.com/littepointR/portweave/internal/modules/serial/fecbus/models.js'

const serialServiceMock = vi.hoisted(() => ({
  listFecbusSessions: vi.fn(async (): Promise<any[]> => []),
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

  it('rejects malformed hex and clamps formatted bytes', () => {
    expect(() => parseHexBytes('abc')).toThrow('HEX 长度必须为偶数')
    expect(() => parseHexBytes('zz')).toThrow('HEX 包含无效字符')
    expect(formatHexBytes([-1, 0x34, 0x100])).toBe('00 34 ff')
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

  it('builds structured request payloads from built-in and custom field definitions', async () => {
    const store = useFecbusStore()
    store.portForm = { ...store.portForm, sessionId: 'fec-1', port: '/tmp/ttyF0' }
    await store.openSession()
    serialServiceMock.fecbusSendRequest.mockClear()
    store.sendForm = {
      ...store.sendForm,
      inputMode: 'structured',
      functionCode: FunctionCode.FunctionQueryDeviceStatus,
      structuredFields: {
        controller_id: '1',
        unit_id: '2',
        device_id: '3',
        channel_id: '4',
      },
    }

    await store.sendRequest()

    expect(serialServiceMock.fecbusSendRequest).toHaveBeenLastCalledWith(expect.objectContaining({
      DataHex: '22 01 02 03 04',
    }))

    store.upsertCustomFunction({
      Code: 8 as FunctionCode,
      Name: '厂商扩展',
      Description: '',
      Direction: 'custom',
      Answer: true,
      Fields: [
        { Key: 'word', Label: 'Word', Offset: 1, Length: 2, Type: 'uint16', Endian: 'big', Enum: null, Meaning: '' },
        { Key: 'raw', Label: 'Raw', Offset: 3, Length: 2, Type: 'hex', Endian: 'little', Enum: null, Meaning: '' },
        { Key: 'name', Label: 'Name', Offset: 5, Length: 3, Type: 'utf8', Endian: 'little', Enum: null, Meaning: '' },
      ],
    })
    store.sendForm = {
      ...store.sendForm,
      functionCode: 8,
      structuredFields: { word: '52', raw: 'aa bb', name: 'AZ' },
    }

    await store.sendRequest()

    expect(serialServiceMock.fecbusSendRequest).toHaveBeenLastCalledWith(expect.objectContaining({
      DataHex: '08 00 34 aa bb 41 5a 00',
    }))
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

  it('updates slave units, filters, and frame cache through backend sessions', async () => {
    const store = useFecbusStore()
    store.portForm = { ...store.portForm, sessionId: 'fec-1', port: '/tmp/ttyF0', role: 'slave' }
    await store.openSession()
    store.slaveForm = {
      address: 4,
      statusCode: StatusCode.StatusBusy,
      autoStatusAnswer: false,
      acceptBroadcast: false,
    }
    const nextPage = {
      Frames: [{
        Seq: 2,
        SessionID: 'fec-1',
        Direction: 'rx',
        Frame: null as any,
        Hex: '7e 02',
        Error: '',
        Timestamp: '' as any,
        Annotated: sampleAnnotation(),
      }],
      Offset: 0,
      Limit: 200,
      Total: 1,
      EOF: true,
    }
    serialServiceMock.queryFecbusFrames.mockResolvedValueOnce(nextPage as any)

    await store.addSlaveUnit({ ...store.slaveForm })
    await store.updateSlaveState()
    await store.stopSlave()
    store.setFrameFilter('fec-1', { direction: 'rx' })
    store.setFrameFilter('fec-1', { search: '7e' })
    const page = await store.queryFrames('fec-1')
    await store.clearFrames('fec-1')
    await store.removeSlaveUnit(4)

    expect(serialServiceMock.addFecbusSlaveUnit).toHaveBeenCalledWith('fec-1', expect.objectContaining({
      Address: 4,
      DefaultStatus: StatusCode.StatusBusy,
      AutoStatusAnswer: false,
      AcceptBroadcast: false,
    }))
    expect(serialServiceMock.updateFecbusSlaveState).toHaveBeenCalledWith('fec-1', expect.objectContaining({
      Address: 4,
      DefaultStatus: StatusCode.StatusBusy,
    }))
    expect(serialServiceMock.stopFecbusSlave).toHaveBeenCalledWith('fec-1')
    expect(serialServiceMock.queryFecbusFrames).toHaveBeenCalledWith(expect.objectContaining({
      SessionID: 'fec-1',
      Direction: 'rx',
      Search: '7e',
    }))
    expect(page).toEqual(nextPage)
    expect(serialServiceMock.clearFecbusFrames).toHaveBeenCalledWith('fec-1')
    expect(store.framePages['fec-1']).toEqual({ Frames: [], Offset: 0, Limit: 200, Total: 0, EOF: true })
    expect(serialServiceMock.removeFecbusSlaveUnit).toHaveBeenCalledWith('fec-1', 4)
  })

  it('surfaces validation and backend errors for FECbus actions', async () => {
    const store = useFecbusStore()

    await expect(store.openSession()).resolves.toBeNull()
    expect(store.error).toBe('请选择串口')

    store.portForm = { ...store.portForm, port: '/tmp/ttyF0' }
    serialServiceMock.openFecbusSession.mockRejectedValueOnce(undefined)
    await expect(store.openSession()).rejects.toBeUndefined()
    expect(store.error).toBe('Failed to open FECbus session')

    store.sessions.set('fec-1', sampleSession('fec-1', 'master'))
    store.setActiveSession('fec-1')
    serialServiceMock.closeFecbusSession.mockRejectedValueOnce(new Error('close denied'))
    await expect(store.closeSession()).rejects.toThrow('close denied')
    expect(store.error).toBe('close denied')

    serialServiceMock.queryFecbusFrames.mockRejectedValueOnce(undefined)
    await expect(store.queryFrames('fec-1')).rejects.toBeUndefined()
    expect(store.error).toBe('Failed to query FECbus frames')

    serialServiceMock.clearFecbusFrames.mockRejectedValueOnce(new Error('clear denied'))
    await expect(store.clearFrames('fec-1')).rejects.toThrow('clear denied')
    expect(store.error).toBe('clear denied')
  })

  it('keeps restored sessions local while normalizing slave and custom state', async () => {
    const store = useFecbusStore()
    store.restoreState({
      ...defaultFecbusWorkspaceState(),
      activeSessionId: 'fec-local',
      sessions: [sampleSession('fec-local', 'slave')],
      slaveUnits: [
        { address: 2, statusCode: StatusCode.StatusReceivedOK, autoStatusAnswer: true, acceptBroadcast: true },
      ],
      framePages: {
        'fec-local': {
          Frames: [{ Seq: 1, SessionID: 'fec-local', Direction: 'rx', Frame: null as any, Hex: '7e 01', Error: '', Timestamp: '' as any, Annotated: sampleAnnotation() }],
          Offset: 0,
          Limit: 200,
          Total: 1,
          EOF: true,
        },
      },
    })

    await expect(store.sendRequest()).resolves.toBeNull()
    expect(store.error).toContain('已恢复会话不能直接发送')
    await expect(store.startSlave()).resolves.toBeNull()
    expect(store.error).toContain('已恢复会话不能启动从站')
    await store.stopSlave()
    await store.updateSlaveState()
    expect(serialServiceMock.stopFecbusSlave).not.toHaveBeenCalled()
    expect(serialServiceMock.updateFecbusSlaveState).not.toHaveBeenCalled()

    await store.addSlaveUnit({ address: 1, statusCode: StatusCode.StatusBusy, autoStatusAnswer: false, acceptBroadcast: false })
    await store.addSlaveUnit({ address: 3, statusCode: StatusCode.StatusProcessing, autoStatusAnswer: true, acceptBroadcast: false })
    expect(store.slaveUnits.map(unit => unit.address)).toEqual([1, 2, 3])
    await store.removeSlaveUnit(1)
    await store.removeSlaveUnit(2)
    await store.removeSlaveUnit(3)
    expect(store.slaveUnits).toEqual([expect.objectContaining({ address: 2 })])

    await store.clearFrames('fec-local')
    expect(serialServiceMock.clearFecbusFrames).not.toHaveBeenCalled()
    expect(store.framePages['fec-local']).toEqual({ Frames: [], Offset: 0, Limit: 200, Total: 0, EOF: true })

    store.upsertCustomFunction({
      Code: 8 as FunctionCode,
      Name: '',
      Description: '',
      Direction: '',
      Answer: false,
      Fields: [{ Key: 'vendor', Label: '', Offset: 0, Length: 0, Type: '', Endian: '', Enum: null, Meaning: '' }],
    })
    expect(store.customFunctions[0]).toMatchObject({
      Code: 8,
      Name: '用户自定义',
      Direction: 'custom',
      Fields: [{ Key: 'vendor', Label: 'vendor', Offset: 1, Length: 1, Type: 'uint8', Endian: 'little', Meaning: '' }],
    })
    store.upsertCustomFunction({ Code: 8 as FunctionCode, Name: '更新定义', Description: '', Direction: 'custom', Answer: true, Fields: [] })
    expect(store.customFunctions).toEqual([expect.objectContaining({ Name: '更新定义' })])
    store.removeCustomFunction(8)
    expect(store.customFunctions).toHaveLength(0)
  })

  it('refreshes backend sessions and records refresh failures', async () => {
    const store = useFecbusStore()
    serialServiceMock.listFecbusSessions.mockResolvedValueOnce([
      {
        ...sampleSession('fec-backend', 'master'),
        Role: '' as SessionRole,
        SlaveUnits: undefined,
      },
    ])

    await store.refreshSessions()

    expect(store.sessionList).toEqual([
      expect.objectContaining({
        ID: 'fec-backend',
        Role: SessionRole.SessionRoleMaster,
        SlaveUnits: [],
      }),
    ])
    expect(store.error).toBeNull()

    serialServiceMock.listFecbusSessions.mockRejectedValueOnce(undefined)
    await store.refreshSessions()

    expect(store.error).toBe('Failed to refresh FECbus sessions')
  })

  it('normalizes open-session fallbacks and closes the active backend session', async () => {
    const store = useFecbusStore()
    store.portForm = {
      ...store.portForm,
      sessionId: '',
      name: 'Fallback serial',
      port: '/tmp/ttyF0',
      role: 'slave',
      baudRate: 0,
      dataBits: 0,
      stopBits: '',
      parity: '',
      flowMode: '',
      timeoutMs: 1,
      retries: -4,
    }

    const id = await store.openSession()

    expect(id).toMatch(/^fecbus-/)
    expect(serialServiceMock.openFecbusSession).toHaveBeenCalledWith(expect.objectContaining({
      ID: expect.stringMatching(/^fecbus-/),
      Name: 'Fallback serial',
      Role: SessionRole.SessionRoleSlave,
      Config: expect.objectContaining({
        BaudRate: 1,
        DataBits: 8,
        StopBits: '1',
        Parity: 'none',
        FlowMode: 'none',
      }),
      TimeoutMs: 10,
      Retries: 0,
    }))

    store.sessions.set('fec-next', sampleSession('fec-next', 'master'))
    store.framePages = { [id!]: { Frames: [], Offset: 0, Limit: 200, Total: 0, EOF: true } }
    store.frameFilters = { [id!]: { direction: 'rx', search: '7e' } }

    await store.closeSession(id)
    await store.closeSession(null as any)

    expect(serialServiceMock.closeFecbusSession).toHaveBeenCalledWith(id)
    expect(store.sessionList.map(session => session.ID)).toEqual(['fec-next'])
    expect(store.activeSessionId).toBe('fec-next')
    expect(store.framePages[id!]).toBeUndefined()
    expect(store.frameFilters[id!]).toBeUndefined()
    expect(store.error).toBeNull()
  })

  it('guards missing sessions and propagates backend action failures', async () => {
    const store = useFecbusStore()

    await expect(store.sendRequest()).resolves.toBeNull()
    expect(store.error).toBe('请先打开 FECbus 会话')
    await expect(store.startSlave()).resolves.toBeNull()
    expect(store.error).toBe('请先打开 FECbus 会话')
    await store.stopSlave()
    await store.updateSlaveState()

    store.sessions.set('fec-1', sampleSession('fec-1', 'master'))
    store.setActiveSession('fec-1')

    serialServiceMock.fecbusSendRequest.mockRejectedValueOnce(undefined)
    await expect(store.sendRequest()).rejects.toBeUndefined()
    expect(store.error).toBe('Failed to send FECbus request')

    serialServiceMock.startFecbusSlave.mockRejectedValueOnce(undefined)
    await expect(store.startSlave()).rejects.toBeUndefined()
    expect(store.error).toBe('Failed to start FECbus slave')

    serialServiceMock.stopFecbusSlave.mockRejectedValueOnce(undefined)
    await expect(store.stopSlave()).rejects.toBeUndefined()
    expect(store.error).toBe('Failed to stop FECbus slave')

    serialServiceMock.updateFecbusSlaveState.mockRejectedValueOnce(undefined)
    await expect(store.updateSlaveState()).rejects.toBeUndefined()
    expect(store.error).toBe('Failed to update FECbus slave state')
  })

  it('uses local frame-cache fallbacks and no-op frame guards', async () => {
    const store = useFecbusStore()

    await expect(store.queryFrames(null as any)).resolves.toBeNull()
    await expect(store.clearFrames(null as any)).resolves.toBeUndefined()

    store.restoreState({
      ...defaultFecbusWorkspaceState(),
      activeSessionId: 'fec-local',
      sessions: [sampleSession('fec-local', 'master')],
    })

    const page = await store.queryFrames('fec-local')

    expect(page).toEqual({ Frames: [], Offset: 0, Limit: 200, Total: 0, EOF: true })
    expect(store.framePages['fec-local']).toEqual(page)
    expect(serialServiceMock.queryFecbusFrames).not.toHaveBeenCalled()
  })

  it('updates existing slave units and builds structured payload edge values', async () => {
    const store = useFecbusStore()
    await store.addSlaveUnit({ address: 5, statusCode: StatusCode.StatusBusy, autoStatusAnswer: false, acceptBroadcast: false })
    await store.addSlaveUnit({ address: 5, statusCode: StatusCode.StatusProcessing, autoStatusAnswer: true, acceptBroadcast: true })

    expect(store.slaveUnits).toEqual(expect.arrayContaining([expect.objectContaining({
      address: 5,
      statusCode: StatusCode.StatusProcessing,
      autoStatusAnswer: true,
      acceptBroadcast: true,
    })]))

    store.sessions.set('fec-1', sampleSession('fec-1', 'master'))
    store.setActiveSession('fec-1')
    store.customFunctions = [{
      Code: 9 as FunctionCode,
      Name: 'Raw edge fields',
      Description: '',
      Direction: 'custom',
      Answer: true,
      Fields: [
        { Key: 'skipUndefined', Label: 'Skip undefined', Offset: 1, Length: 1, Type: 'uint8', Endian: 'little', Enum: null, Meaning: '' },
        { Key: 'skipEmpty', Label: 'Skip empty', Offset: 2, Length: 1, Type: 'uint8', Endian: 'little', Enum: null, Meaning: '' },
        { Key: 'ignored', Label: 'Ignored', Offset: 0, Length: 0, Type: 'uint8', Endian: 'little', Enum: null, Meaning: '' },
        { Key: 'nan', Label: 'NaN', Offset: 1, Length: 1, Type: 'uint8', Endian: 'little', Enum: null, Meaning: '' },
        { Key: 'word', Label: 'Word', Offset: 2, Length: 2, Type: 'uint16', Endian: 'little', Enum: null, Meaning: '' },
      ],
    }]
    store.sendForm = {
      ...store.sendForm,
      inputMode: 'structured',
      functionCode: 9,
      structuredFields: {
        skipEmpty: '',
        ignored: '7',
        nan: 'not-a-number',
        word: '4660',
      },
    }

    await store.sendRequest()

    expect(serialServiceMock.fecbusSendRequest).toHaveBeenLastCalledWith(expect.objectContaining({
      DataHex: '09 00 ff 12',
    }))
  })

  it('restores partial snapshots with defaulted collections and function fields', () => {
    const store = useFecbusStore()

    store.restoreState(undefined)
    expect(store.activeSessionId).toBeNull()
    expect(store.sessionList).toEqual([])
    expect(store.portForm).toEqual(expect.objectContaining({
      sessionId: expect.stringMatching(/^fecbus-/),
      port: '',
      role: 'master',
    }))

    store.restoreState({
      ...defaultFecbusWorkspaceState(),
      sessions: undefined,
      slaveUnits: [{ address: 99, statusCode: undefined, autoStatusAnswer: undefined, acceptBroadcast: undefined }],
      customFunctions: undefined,
      frameFilters: undefined,
      framePages: undefined,
      history: undefined,
    } as any)

    expect(store.sessionList).toEqual([])
    expect(store.slaveUnits).toEqual([{
      address: 63,
      statusCode: StatusCode.StatusReceivedOK,
      autoStatusAnswer: true,
      acceptBroadcast: true,
    }])
    expect(store.customFunctions).toEqual([])
    expect(store.frameFilters).toEqual({})
    expect(store.framePages).toEqual({})
    expect(store.history).toEqual([])

    store.upsertCustomFunction({
      Code: 300 as FunctionCode,
      Name: '',
      Description: undefined as any,
      Direction: '',
      Answer: false,
      Fields: undefined as any,
    })
    expect(store.customFunctions).toEqual([expect.objectContaining({
      Code: 255,
      Name: '用户自定义',
      Description: '',
      Direction: 'custom',
      Fields: [],
    })])
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
