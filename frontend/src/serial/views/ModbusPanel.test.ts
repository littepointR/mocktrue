import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import ModbusPanel from './ModbusPanel.vue'
import { useModbusStore } from '../stores/modbusStore'
import { FrameMode, SessionRole } from '../../../bindings/github.com/suyue/mocktrue/internal/modules/serial/modbus/models.js'

vi.mock('../services/serialService', () => ({
  serialService: {
    enumeratePorts: vi.fn(async () => []),
    listPorts: vi.fn(async () => []),
    listModbusSessions: vi.fn(async () => []),
    openModbusSession: vi.fn(async () => null),
    addModbusSlaveUnit: vi.fn(async () => undefined),
    removeModbusSlaveUnit: vi.fn(async () => undefined),
  },
}))

describe('ModbusPanel', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
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
  })

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

const stubs = {
  NAlert: { template: '<div><slot /></div>' },
  NButton: { template: '<button v-bind="$attrs" @click="$emit(\'click\', $event)"><slot /></button>' },
  NButtonGroup: { template: '<div><slot /></div>' },
  NCheckbox: { template: '<label><input type="checkbox" /><slot /></label>' },
  NForm: { template: '<form><slot /></form>' },
  NFormItem: { props: ['label'], template: '<label><span>{{ label }}</span><slot /></label>' },
  NInput: { props: ['value', 'disabled'], emits: ['update:value'], template: '<input v-bind="$attrs" :value="value" :disabled="disabled" @input="$emit(\'update:value\', $event.target.value)" />' },
  NInputNumber: { props: ['value', 'disabled'], emits: ['update:value'], template: '<input v-bind="$attrs" type="number" :value="value" :disabled="disabled" @input="$emit(\'update:value\', Number($event.target.value))" />' },
  NSelect: {
    props: ['value', 'options', 'disabled'],
    emits: ['update:value'],
    template: '<select v-bind="$attrs" :value="value" :disabled="disabled" @change="$emit(\'update:value\', $event.target.value)"><option v-for="option in options" :key="option.value" :value="option.value">{{ option.label }}</option></select>',
  },
  NSpace: { template: '<div><slot /></div>' },
  NSwitch: { template: '<button><slot /></button>' },
  NTable: { template: '<table><slot /></table>' },
  NTag: { template: '<span><slot /></span>' },
}
