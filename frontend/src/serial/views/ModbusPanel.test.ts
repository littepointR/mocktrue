import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import type { ComponentPublicInstance } from 'vue'
import ModbusPanel from './ModbusPanel.vue'
import { useModbusStore } from '../stores/modbusStore'
import { serialService } from '../services/serialService'
import { FrameMode, SessionRole } from '../../../bindings/github.com/littepointR/portweave/internal/modules/serial/modbus/models.js'

vi.mock('../services/serialService', () => ({
  serialService: {
    enumeratePorts: vi.fn(async () => []),
    listPorts: vi.fn(async () => []),
    listModbusSessions: vi.fn(async () => []),
    openModbusSession: vi.fn(async () => null),
    modbusMasterRequest: vi.fn(async () => null),
    modbusReadRegisters: vi.fn(async () => ({ Transaction: null, RawRegisters: [], Bits: [], Values: [] })),
    modbusScanUnitIDs: vi.fn(async () => ({ SessionID: 'modbus-master', ActiveUnitIDs: [], Results: [] })),
    modbusScanRegisters: vi.fn(async () => ({ SessionID: 'modbus-master', UnitID: 1, Values: [], Ranges: [] })),
    startModbusSlave: vi.fn(async () => sampleSession('modbus-slave', SessionRole.SessionRoleSlave)),
    stopModbusSlave: vi.fn(async () => undefined),
    updateModbusSlaveUnitData: vi.fn(async () => undefined),
    addModbusSlaveUnit: vi.fn(async () => undefined),
    removeModbusSlaveUnit: vi.fn(async () => undefined),
    listModbusSlaveUnits: vi.fn(async () => []),
  },
}))

describe('ModbusPanel', () => {
  const slowCoverageTestTimeoutMs = 15000

  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('opens and refreshes creation sessions through the toolbar', async () => {
    const store = useModbusStore()
    store.portForm.port = '/tmp/ttyM0'
    vi.mocked(serialService.openModbusSession).mockResolvedValueOnce(sampleSession('modbus-open', SessionRole.SessionRoleMaster) as any)

    const wrapper = mount(ModbusPanel, {
      global: { stubs },
    })

    await findButton(wrapper, '刷新').trigger('click')
    await flushPromises()
    expect(serialService.listModbusSessions).toHaveBeenCalled()

    await findButton(wrapper, '打开').trigger('click')
    await flushPromises()

    expect(serialService.openModbusSession).toHaveBeenCalledWith(expect.objectContaining({
      Config: expect.objectContaining({ PortName: '/tmp/ttyM0' }),
      Role: SessionRole.SessionRoleMaster,
    }))
    expect(wrapper.emitted('opened')?.[0]).toEqual(['modbus-open'])
    expect(store.activeSessionId).toBe('modbus-open')
    expect(wrapper.text()).toContain('RX 0 / TX 0')
  })

  it('only renders session creation controls in the operation panel', () => {
    const wrapper = mount(ModbusPanel, {
      global: { stubs },
    })
    const text = wrapper.text()

    expect(text).toContain('Modbus 调试')
    expect(text).toContain('角色')
    expect(text).toContain('打开')
    expect(text).not.toContain('主站')
    expect(text).not.toContain('寄存器读取')
    expect(text).not.toContain('扫描')
    expect(text).not.toContain('从站')
    expect(text).not.toContain('历史')
  })

  it('renders a master workbench with toolbar, register controls, and data grid in tab content', () => {
    const store = useModbusStore()
    store.restoreState({
      ...store.exportState(),
      activeSessionId: 'modbus-master',
      sessions: [sampleSession('modbus-master', SessionRole.SessionRoleMaster)],
    })
    store.registerReadResult = {
      Transaction: null,
      RawRegisters: [24, 42],
      Bits: [],
      Values: [{
        Address: 0,
        Mapping: {
          Address: 0,
          DataType: 'uint16' as any,
          WordOrder: 'big' as any,
          Length: 0,
          ScalingFactor: 0,
          Comment: '温度',
          Interpolate: null,
          GroupEnd: false,
        },
        Value: {
          DataType: 'uint16' as any,
          Raw: [24],
          Display: '24',
          Numeric: 24,
        },
        Error: '',
      }],
    }

    const wrapper = mount(ModbusPanel, {
      props: { variant: 'tab', sessionId: 'modbus-master' },
      global: { stubs },
    })
    const text = wrapper.text()

    expect(text).toContain('Modbus 主站')
    expect(text).toContain('读')
    expect(text).toContain('轮询')
    expect(text).toContain('扫描 Unit')
    expect(text).toContain('扫描寄存器')
    expect(text).toContain('Raw')
    expect(text).toContain('日志')
    expect(wrapper.find('[data-testid="modbus-master-unit-tabs"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="modbus-master-unit-tab-1"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="modbus-master-unit-tabs"]').text()).toContain('主站 Unit')
    expect(wrapper.find('[data-testid="modbus-master-config-summary"]').text()).toContain('Holding Registers')
    expect(wrapper.find('.modbus-panel__collapse-state').text()).toBe('展开')
    expect(text).toContain('配置')
    expect(text).not.toContain('轮询 ms')
    expect(text).toContain('Coils')
    expect(text).toContain('Discrete Inputs')
    expect(text).toContain('Input Registers')
    expect(text).toContain('Holding Registers')
    expect(text).toContain('映射')
    expect(text).toContain('别名')
    expect(text).toContain('字序')
    expect(text).toContain('系数')
    expect(text).toContain('温度')
    expect(text).not.toContain('暂无')
    expect(text).not.toContain('未映射')
    expect(wrapper.find('[data-testid="modbus-master-table-coils"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="modbus-master-table-discrete_inputs"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="modbus-master-table-coils"]').classes()).toContain('modbus-panel__bool-table')
    expect(wrapper.find('[data-testid="modbus-master-table-discrete_inputs"]').classes()).toContain('modbus-panel__bool-table')
    expect(wrapper.find('[data-testid="modbus-master-table-input_registers"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="modbus-master-table-holding_registers"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="modbus-master-table-holding_registers"] [data-testid="resize-handle-address"]').exists()).toBe(true)
    expect(wrapper.find('.modbus-panel__mapping-band').exists()).toBe(true)
    expect(wrapper.find('.modbus-panel__mapping-editor').exists()).toBe(false)
  })

  it('selects a master register card when clicking inside its table', async () => {
    const store = useModbusStore()
    store.restoreState({
      ...store.exportState(),
      activeSessionId: 'modbus-master',
      sessions: [sampleSession('modbus-master', SessionRole.SessionRoleMaster)],
    })
    expect(store.masterGrid.registerType).toBe('holding_registers')

    const wrapper = mount(ModbusPanel, {
      props: { variant: 'tab', sessionId: 'modbus-master' },
      global: { stubs },
    })

    await wrapper.find('[data-testid="modbus-master-table-coils"] tbody tr').trigger('click')

    expect(store.masterGrid.registerType).toBe('coils')
    expect(wrapper.find('[data-testid="modbus-master-card-coils"]').classes()).toContain('modbus-panel__register-card--active')
  })

  it('adds and switches master Unit tabs without sharing register rows', async () => {
    const store = useModbusStore()
    store.restoreState({
      ...store.exportState(),
      activeSessionId: 'modbus-master',
      sessions: [sampleSession('modbus-master', SessionRole.SessionRoleMaster)],
    })

    const wrapper = mount(ModbusPanel, {
      props: { variant: 'tab', sessionId: 'modbus-master' },
      global: { stubs },
    })

    store.masterRegisterTables.find(table => table.type === 'holding_registers')!.rows[0].value = '11'
    await wrapper.find('[data-testid="modbus-add-master-unit"]').trigger('click')
    await wrapper.find('[data-testid="modbus-new-master-unit-id"] input').setValue('2')
    await wrapper.find('[data-testid="modbus-confirm-master-unit"]').trigger('click')

    expect(store.activeMasterUnitId).toBe(2)
    expect(store.masterRegisterTables.find(table => table.type === 'holding_registers')?.rows[0].value).toBe('0')
    store.masterRegisterTables.find(table => table.type === 'holding_registers')!.rows[0].value = '22'

    await wrapper.find('[data-testid="modbus-master-unit-tab-1"]').trigger('click')
    expect(store.activeMasterUnitId).toBe(1)
    expect(store.masterRegisterTables.find(table => table.type === 'holding_registers')?.rows[0].value).toBe('11')

    await wrapper.find('[data-testid="modbus-master-unit-tab-2"]').trigger('click')
    expect(store.masterRegisterTables.find(table => table.type === 'holding_registers')?.rows[0].value).toBe('22')
  }, slowCoverageTestTimeoutMs)

  it('uses the same header layout for every master register card', () => {
    const store = useModbusStore()
    store.restoreState({
      ...store.exportState(),
      activeSessionId: 'modbus-master',
      sessions: [sampleSession('modbus-master', SessionRole.SessionRoleMaster)],
    })

    const wrapper = mount(ModbusPanel, {
      props: { variant: 'tab', sessionId: 'modbus-master' },
      global: { stubs },
    })

    const headers = wrapper.findAll('.modbus-panel__register-card > .modbus-panel__section-header')
    expect(headers).toHaveLength(4)
    for (const header of headers) {
      expect(header.find('.modbus-panel__section-title').exists()).toBe(true)
      expect(header.findAll('.modbus-panel__toolbar-group button')).toHaveLength(3)
      expect(header.text()).toContain('BE')
      expect(header.text()).toContain('LE')
      expect(header.text()).toContain('添加')
    }
  })

  it('renders master register cards as ordered full-width sections', () => {
    const store = useModbusStore()
    store.restoreState({
      ...store.exportState(),
      activeSessionId: 'modbus-master',
      sessions: [sampleSession('modbus-master', SessionRole.SessionRoleMaster)],
    })

    const wrapper = mount(ModbusPanel, {
      props: { variant: 'tab', sessionId: 'modbus-master' },
      global: { stubs },
    })

    const cards = wrapper.findAll('[data-testid^="modbus-master-card-"]')
    expect(cards).toHaveLength(4)
    expect(cards.map(card => card.attributes('data-testid'))).toEqual([
      'modbus-master-card-coils',
      'modbus-master-card-discrete_inputs',
      'modbus-master-card-input_registers',
      'modbus-master-card-holding_registers',
    ])
  })

  it('keeps mapping fields inline in every master register table', () => {
    const store = useModbusStore()
    store.restoreState({
      ...store.exportState(),
      activeSessionId: 'modbus-master',
      sessions: [sampleSession('modbus-master', SessionRole.SessionRoleMaster)],
    })

    const wrapper = mount(ModbusPanel, {
      props: { variant: 'tab', sessionId: 'modbus-master' },
      global: { stubs },
    })

    const headers = wrapper.find('[data-testid="modbus-master-table-holding_registers"] thead tr')
      .findAll('th')
      .map(header => header.text())
    expect(headers).toEqual(['地址', '别名', '值', 'Raw', '映射', '类型', '字序', '长度', '系数', ''])
    expect(wrapper.find('.modbus-panel__mapping-editor').exists()).toBe(false)
    expect(store.masterRegisterTables.find(table => table.type === 'input_registers')?.mappings[0].comment).toBe('寄存器 0')
    expect(wrapper.find('[data-testid="modbus-mapping-alias-input_registers-0"]').exists()).toBe(true)
  })

  it('makes continuation rows read-only for multi-register mappings', () => {
    const store = useModbusStore()
    store.restoreState({
      ...store.exportState(),
      activeSessionId: 'modbus-master',
      sessions: [sampleSession('modbus-master', SessionRole.SessionRoleMaster)],
    })

    const wrapper = mount(ModbusPanel, {
      props: { variant: 'tab', sessionId: 'modbus-master' },
      global: { stubs },
    })

    const startRow = wrapper.find('[data-testid="modbus-master-row-holding_registers-0"]')
    const continuationRow = wrapper.find('[data-testid="modbus-master-row-holding_registers-1"]')
    expect(startRow.text()).toContain('起')
    expect(continuationRow.text()).toContain('止')
    expect(startRow.attributes('data-readonly')).toBe('false')
    expect(continuationRow.attributes('data-readonly')).toBe('true')
    expect(startRow.find('[data-testid="modbus-remove-mapping-holding_registers-0"]').exists()).toBe(true)
    expect(continuationRow.find('[data-testid="modbus-remove-mapping-holding_registers-1"]').exists()).toBe(false)
    expect(continuationRow.find('[data-testid="modbus-add-mapping-holding_registers-1"]').exists()).toBe(false)
  })

  it('marks rows from the same mapping with the same color and compact table density', () => {
    const store = useModbusStore()
    store.restoreState({
      ...store.exportState(),
      activeSessionId: 'modbus-master',
      sessions: [sampleSession('modbus-master', SessionRole.SessionRoleMaster)],
    })

    const wrapper = mount(ModbusPanel, {
      props: { variant: 'tab', sessionId: 'modbus-master' },
      global: { stubs },
    })

    const startRow = wrapper.find('[data-testid="modbus-master-row-holding_registers-0"]')
    const continuationRow = wrapper.find('[data-testid="modbus-master-row-holding_registers-1"]')
    const startColorClass = startRow.classes().find(name => name.startsWith('modbus-panel__mapping-band--'))
    const continuationColorClass = continuationRow.classes().find(name => name.startsWith('modbus-panel__mapping-band--'))

    expect(startRow.classes()).toContain('modbus-panel__mapping-band')
    expect(startColorClass).toBeTruthy()
    expect(continuationColorClass).toBe(startColorClass)
    expect(wrapper.find('[data-testid="modbus-master-table-holding_registers"]').classes()).toContain('modbus-panel__dense-table')
  })

  it('adds a mapping inline from an unmapped master register row', async () => {
    const store = useModbusStore()
    store.restoreState({
      ...store.exportState(),
      activeSessionId: 'modbus-master',
      sessions: [sampleSession('modbus-master', SessionRole.SessionRoleMaster)],
    })
    const coilsTable = store.masterRegisterTables.find(table => table.type === 'coils')!
    expect(coilsTable.mappings).toHaveLength(0)

    const wrapper = mount(ModbusPanel, {
      props: { variant: 'tab', sessionId: 'modbus-master' },
      global: { stubs },
    })

    await wrapper.find('[data-testid="modbus-add-mapping-coils-0"]').trigger('click')

    expect(coilsTable.mappings).toHaveLength(1)
    expect(coilsTable.mappings[0]).toMatchObject({ address: 0, dataType: 'uint16' })
    expect(wrapper.find('[data-testid="modbus-master-row-coils-0"]').text()).toContain('映射')
    expect(wrapper.text()).not.toContain('未映射')
  })

  it('expands master configuration controls on demand', async () => {
    const store = useModbusStore()
    store.restoreState({
      ...store.exportState(),
      activeSessionId: 'modbus-master',
      sessions: [sampleSession('modbus-master', SessionRole.SessionRoleMaster)],
    })

    const wrapper = mount(ModbusPanel, {
      props: { variant: 'tab', sessionId: 'modbus-master' },
      global: { stubs },
    })

    expect(wrapper.text()).not.toContain('轮询 ms')
    await wrapper.find('[data-testid="modbus-master-config-toggle"]').trigger('click')
    expect(wrapper.text()).toContain('轮询 ms')
    expect(wrapper.text()).toContain('超时 ms')
    expect(wrapper.text()).toContain('重试')
  })

  it('renders arbitrary master request form with all supported function options', async () => {
    const store = useModbusStore()
    store.restoreState({
      ...store.exportState(),
      activeSessionId: 'modbus-master',
      sessions: [sampleSession('modbus-master', SessionRole.SessionRoleMaster)],
    })

    const wrapper = mount(ModbusPanel, {
      props: { variant: 'tab', sessionId: 'modbus-master' },
      global: { stubs },
    })

    expect(wrapper.find('[data-testid="modbus-master-request-toggle"]').exists()).toBe(true)
    expect(wrapper.text()).not.toContain('线圈值')

    await wrapper.find('[data-testid="modbus-master-request-toggle"]').trigger('click')

    const form = wrapper.find('[data-testid="modbus-master-request-form"]')
    expect(form.exists()).toBe(true)
    expect(form.text()).toContain('Unit ID')
    expect(form.text()).toContain('功能码')
    expect(form.text()).toContain('地址模式')
    expect(form.text()).toContain('地址')
    expect(form.text()).toContain('数量')
    expect(form.text()).toContain('超时 ms')
    expect(form.text()).toContain('重试')
    expect(form.text()).toContain('发送请求')
    expect(form.find('[data-testid="modbus-master-request-function"]').attributes('data-functions')).toBe('1,2,3,4,5,6,15,16')
  })

  it('shows request value fields for single and multi-write function codes', async () => {
    const store = useModbusStore()
    store.restoreState({
      ...store.exportState(),
      activeSessionId: 'modbus-master',
      sessions: [sampleSession('modbus-master', SessionRole.SessionRoleMaster)],
    })
    const wrapper = mount(ModbusPanel, {
      props: { variant: 'tab', sessionId: 'modbus-master' },
      global: { stubs },
    })
    await wrapper.find('[data-testid="modbus-master-request-toggle"]').trigger('click')

    store.masterForm.functionCode = 5
    await wrapper.vm.$nextTick()
    expect(wrapper.find('[data-testid="modbus-master-request-value"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="modbus-master-request-coil-values"]').exists()).toBe(false)
    expect(wrapper.find('[data-testid="modbus-master-request-register-values"]').exists()).toBe(false)

    store.masterForm.functionCode = 15
    await wrapper.vm.$nextTick()
    expect(wrapper.find('[data-testid="modbus-master-request-value"]').exists()).toBe(false)
    expect(wrapper.find('[data-testid="modbus-master-request-coil-values"]').exists()).toBe(true)

    store.masterForm.functionCode = 16
    await wrapper.vm.$nextTick()
    expect(wrapper.find('[data-testid="modbus-master-request-coil-values"]').exists()).toBe(false)
    expect(wrapper.find('[data-testid="modbus-master-request-register-values"]').exists()).toBe(true)
  })

  it('sends a function 16 request through the existing store path', async () => {
    vi.mocked(serialService.modbusMasterRequest).mockResolvedValueOnce({
      ID: 'tx-1',
      SessionID: 'modbus-master',
      StartedAt: '',
      CompletedAt: '',
      UnitID: 2,
      Mode: FrameMode.FrameModeRTU,
      RequestPDU: { Function: 16, Data: '' },
      ResponsePDU: { Function: 16, Data: '' },
      RequestFrameHex: '02 10 00 0A 00 03',
      ResponseFrameHex: '02 10 00 0A 00 03',
      BytesWritten: 13,
      Response: {
        Function: 16,
        Exception: false,
        ExceptionCode: 0,
        Address: 10,
        Quantity: 3,
        Value: 0,
        Values: [],
        Bits: [],
        Raw: '',
      },
      Error: '',
    } as any)
    const store = useModbusStore()
    store.sessions.set('modbus-master', sampleSession('modbus-master', SessionRole.SessionRoleMaster))
    store.setActiveSession('modbus-master')
    const wrapper = mount(ModbusPanel, {
      props: { variant: 'tab', sessionId: 'modbus-master' },
      global: { stubs },
    })
    await wrapper.find('[data-testid="modbus-master-request-toggle"]').trigger('click')

    await wrapper.find('[data-testid="modbus-master-request-unit"] input').setValue('2')
    store.masterForm.functionCode = 16
    await wrapper.vm.$nextTick()
    await wrapper.find('[data-testid="modbus-master-request-address"] input').setValue('10')
    await wrapper.find('[data-testid="modbus-master-request-quantity"] input').setValue('3')
    await wrapper.find('[data-testid="modbus-master-request-register-values"] input').setValue('11 22 33')
    await wrapper.find('[data-testid="modbus-master-request-send"]').trigger('click')
    await flushPromises()

    expect(serialService.modbusMasterRequest).toHaveBeenCalledWith(expect.objectContaining({
      SessionID: 'modbus-master',
      UnitID: 2,
      Function: 16,
      Address: 10,
      Quantity: 3,
      RegisterValues: [11, 22, 33],
    }))
    expect(wrapper.find('[data-testid="modbus-master-request-result"]').text()).toContain('02 10 00 0A 00 03')
  }, slowCoverageTestTimeoutMs)

  it('runs master toolbar read, scan, polling, and log actions', async () => {
    vi.mocked(serialService.modbusReadRegisters).mockResolvedValueOnce({
      Transaction: sampleTransaction('read-1'),
      RawRegisters: [5, 6],
      Bits: [],
      Values: [],
    } as any)
    vi.mocked(serialService.modbusScanUnitIDs).mockResolvedValueOnce({
      SessionID: 'modbus-master',
      ActiveUnitIDs: [1, 2],
      Results: [{ UnitID: 2, Active: true, Error: '' }],
    } as any)
    vi.mocked(serialService.modbusScanRegisters).mockResolvedValueOnce({
      SessionID: 'modbus-master',
      UnitID: 1,
      Values: [{ Address: 0, Value: 5 }],
      Ranges: [],
    } as any)
    const store = useModbusStore()
    store.sessions.set('modbus-master', sampleSession('modbus-master', SessionRole.SessionRoleMaster))
    store.setActiveSession('modbus-master')
    const wrapper = mount(ModbusPanel, {
      props: { variant: 'tab', sessionId: 'modbus-master' },
      global: { stubs },
    })

    await findButton(wrapper, '读').trigger('click')
    await flushPromises()
    expect(serialService.modbusReadRegisters).toHaveBeenCalledWith(expect.objectContaining({
      SessionID: 'modbus-master',
      Function: 3,
    }))
    expect(wrapper.text()).toContain('5, 6')

    await findButton(wrapper, '扫描 Unit').trigger('click')
    await findButton(wrapper, '扫描寄存器').trigger('click')
    await flushPromises()
    expect(wrapper.text()).toContain('1, 2')
    expect(wrapper.text()).toContain('0=5')

    await findButton(wrapper, '日志').trigger('click')
    expect(store.masterGrid.logVisible).toBe(true)
    expect(wrapper.text()).toContain('01 03 00 00 00 02')

    await findButton(wrapper, '轮询').trigger('click')
    expect(store.registerReadForm.polling).toBe(true)
    await findButton(wrapper, '停止轮询').trigger('click')
    expect(store.registerReadForm.polling).toBe(false)
  }, slowCoverageTestTimeoutMs)

  it('updates inline master mappings and row actions from the table controls', async () => {
    const store = useModbusStore()
    store.sessions.set('modbus-master', sampleSession('modbus-master', SessionRole.SessionRoleMaster))
    store.setActiveSession('modbus-master')
    const table = store.masterRegisterTables.find(item => item.type === 'holding_registers')!
    const wrapper = mount(ModbusPanel, {
      props: { variant: 'tab', sessionId: 'modbus-master' },
      global: { stubs },
    })

    table.mappings[0].comment = '温度 A'
    table.mappings[0].dataType = 'float'
    table.mappings[0].wordOrder = 'little'
    table.mappings[0].length = 3
    table.mappings[0].scalingFactor = 0.5
    await wrapper.vm.$nextTick()

    expect(table.mappings[0]).toMatchObject({
      comment: '温度 A',
      dataType: 'float',
      wordOrder: 'little',
      length: 3,
      scalingFactor: 0.5,
    })

    await wrapper.find('[data-testid="modbus-remove-mapping-holding_registers-0"]').trigger('click')
    expect(table.mappings.some(row => row.address === 0)).toBe(false)
    await wrapper.find('[data-testid="modbus-add-mapping-holding_registers-0"]').trigger('click')
    expect(table.mappings.some(row => row.address === 0)).toBe(true)
    await wrapper.find('[data-testid="modbus-remove-row-holding_registers-0"]').trigger('click')
    expect(table.rows.some(row => row.address === 0)).toBe(false)
  })

  it('drives master table address, mapping, endian, and add-row controls', async () => {
    const store = useModbusStore()
    store.sessions.set('modbus-master', sampleSession('modbus-master', SessionRole.SessionRoleMaster))
    store.setActiveSession('modbus-master')
    const table = store.masterRegisterTables.find(item => item.type === 'holding_registers')!
    const wrapper = mount(ModbusPanel, {
      props: { variant: 'tab', sessionId: 'modbus-master' },
      global: { stubs },
    })
    const card = wrapper.find('[data-testid="modbus-master-card-holding_registers"]')

    await wrapper.find('[data-testid="modbus-master-row-holding_registers-0"] input').setValue('5')
    await wrapper.vm.$nextTick()
    expect(table.rows[0].address).toBe(5)
    expect(table.mappings[0].address).toBe(5)

    const editedRow = wrapper.find('[data-testid="modbus-master-row-holding_registers-5"]')
    const rowInputs = editedRow.findAll('input')
    await rowInputs[1].setValue('温度 B')
    ;(wrapper.findComponent('[data-testid="modbus-mapping-type-holding_registers-5"]') as VueWrapper<ComponentPublicInstance>).vm.$emit('update:value', 'double')
    ;(wrapper.findComponent('[data-testid="modbus-mapping-word-holding_registers-5"]') as VueWrapper<ComponentPublicInstance>).vm.$emit('update:value', 'little')
    await wrapper.vm.$nextTick()
    await rowInputs[2].setValue('4')
    await rowInputs[3].setValue('0.25')
    expect(table.mappings[0]).toMatchObject({
      address: 5,
      comment: '温度 B',
      dataType: 'double',
      wordOrder: 'little',
      length: 4,
      scalingFactor: 0.25,
    })

    await card.findAll('button').find(button => button.text() === 'BE')!.trigger('click')
    expect(store.masterGrid.littleEndian).toBe(false)
    expect(table.mappings.every(row => row.wordOrder === 'big')).toBe(true)
    await card.findAll('button').find(button => button.text() === 'LE')!.trigger('click')
    expect(store.masterGrid.littleEndian).toBe(true)
    expect(table.mappings.every(row => row.wordOrder === 'little')).toBe(true)

    const beforeRows = table.rows.length
    await card.findAll('button').find(button => button.text() === '添加')!.trigger('click')
    expect(table.rows).toHaveLength(beforeRows + 1)
    expect(table.length).toBeGreaterThanOrEqual(table.rows.length)
  })

  it('renders a slave workbench with unit tabs and four data sections instead of textareas', () => {
    const store = useModbusStore()
    store.restoreState({
      ...store.exportState(),
      activeSessionId: 'modbus-slave',
      sessions: [sampleSession('modbus-slave', SessionRole.SessionRoleSlave)],
    })

    const wrapper = mount(ModbusPanel, {
      props: { variant: 'tab', sessionId: 'modbus-slave' },
      global: { stubs },
    })
    const text = wrapper.text()

    expect(text).toContain('Modbus 从站')
    expect(text).toContain('Unit')
    expect(text).toContain('BE')
    expect(text).toContain('LE')
    expect(text).toContain('Coils')
    expect(text).toContain('Discrete Inputs')
    expect(text).toContain('Input Registers')
    expect(text).toContain('Holding Registers')
    expect(wrapper.findAll('textarea')).toHaveLength(0)
    expect(wrapper.find('[data-testid="modbus-slave-unit-tabs"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="modbus-slave-unit-tab-1"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="modbus-slave-unit-tabs"]').text()).toContain('从站 Unit')
    expect(wrapper.find('[data-testid="modbus-slave-table-coils"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="modbus-slave-table-discreteInputs"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="modbus-slave-table-coils"]').classes()).toContain('modbus-panel__bool-table')
    expect(wrapper.find('[data-testid="modbus-slave-table-discreteInputs"]').classes()).toContain('modbus-panel__bool-table')
    expect(wrapper.find('[data-testid="modbus-slave-table-inputRegisters"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="modbus-slave-table-holdingRegisters"]').exists()).toBe(true)
    expect(wrapper.find('.modbus-panel__row-actions').exists()).toBe(true)
  })

  it('explains slave backend persistence only covers addresses and values', () => {
    const store = useModbusStore()
    store.restoreState({
      ...store.exportState(),
      activeSessionId: 'modbus-slave',
      sessions: [sampleSession('modbus-slave', SessionRole.SessionRoleSlave)],
    })

    const wrapper = mount(ModbusPanel, {
      props: { variant: 'tab', sessionId: 'modbus-slave' },
      global: { stubs },
    })
    const hint = wrapper.find('[data-testid="modbus-slave-model-hint"]')

    expect(hint.exists()).toBe(true)
    expect(hint.text()).toContain('addresses')
    expect(hint.text()).toContain('values')
    expect(hint.text()).toContain('UI-only')
    expect(hint.text()).toContain('dataType')
    expect(hint.text()).toContain('comment')
    expect(hint.text()).toContain('word order')
  })

  it('adds and switches slave Unit tabs with independent data grids', async () => {
    const store = useModbusStore()
    store.restoreState({
      ...store.exportState(),
      activeSessionId: 'modbus-slave',
      sessions: [sampleSession('modbus-slave', SessionRole.SessionRoleSlave)],
    })

    const wrapper = mount(ModbusPanel, {
      props: { variant: 'tab', sessionId: 'modbus-slave' },
      global: { stubs },
    })

    store.slaveUnitGrids.find(unit => unit.unitId === 1)!.holdingRegisters[0].value = 11
    await wrapper.find('[data-testid="modbus-add-slave-unit"]').trigger('click')
    await wrapper.find('[data-testid="modbus-new-slave-unit-id"] input').setValue('2')
    await wrapper.find('[data-testid="modbus-confirm-slave-unit"]').trigger('click')

    expect(store.activeSlaveUnitId).toBe(2)
    expect(store.slaveUnitGrids.find(unit => unit.unitId === 2)?.holdingRegisters[0].value).toBe(24)
    store.slaveUnitGrids.find(unit => unit.unitId === 2)!.holdingRegisters[0].value = 22

    await wrapper.find('[data-testid="modbus-slave-unit-tab-1"]').trigger('click')
    expect(store.activeSlaveUnitId).toBe(1)
    expect(store.slaveUnitGrids.find(unit => unit.unitId === 1)?.holdingRegisters[0].value).toBe(11)

    await wrapper.find('[data-testid="modbus-slave-unit-tab-2"]').trigger('click')
    expect(store.activeSlaveUnitId).toBe(2)
    expect(store.slaveUnitGrids.find(unit => unit.unitId === 2)?.holdingRegisters[0].value).toBe(22)

    await findButton(wrapper, '删除 Unit').trigger('click')
    await flushPromises()
    expect(store.slaveUnitGrids.map(unit => unit.unitId)).toEqual([1])
    expect(store.activeSlaveUnitId).toBe(1)
  })

  it('uses slave toolbar and row controls to apply backend-backed data grids', async () => {
    const store = useModbusStore()
    store.sessions.set('modbus-slave', { ...sampleSession('modbus-slave', SessionRole.SessionRoleSlave), SlaveRunning: false })
    store.setActiveSession('modbus-slave')
    const wrapper = mount(ModbusPanel, {
      props: { variant: 'tab', sessionId: 'modbus-slave' },
      global: { stubs },
    })

    await findButton(wrapper, '启动从站').trigger('click')
    await flushPromises()
    expect(serialService.startModbusSlave).toHaveBeenCalledWith(expect.objectContaining({
      SessionID: 'modbus-slave',
      Units: [expect.objectContaining({ UnitID: 1 })],
    }))

    store.sessions.set('modbus-slave', { ...sampleSession('modbus-slave', SessionRole.SessionRoleSlave), SlaveRunning: true })
    store.setActiveSession('modbus-slave')
    await findButton(wrapper, '应用数据').trigger('click')
    await flushPromises()
    expect(serialService.updateModbusSlaveUnitData).toHaveBeenCalledWith(
      'modbus-slave',
      1,
      expect.objectContaining({ HoldingRegisters: expect.any(Array) })
    )

    await findButton(wrapper, '停止从站').trigger('click')
    await flushPromises()
    expect(serialService.stopModbusSlave).toHaveBeenCalledWith('modbus-slave')

    await findButton(wrapper, 'ON').trigger('click')
    await wrapper.vm.$nextTick()
    expect(store.slaveUnitGrids.find(unit => unit.unitId === 1)?.coils[0].value).toBe(false)
    await wrapper.find('[data-testid="modbus-slave-table-coils"] .modbus-panel__row-actions button').trigger('click')
    expect(store.slaveUnitGrids.find(unit => unit.unitId === 1)?.coils).toHaveLength(2)
  })

  it('adds and removes rows across the slave data grids', async () => {
    const store = useModbusStore()
    store.sessions.set('modbus-slave', sampleSession('modbus-slave', SessionRole.SessionRoleSlave))
    store.setActiveSession('modbus-slave')
    store.activeSlaveUnitId = 9
    const wrapper = mount(ModbusPanel, {
      props: { variant: 'tab', sessionId: 'modbus-slave' },
      global: { stubs },
    })
    await wrapper.vm.$nextTick()

    const grid = store.slaveUnitGrids.find(unit => unit.unitId === 9)!
    expect(grid).toBeTruthy()
    const addButtons = wrapper.findAll('.modbus-panel__slave-grid .modbus-panel__section-header button')
      .filter(button => button.text() === '添加')
    expect(addButtons).toHaveLength(4)
    await addButtons[0].trigger('click')
    await addButtons[1].trigger('click')
    await addButtons[2].trigger('click')
    await addButtons[3].trigger('click')
    expect(grid.coils).toHaveLength(4)
    expect(grid.discreteInputs).toHaveLength(3)
    expect(grid.inputRegisters).toHaveLength(3)
    expect(grid.holdingRegisters).toHaveLength(3)

    const inputRegisterInputs = wrapper.find('[data-testid="modbus-slave-table-inputRegisters"]').findAll('input')
    await inputRegisterInputs[0].setValue('5')
    await inputRegisterInputs[1].setValue('77')
    await inputRegisterInputs[2].setValue('输入备注')
    expect(grid.inputRegisters[0]).toMatchObject({ address: 5, value: 77, comment: '输入备注' })

    const holdingRegisterInputs = wrapper.find('[data-testid="modbus-slave-table-holdingRegisters"]').findAll('input')
    await holdingRegisterInputs[0].setValue('6')
    await holdingRegisterInputs[1].setValue('88')
    await holdingRegisterInputs[2].setValue('保持备注')
    expect(grid.holdingRegisters[0]).toMatchObject({ address: 6, value: 88, comment: '保持备注' })

    await wrapper.find('[data-testid="modbus-slave-table-discreteInputs"] .modbus-panel__row-actions button').trigger('click')
    await wrapper.findAll('[data-testid="modbus-slave-table-inputRegisters"] .modbus-panel__row-actions button')
      .find(button => button.text() === '删除')!
      .trigger('click')
    await wrapper.findAll('[data-testid="modbus-slave-table-holdingRegisters"] .modbus-panel__row-actions button')
      .find(button => button.text() === '删除')!
      .trigger('click')
    expect(grid.discreteInputs).toHaveLength(2)
    expect(grid.inputRegisters).toHaveLength(2)
    expect(grid.holdingRegisters).toHaveLength(2)
  })

  it('reacts to session prop changes and exposes empty, raw-bit, exception, and value summaries', async () => {
    const store = useModbusStore()
    store.sessions.set('modbus-empty', sampleSession('modbus-empty', SessionRole.SessionRoleMaster))
    store.sessions.set('modbus-master', sampleSession('modbus-master', SessionRole.SessionRoleMaster))
    store.setActiveSession('modbus-empty')
    store.registerReadResult = {
      Transaction: null,
      RawRegisters: [],
      Bits: [true, false],
      Values: [],
    }
    store.history = [
      {
        ...sampleTransaction('exception-tx'),
        Response: { ...sampleTransaction('exception-tx').Response, Exception: true, ExceptionCode: 2, Values: [], Bits: [] },
      },
      {
        ...sampleTransaction('value-tx'),
        Response: { ...sampleTransaction('value-tx').Response, Values: [], Bits: [], Quantity: 0, Value: 99 },
      },
      {
        ...sampleTransaction('empty-tx'),
        Response: { ...sampleTransaction('empty-tx').Response, Values: [], Bits: [], Quantity: 0, Value: 0 },
        ResponseFrameHex: '',
      },
    ] as any

    const wrapper = mount(ModbusPanel, {
      props: { variant: 'tab', sessionId: 'missing-session' },
      global: { stubs },
    })
    expect(wrapper.text()).toContain('会话不存在')
    expect(store.activeSessionId).toBe('missing-session')

    await wrapper.setProps({ sessionId: 'modbus-master' })
    await wrapper.vm.$nextTick()
    expect(store.activeSessionId).toBe('modbus-master')
    expect(wrapper.text()).toContain('1, 0')

    await findButton(wrapper, '日志').trigger('click')

    expect(wrapper.text()).toContain('异常码 2')
    expect(wrapper.text()).toContain('值 99')
    expect(wrapper.text()).toContain('无响应数据')
  })

  it('covers master register focus, toolbar toggles, delete fallback, and dialog saturation', async () => {
    const store = useModbusStore()
    store.restoreState({
      ...store.exportState(),
      activeSessionId: 'modbus-master',
      sessions: [sampleSession('modbus-master', SessionRole.SessionRoleMaster)],
    })
    store.addMasterUnit(2)
    const wrapper = mount(ModbusPanel, {
      props: { variant: 'tab', sessionId: 'modbus-master' },
      global: { stubs },
    })
    const inputCard = wrapper.find('[data-testid="modbus-master-card-input_registers"]')

    await inputCard.trigger('focusin')
    expect(store.masterGrid.registerType).toBe('input_registers')
    await wrapper.find('[data-testid="modbus-master-card-coils"] .modbus-panel__title-button').trigger('click')
    expect(store.masterGrid.registerType).toBe('coils')
    await findButton(wrapper, 'Raw').trigger('click')
    expect(store.masterGrid.rawVisible).toBe(false)
    await findButton(wrapper, '删除').trigger('click')
    expect(store.masterUnitGrids.map(unit => unit.unitId)).toEqual([1])

    store.masterUnitGrids = Array.from({ length: 247 }, (_, index) => ({
      ...store.masterUnitGrids[0],
      unitId: index + 1,
    }))
    await wrapper.vm.$nextTick()
    await wrapper.find('[data-testid="modbus-add-master-unit"]').trigger('click')

    expect(wrapper.find('[data-testid="modbus-master-unit-dialog"]').exists()).toBe(true)
    expect((wrapper.find('[data-testid="modbus-new-master-unit-id"] input').element as HTMLInputElement).value).toBe('247')
  })

  it('drives collapsed master/slave config controls and slave row selectors', async () => {
    const store = useModbusStore()
    store.restoreState({
      ...store.exportState(),
      activeSessionId: 'modbus-master',
      sessions: [sampleSession('modbus-master', SessionRole.SessionRoleMaster)],
    })
    const masterWrapper = mount(ModbusPanel, {
      props: { variant: 'tab', sessionId: 'modbus-master' },
      global: { stubs },
    })
    await masterWrapper.find('[data-testid="modbus-master-config-toggle"]').trigger('click')
    await masterWrapper.find('[data-testid="modbus-master-request-toggle"]').trigger('click')
    ;(masterWrapper.findComponent('[data-testid="modbus-master-request-function"]') as VueWrapper<ComponentPublicInstance>).vm.$emit('update:value', 2)
    await masterWrapper.vm.$nextTick()
    expect(store.masterForm.functionCode).toBe(2)

    store.restoreState({
      ...store.exportState(),
      activeSessionId: 'modbus-slave',
      sessions: [sampleSession('modbus-slave', SessionRole.SessionRoleSlave)],
    })
    const slaveWrapper = mount(ModbusPanel, {
      props: { variant: 'tab', sessionId: 'modbus-slave' },
      global: { stubs },
    })
    await slaveWrapper.vm.$nextTick()
    await slaveWrapper.find('.modbus-panel__collapsible .modbus-panel__collapse-header').trigger('click')
    await slaveWrapper.vm.$nextTick()
    expect(slaveWrapper.text()).toContain('字序')
    await slaveWrapper.find('[data-testid="modbus-slave-table-coils"] input').setValue('9')
    await slaveWrapper.find('[data-testid="modbus-slave-table-discreteInputs"] input').setValue('10')
    await slaveWrapper.vm.$nextTick()

    const grid = store.slaveUnitGrids.find(unit => unit.unitId === 1)!
    expect(grid.coils[0].address).toBe(9)
    expect(grid.discreteInputs[0].address).toBe(10)
  })
})

function sampleSession(id: string, role: SessionRole) {
  return {
    ID: id,
    Name: role === SessionRole.SessionRoleMaster ? 'Modbus 主站' : 'Modbus 从站',
    Mode: FrameMode.FrameModeRTU,
    Role: role,
    Config: {
      PortName: '/tmp/ttyM0',
      BaudRate: 115200,
      DataBits: 8,
      StopBits: '1',
      Parity: 'none',
      FlowMode: 'none',
      ReadBufKB: 32,
    },
    Status: 'open',
    RxBytes: 0,
    TxBytes: 0,
    SlaveRunning: role === SessionRole.SessionRoleSlave,
    UnitID: 1,
    UnitIDs: [1],
    StartedAt: '',
    StoppedAt: '',
    LastError: '',
  }
}

function sampleTransaction(id: string) {
  return {
    ID: id,
    SessionID: 'modbus-master',
    StartedAt: '',
    CompletedAt: '',
    UnitID: 1,
    Mode: FrameMode.FrameModeRTU,
    RequestPDU: { Function: 3, Data: '' },
    ResponsePDU: { Function: 3, Data: '' },
    RequestFrameHex: '01 03 00 00 00 02',
    ResponseFrameHex: '01 03 04 00 05 00 06',
    BytesWritten: 8,
    Response: {
      Function: 3,
      Exception: false,
      ExceptionCode: 0,
      Address: 0,
      Quantity: 2,
      Value: 0,
      Values: [5, 6],
      Bits: [],
      Raw: '',
    },
    Error: '',
  }
}

function findButton(wrapper: ReturnType<typeof mount>, label: string) {
  const button = wrapper.findAll('button').find(item => item.text() === label)
  if (!button) throw new Error(`Button not found: ${label}`)
  return button
}

const stubs = {
  NAlert: { template: '<div><slot /></div>' },
  NButton: { props: ['disabled'], emits: ['click'], template: '<button v-bind="$attrs" :disabled="disabled" @click="$emit(\'click\', $event)"><slot /></button>' },
  NButtonGroup: { template: '<div><slot /></div>' },
  NCheckbox: { template: '<label><input type="checkbox" /><slot /></label>' },
  NForm: { template: '<form><slot /></form>' },
  NFormItem: { props: ['label'], template: '<label><span>{{ label }}</span><slot /></label>' },
  NInput: { name: 'NInput', props: ['value', 'disabled'], emits: ['update:value'], template: '<input v-bind="$attrs" :value="value" :disabled="disabled" @input="$emit(\'update:value\', $event.target.value)" />' },
  NInputNumber: { name: 'NInputNumber', props: ['value', 'disabled'], emits: ['update:value'], template: '<input v-bind="$attrs" type="number" :value="value" :disabled="disabled" @input="$emit(\'update:value\', Number($event.target.value))" />' },
  NSelect: {
    name: 'NSelect',
    props: ['value', 'options', 'disabled'],
    emits: ['update:value'],
    template: '<select v-bind="$attrs" :value="value" :disabled="disabled" @change="$emit(\'update:value\', $event.target.value)"><option v-for="option in options" :key="option.value" :value="option.value">{{ option.label }}</option></select>',
  },
  NSpace: { template: '<div><slot /></div>' },
  NSwitch: { template: '<button><slot /></button>' },
  NTable: { template: '<table><slot /></table>' },
  NTag: { template: '<span><slot /></span>' },
}
