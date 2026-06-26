<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import {
  NAlert,
  NButton,
  NForm,
  NFormItem,
  NInput,
  NInputNumber,
  NSelect,
  NSpace,
  NSwitch,
  NTag,
} from 'naive-ui'
import { useSerialStore } from '../stores/serialStore'
import { defaultModbusSlaveUnitGrid, useModbusStore } from '../stores/modbusStore'
import ResizableTable, { type ResizableTableColumn } from '../components/ResizableTable.vue'
import type {
  ModbusDataType,
  ModbusMasterTableRow,
  ModbusMasterRegisterTableState,
  ModbusRegisterMappingRow,
  ModbusRegisterType,
  ModbusSlaveBoolRow,
  ModbusSlaveRegisterRow,
  ModbusSlaveUnitGridState,
  ModbusWordOrder,
} from '../stores/modbusStore'
import { SessionRole } from '../../../bindings/github.com/littepointR/portweave/internal/modules/serial/modbus/models.js'

const props = withDefaults(defineProps<{
  variant?: 'create' | 'tab'
  sessionId?: string
}>(), {
  variant: 'create',
  sessionId: '',
})

const emit = defineEmits<{
  opened: [id: string]
}>()

const serialStore = useSerialStore()
const modbusStore = useModbusStore()
const loading = ref(false)
const sending = ref(false)
const slaveLoading = ref(false)
const reading = ref(false)
const scanningUnits = ref(false)
const scanningRegisters = ref(false)
const masterConfigOpen = ref(false)
const masterRequestOpen = ref(false)
const slaveConfigOpen = ref(false)
const masterUnitDialogOpen = ref(false)
const slaveUnitDialogOpen = ref(false)
const pendingMasterUnitId = ref(1)
const pendingSlaveUnitId = ref(1)

const baudOptions = [9600, 19200, 38400, 57600, 115200, 230400, 460800, 921600, 1000000, 2000000, 4000000].map(v => ({
  label: String(v),
  value: v,
}))
const dataBitsOptions = [5, 6, 7, 8].map(v => ({ label: String(v), value: v }))
const stopBitsOptions = [
  { label: '1', value: '1' },
  { label: '1.5', value: '1.5' },
  { label: '2', value: '2' },
]
const parityOptions = [
  { label: '无', value: 'none' },
  { label: '偶', value: 'even' },
  { label: '奇', value: 'odd' },
  { label: 'Mark', value: 'mark' },
  { label: 'Space', value: 'space' },
]
const flowOptions = [
  { label: '无', value: 'none' },
  { label: '硬件 (RTS/CTS)', value: 'hw_rtscts' },
  { label: '软件 (XON/XOFF)', value: 'sw_xonxoff' },
]
const modeOptions = [
  { label: 'RTU', value: 'rtu' },
  { label: 'ASCII', value: 'ascii' },
]
const roleOptions = [
  { label: 'Master', value: 'master' },
  { label: 'Slave', value: 'slave' },
]
const addressModeOptions = [
  { label: '0-based', value: 'zero-based' },
  { label: 'PLC 地址', value: 'plc' },
]
const functionOptions = [
  { label: '01 读线圈', value: 1 },
  { label: '02 读离散输入', value: 2 },
  { label: '03 读保持寄存器', value: 3 },
  { label: '04 读输入寄存器', value: 4 },
  { label: '05 写单线圈', value: 5 },
  { label: '06 写单寄存器', value: 6 },
  { label: '0F 写多线圈', value: 15 },
  { label: '10 写多寄存器', value: 16 },
]
const registerFunctionOptions = [
  { label: '03 读保持寄存器', value: 3 },
  { label: '04 读输入寄存器', value: 4 },
]
const coilValueOptions = [
  { label: 'ON', value: 1 },
  { label: 'OFF', value: 0 },
]
const registerTypeOptions = [
  { label: 'Coils', value: 'coils' },
  { label: 'Discrete Inputs', value: 'discrete_inputs' },
  { label: 'Input Registers', value: 'input_registers' },
  { label: 'Holding Registers', value: 'holding_registers' },
]
const addressBaseOptions = [
  { label: '0-based', value: 0 },
  { label: '1-based / PLC', value: 1 },
]
const dataTypeOptions = [
  { label: 'none', value: 'none' },
  { label: 'int16', value: 'int16' },
  { label: 'uint16', value: 'uint16' },
  { label: 'int32', value: 'int32' },
  { label: 'uint32', value: 'uint32' },
  { label: 'int64', value: 'int64' },
  { label: 'uint64', value: 'uint64' },
  { label: 'float', value: 'float' },
  { label: 'double', value: 'double' },
  { label: 'unix', value: 'unix' },
  { label: 'datetime', value: 'datetime' },
  { label: 'utf8', value: 'utf8' },
]
const wordOrderOptions = [
  { label: 'BE', value: 'big' },
  { label: 'LE', value: 'little' },
]
const masterRegisterColumns: ResizableTableColumn[] = [
  { key: 'address', label: '地址', width: 84, minWidth: 70 },
  { key: 'alias', label: '别名', width: 150, minWidth: 100 },
  { key: 'value', label: '值', width: 90, minWidth: 72 },
  { key: 'raw', label: 'Raw', width: 96, minWidth: 72 },
  { key: 'mapping', label: '映射', width: 110, minWidth: 84 },
  { key: 'type', label: '类型', width: 118, minWidth: 96 },
  { key: 'word', label: '字序', width: 82, minWidth: 70 },
  { key: 'length', label: '长度', width: 84, minWidth: 70 },
  { key: 'scale', label: '系数', width: 84, minWidth: 70 },
  { key: 'actions', label: '', width: 150, minWidth: 120 },
]
const mappedResultColumns: ResizableTableColumn[] = [
  { key: 'address', label: '地址', width: 90, minWidth: 70 },
  { key: 'type', label: '类型', width: 110, minWidth: 84 },
  { key: 'value', label: '值', width: 180, minWidth: 120 },
  { key: 'comment', label: '注释', width: 220, minWidth: 120 },
]
const slaveBoolColumns: ResizableTableColumn[] = [
  { key: 'address', label: '地址', width: 120, minWidth: 84 },
  { key: 'value', label: '值', width: 90, minWidth: 70 },
  { key: 'comment', label: '备注', width: 160, minWidth: 90 },
  { key: 'actions', label: '', width: 80, minWidth: 70 },
]
const slaveRegisterColumns: ResizableTableColumn[] = [
  { key: 'address', label: '地址', width: 120, minWidth: 84 },
  { key: 'value', label: '值', width: 120, minWidth: 84 },
  { key: 'type', label: '类型', width: 130, minWidth: 96 },
  { key: 'mapping', label: '映射', width: 110, minWidth: 84 },
  { key: 'comment', label: '注释', width: 170, minWidth: 100 },
  { key: 'actions', label: '', width: 120, minWidth: 96 },
]

const portOptions = computed(() =>
  serialStore.ports.map(p => ({
    label: p.FriendlyName || p.Name,
    value: p.Name,
  }))
)
const selectedFunction = computed(() => modbusStore.masterForm.functionCode)
const isReadFunction = computed(() => [1, 2, 3, 4].includes(selectedFunction.value))
const isSingleWrite = computed(() => [5, 6].includes(selectedFunction.value))
const isMultiCoilWrite = computed(() => selectedFunction.value === 15)
const isMultiRegisterWrite = computed(() => selectedFunction.value === 16)
const targetSessionId = computed(() => props.sessionId || modbusStore.activeSessionId)
const activeSession = computed(() =>
  targetSessionId.value ? modbusStore.sessions.get(targetSessionId.value) ?? null : null
)
const isMasterSession = computed(() => activeSession.value?.Role === SessionRole.SessionRoleMaster)
const isSlaveSession = computed(() => activeSession.value?.Role === SessionRole.SessionRoleSlave)
const lastTransaction = computed(() => modbusStore.history[0] ?? null)
const registerRawValues = computed(() => modbusStore.registerReadResult?.RawRegisters ?? [])
const registerMappedValues = computed(() => modbusStore.registerReadResult?.Values ?? [])
const registerBits = computed(() => modbusStore.registerReadResult?.Bits ?? [])
const activeUnitIds = computed(() => modbusStore.unitScanResult?.ActiveUnitIDs ?? [])
const registerScanValues = computed(() => modbusStore.registerScanResult?.Values ?? [])
const activeSlaveGrid = computed(() => currentSlaveGrid())
const activeMasterTable = computed(() => currentMasterTable())

onMounted(() => {
  selectTargetSession()
  serialStore.refreshPorts()
  modbusStore.refreshSessions()
})

watch(
  () => props.sessionId,
  () => selectTargetSession()
)

function selectTargetSession() {
  if (props.sessionId) {
    modbusStore.setActiveSession(props.sessionId)
  }
}

async function openSession() {
  loading.value = true
  try {
    const id = await modbusStore.openSession()
    if (id) emit('opened', id)
  } finally {
    loading.value = false
  }
}

async function refresh() {
  loading.value = true
  try {
    await Promise.all([serialStore.refreshPorts(), modbusStore.refreshSessions()])
  } finally {
    loading.value = false
  }
}

async function sendRequest() {
  selectTargetSession()
  sending.value = true
  try {
    await modbusStore.sendMasterRequest()
  } finally {
    sending.value = false
  }
}

async function startSlave() {
  selectTargetSession()
  slaveLoading.value = true
  try {
    await modbusStore.startSlave()
  } finally {
    slaveLoading.value = false
  }
}

async function stopSlave() {
  selectTargetSession()
  slaveLoading.value = true
  try {
    await modbusStore.stopSlave()
  } finally {
    slaveLoading.value = false
  }
}

async function applySlaveData() {
  selectTargetSession()
  slaveLoading.value = true
  try {
    await modbusStore.applySlaveData()
  } finally {
    slaveLoading.value = false
  }
}

function openAddMasterUnitDialog() {
  selectTargetSession()
  pendingMasterUnitId.value = nextAvailableUnitId(modbusStore.masterUnitGrids.map(unit => unit.unitId))
  masterUnitDialogOpen.value = true
}

function confirmAddMasterUnit() {
  if (modbusStore.addMasterUnit(Number(pendingMasterUnitId.value))) {
    masterUnitDialogOpen.value = false
  }
}

function openAddSlaveUnitDialog() {
  selectTargetSession()
  pendingSlaveUnitId.value = nextAvailableUnitId(modbusStore.slaveUnitGrids.map(unit => unit.unitId))
  slaveUnitDialogOpen.value = true
}

async function confirmAddSlaveUnit() {
  selectTargetSession()
  slaveLoading.value = true
  try {
    if (await modbusStore.addSlaveUnit(Number(pendingSlaveUnitId.value))) {
      slaveUnitDialogOpen.value = false
    }
  } finally {
    slaveLoading.value = false
  }
}

async function removeSlaveUnit() {
  selectTargetSession()
  slaveLoading.value = true
  try {
    await modbusStore.removeSlaveUnit()
  } finally {
    slaveLoading.value = false
  }
}

async function readRegisters() {
  selectTargetSession()
  reading.value = true
  try {
    await modbusStore.readRegisters()
  } finally {
    reading.value = false
  }
}

async function scanUnitIDs() {
  selectTargetSession()
  scanningUnits.value = true
  try {
    await modbusStore.scanUnitIDs()
  } finally {
    scanningUnits.value = false
  }
}

async function scanRegisters() {
  selectTargetSession()
  scanningRegisters.value = true
  try {
    await modbusStore.scanRegisters()
  } finally {
    scanningRegisters.value = false
  }
}

function togglePolling() {
  selectTargetSession()
  if (modbusStore.registerReadForm.polling) {
    modbusStore.stopRegisterPolling()
  } else {
    modbusStore.startRegisterPolling()
  }
}

function currentMasterTable(): ModbusMasterRegisterTableState {
  const type = modbusStore.masterGrid.registerType
  const existing = modbusStore.masterRegisterTables.find(table => table.type === type)
  if (existing) return existing
  const fallback = modbusStore.masterRegisterTables[0]
  return fallback
}

function selectMasterTable(table: ModbusMasterRegisterTableState) {
  modbusStore.masterGrid.registerType = table.type
  modbusStore.masterGrid.unitId = modbusStore.activeMasterUnitId
  modbusStore.masterGrid.address = table.address
  modbusStore.masterGrid.length = table.length
  modbusStore.masterGrid.addressBase = table.addressBase
  modbusStore.masterGrid.pollRateMs = table.pollRateMs
  modbusStore.masterGrid.timeoutMs = table.timeoutMs
  modbusStore.masterGrid.retries = table.retries
  modbusStore.masterMappings = table.mappings.map(row => ({ ...row }))
}

function registerTypeLabel(type: ModbusRegisterType): string {
  return registerTypeOptions.find(option => option.value === type)?.label ?? type
}

function nextAvailableUnitId(unitIds: number[]): number {
  const used = new Set(unitIds)
  for (let id = 1; id <= 247; id += 1) {
    if (!used.has(id)) return id
  }
  return 247
}

function addMasterRow(table: ModbusMasterRegisterTableState) {
  const nextAddress = Math.max(-1, ...table.rows.map(row => row.address)) + 1
  table.rows.push({
    id: `${table.type}-${Date.now().toString(36)}`,
    address: nextAddress,
    value: '0',
    raw: '',
    display: '',
    mappingId: '',
  })
  table.length = Math.max(table.length, table.rows.length)
  selectMasterTable(table)
}

function removeMasterRow(table: ModbusMasterRegisterTableState, row: ModbusMasterTableRow) {
  const mapping = mappingForAddress(table, row.address)
  if (isMappingContinuation(mapping, row.address)) return
  if (isMappingStart(mapping, row.address)) {
    table.mappings = table.mappings.filter(item => item.id !== mapping?.id)
  }
  table.rows = table.rows.filter(item => item.id !== row.id)
  table.length = Math.max(1, table.rows.length)
  selectMasterTable(table)
}

function addMappingRow(table = activeMasterTable.value, address?: number) {
  const nextAddress = address ?? Math.max(-1, ...table.mappings.map(row => row.address)) + 1
  if (mappingForAddress(table, nextAddress)) return
  table.mappings = [
    ...table.mappings,
    {
      id: `map-${Date.now().toString(36)}`,
      address: nextAddress,
      dataType: 'uint16',
      wordOrder: modbusStore.masterGrid.littleEndian ? 'little' : 'big',
      length: 0,
      scalingFactor: 0,
      comment: '',
      groupEnd: false,
    },
  ]
  selectMasterTable(table)
}

function removeMappingRow(table: ModbusMasterRegisterTableState, row: ModbusRegisterMappingRow) {
  table.mappings = table.mappings.filter(item => item.id !== row.id)
  selectMasterTable(table)
}

function updateMappingDataType(row: ModbusRegisterMappingRow, value: string) {
  row.dataType = value as ModbusDataType
}

function updateMappingWordOrder(row: ModbusRegisterMappingRow, value: string) {
  row.wordOrder = value as ModbusWordOrder
}

function updateMappingComment(row: ModbusRegisterMappingRow | null, value: string) {
  if (!row) return
  row.comment = value
}

function updateMappingLength(row: ModbusRegisterMappingRow | null, value: number | null) {
  if (!row) return
  row.length = Math.max(0, Math.trunc(Number(value) || 0))
}

function updateMappingScalingFactor(row: ModbusRegisterMappingRow | null, value: number | null) {
  if (!row) return
  row.scalingFactor = Number(value) || 0
}

function updateMasterRowAddress(table: ModbusMasterRegisterTableState, row: ModbusMasterTableRow, value: number | null) {
  const previousAddress = row.address
  const nextAddress = Math.max(0, Math.trunc(Number(value) || 0))
  const mapping = mappingForAddress(table, previousAddress)
  row.address = nextAddress
  if (isMappingStart(mapping, previousAddress)) {
    mapping.address = nextAddress
  }
}

function toggleMasterEndian(littleEndian: boolean) {
  modbusStore.masterGrid.littleEndian = littleEndian
  activeMasterTable.value.mappings = activeMasterTable.value.mappings.map(row => ({
    ...row,
    wordOrder: littleEndian ? 'little' : 'big',
  }))
  selectMasterTable(activeMasterTable.value)
}

function mappingForAddress(table: ModbusMasterRegisterTableState, address: number): ModbusRegisterMappingRow | null {
  return table.mappings.find(mapping => address >= mapping.address && address <= mappingEndAddress(mapping)) ?? null
}

function mappingEndAddress(mapping: ModbusRegisterMappingRow): number {
  return mapping.address + mappingLength(mapping) - 1
}

function mappingLength(mapping: ModbusRegisterMappingRow): number {
  if (mapping.length > 0) return mapping.length
  switch (mapping.dataType) {
    case 'int32':
    case 'uint32':
    case 'float':
    case 'unix':
      return 2
    case 'int64':
    case 'uint64':
    case 'double':
    case 'datetime':
      return 4
    case 'utf8':
      return 24
    case 'none':
      return 1
    default:
      return 1
  }
}

function mappingColorIndex(mapping: ModbusRegisterMappingRow | null): number {
  if (!mapping) return 0
  const seed = Array.from(mapping.id || String(mapping.address)).reduce((sum, char) => sum + char.charCodeAt(0), 0)
  return (seed % 6) + 1
}

function mappingClass(mapping: ModbusRegisterMappingRow | null): string {
  return mapping ? `modbus-panel__mapping-band modbus-panel__mapping-band--${mappingColorIndex(mapping)}` : ''
}

function mappingRole(mapping: ModbusRegisterMappingRow | null, address: number): string {
  if (!mapping) return ''
  if (address === mapping.address && mappingLength(mapping) > 1) return '起'
  if (address === mappingEndAddress(mapping) && mappingLength(mapping) > 1) return '止'
  if (address > mapping.address && address < mappingEndAddress(mapping)) return '续'
  return '映射'
}

function isMappingStart(mapping: ModbusRegisterMappingRow | null, address: number): mapping is ModbusRegisterMappingRow {
  return Boolean(mapping && mapping.address === address)
}

function isMappingContinuation(mapping: ModbusRegisterMappingRow | null, address: number): boolean {
  return Boolean(mapping && mapping.address !== address)
}

function rowDisplayValue(row: ModbusMasterTableRow): string {
  return row.display || row.value || '0'
}

function currentSlaveGrid(): ModbusSlaveUnitGridState {
  const unitId = Number(modbusStore.activeSlaveUnitId || 1)
  const existing = modbusStore.slaveUnitGrids.find(unit => unit.unitId === unitId)
  if (existing) return existing
  const next = defaultModbusSlaveUnitGrid(unitId)
  modbusStore.slaveUnitGrids = [...modbusStore.slaveUnitGrids, next].sort((a, b) => a.unitId - b.unitId)
  return next
}

function addBoolRow(key: 'coils' | 'discreteInputs') {
  const grid = currentSlaveGrid()
  const rows = grid[key]
  const nextAddress = Math.max(-1, ...rows.map(row => row.address)) + 1
  rows.push({ id: `${key}-${Date.now().toString(36)}`, address: nextAddress, value: false })
}

function removeBoolRow(key: 'coils' | 'discreteInputs', row: ModbusSlaveBoolRow) {
  const grid = currentSlaveGrid()
  grid[key] = grid[key].filter(item => item.id !== row.id)
}

function addRegisterRow(key: 'inputRegisters' | 'holdingRegisters') {
  const grid = currentSlaveGrid()
  const rows = grid[key]
  const nextAddress = Math.max(-1, ...rows.map(row => row.address)) + 1
  rows.push({
    id: `${key}-${Date.now().toString(36)}`,
    address: nextAddress,
    value: 0,
    dataType: 'uint16',
    comment: '',
  })
}

function removeRegisterRow(key: 'inputRegisters' | 'holdingRegisters', row: ModbusSlaveRegisterRow) {
  const grid = currentSlaveGrid()
  grid[key] = grid[key].filter(item => item.id !== row.id)
}

function updateSlaveRegisterDataType(row: ModbusSlaveRegisterRow, value: string) {
  row.dataType = value as ModbusDataType
}

function mappingForSlaveRegister(row: ModbusSlaveRegisterRow): ModbusRegisterMappingRow {
  return {
    id: row.id,
    address: row.address,
    dataType: row.dataType,
    wordOrder: modbusStore.masterGrid.littleEndian ? 'little' : 'big',
    length: 0,
    scalingFactor: 0,
    comment: row.comment,
    groupEnd: false,
  }
}

function responseSummary(tx: any): string {
  if (!tx) return ''
  if (tx.Error) return tx.Error
  if (tx.Response?.Exception) return `异常码 ${tx.Response.ExceptionCode}`
  if (tx.Response?.Values?.length) return tx.Response.Values.join(', ')
  if (tx.Response?.Bits?.length) return tx.Response.Bits.map((value: boolean) => value ? '1' : '0').join(' ')
  if (tx.Response?.Quantity) return `地址 ${tx.Response.Address}, 数量 ${tx.Response.Quantity}`
  if (tx.Response?.Value) return `地址 ${tx.Response.Address}, 值 ${tx.Response.Value}`
  return tx.ResponseFrameHex || '无响应数据'
}
</script>

<template>
  <div class="modbus-panel" :class="{ 'modbus-panel--tab': variant === 'tab' }">
    <div class="modbus-panel__header">
      <strong>{{ variant === 'create' ? 'Modbus 调试' : (activeSession?.Name || activeSession?.ID || 'Modbus') }}</strong>
      <NButton size="tiny" secondary :loading="loading" @click="refresh">刷新</NButton>
    </div>

    <NAlert v-if="modbusStore.error" type="error" closable @close="modbusStore.clearError()">
      {{ modbusStore.error }}
    </NAlert>

    <section v-if="variant === 'create'" class="modbus-panel__section">
      <div class="modbus-panel__section-title">会话</div>
      <NForm label-placement="top" size="small">
        <NFormItem label="端口">
          <NSelect
            v-model:value="modbusStore.portForm.port"
            :options="portOptions"
            placeholder="选择串口"
            filterable
            :disabled="loading"
          />
        </NFormItem>
        <div class="modbus-panel__grid">
          <NFormItem label="角色">
            <NSelect v-model:value="modbusStore.portForm.role" :options="roleOptions" :disabled="loading" />
          </NFormItem>
          <NFormItem label="模式">
            <NSelect v-model:value="modbusStore.portForm.mode" :options="modeOptions" :disabled="loading" />
          </NFormItem>
          <NFormItem label="波特率">
            <NSelect v-model:value="modbusStore.portForm.baudRate" :options="baudOptions" :disabled="loading" />
          </NFormItem>
          <NFormItem label="数据位">
            <NSelect v-model:value="modbusStore.portForm.dataBits" :options="dataBitsOptions" :disabled="loading" />
          </NFormItem>
          <NFormItem label="停止位">
            <NSelect v-model:value="modbusStore.portForm.stopBits" :options="stopBitsOptions" :disabled="loading" />
          </NFormItem>
          <NFormItem label="校验">
            <NSelect v-model:value="modbusStore.portForm.parity" :options="parityOptions" :disabled="loading" />
          </NFormItem>
          <NFormItem label="流控">
            <NSelect v-model:value="modbusStore.portForm.flowMode" :options="flowOptions" :disabled="loading" />
          </NFormItem>
          <NFormItem label="超时 ms">
            <NInputNumber v-model:value="modbusStore.portForm.timeoutMs" :min="10" :step="50" :disabled="loading" />
          </NFormItem>
          <NFormItem label="重试">
            <NInputNumber v-model:value="modbusStore.portForm.retries" :min="0" :step="1" :disabled="loading" />
          </NFormItem>
        </div>
        <NFormItem label="名称">
          <NInput v-model:value="modbusStore.portForm.name" placeholder="可选" :disabled="loading" />
        </NFormItem>
        <NSpace justify="end">
          <NButton
            type="primary"
            size="small"
            :loading="loading"
            :disabled="!modbusStore.portForm.port"
            @click="openSession"
          >
            打开
          </NButton>
        </NSpace>
      </NForm>
      <div v-if="activeSession" class="modbus-panel__status">
        <NTag size="small" :type="activeSession.SlaveRunning ? 'success' : 'info'">
          {{ activeSession.Status }}
        </NTag>
        <span>RX {{ activeSession.RxBytes }} / TX {{ activeSession.TxBytes }}</span>
      </div>
    </section>

    <div v-else-if="!activeSession" class="modbus-panel__empty">会话不存在</div>

    <section v-if="variant === 'tab' && isMasterSession" class="modbus-panel__workbench">
      <div class="modbus-panel__toolbar">
        <div class="modbus-panel__toolbar-group">
          <NButton
            type="primary"
            size="small"
            :loading="reading"
            :disabled="!activeSession || activeSession.SlaveRunning"
            @click="readRegisters"
          >
            读
          </NButton>
          <NButton size="small" :disabled="!activeSession || activeSession.SlaveRunning" @click="togglePolling">
            {{ modbusStore.registerReadForm.polling ? '停止轮询' : '轮询' }}
          </NButton>
          <NButton size="small" :loading="scanningUnits" :disabled="!activeSession || activeSession.SlaveRunning" @click="scanUnitIDs">
            扫描 Unit
          </NButton>
          <NButton size="small" :loading="scanningRegisters" :disabled="!activeSession || activeSession.SlaveRunning" @click="scanRegisters">
            扫描寄存器
          </NButton>
        </div>
        <div class="modbus-panel__toolbar-group">
          <NButton size="small" secondary :type="modbusStore.masterGrid.rawVisible ? 'primary' : 'default'" @click="modbusStore.masterGrid.rawVisible = !modbusStore.masterGrid.rawVisible">
            Raw
          </NButton>
          <NButton size="small" secondary :type="modbusStore.masterGrid.logVisible ? 'primary' : 'default'" @click="modbusStore.masterGrid.logVisible = !modbusStore.masterGrid.logVisible">
            日志
          </NButton>
        </div>
      </div>

      <div class="modbus-panel__unit-tabs" data-testid="modbus-master-unit-tabs">
        <span class="modbus-panel__unit-tabs-label">主站 Unit</span>
        <div class="modbus-panel__unit-tab-list">
          <button
            v-for="unit in modbusStore.masterUnitGrids"
            :key="unit.unitId"
            class="modbus-panel__unit-tab"
            :class="{ 'modbus-panel__unit-tab--active': unit.unitId === modbusStore.activeMasterUnitId }"
            :data-testid="`modbus-master-unit-tab-${unit.unitId}`"
            type="button"
            @click="modbusStore.selectMasterUnit(unit.unitId)"
          >
            Unit {{ unit.unitId }}
          </button>
        </div>
        <div class="modbus-panel__unit-tab-actions">
          <NButton size="small" data-testid="modbus-add-master-unit" @click="openAddMasterUnitDialog">添加</NButton>
          <NButton
            size="small"
            :disabled="modbusStore.masterUnitGrids.length <= 1"
            @click="modbusStore.removeMasterUnit()"
          >
            删除
          </NButton>
        </div>
      </div>

      <section class="modbus-panel__section modbus-panel__collapsible">
        <button
          class="modbus-panel__collapse-header"
          data-testid="modbus-master-config-toggle"
          type="button"
          @click="masterConfigOpen = !masterConfigOpen"
        >
          <span class="modbus-panel__collapse-label">
            <span class="modbus-panel__section-title">配置</span>
            <span class="modbus-panel__collapse-state">{{ masterConfigOpen ? '收起' : '展开' }}</span>
          </span>
          <span class="modbus-panel__collapse-summary" data-testid="modbus-master-config-summary">
            Unit {{ activeMasterTable.unitId }} · {{ registerTypeLabel(activeMasterTable.type) }} · {{ activeMasterTable.addressBase === 1 ? 'PLC' : '0-based' }}
          </span>
        </button>
        <div v-if="masterConfigOpen" class="modbus-panel__config-strip">
          <NFormItem label="当前区域">
            <NSelect v-model:value="modbusStore.masterGrid.registerType" :options="registerTypeOptions" />
          </NFormItem>
          <NFormItem label="地址">
            <NInputNumber v-model:value="activeMasterTable.address" :min="0" />
          </NFormItem>
          <NFormItem label="长度">
            <NInputNumber v-model:value="activeMasterTable.length" :min="1" :max="125" />
          </NFormItem>
          <NFormItem label="地址基准">
            <NSelect v-model:value="activeMasterTable.addressBase" :options="addressBaseOptions" />
          </NFormItem>
          <NFormItem label="轮询 ms">
            <NInputNumber v-model:value="activeMasterTable.pollRateMs" :min="10" :step="10" />
          </NFormItem>
          <NFormItem label="超时 ms">
            <NInputNumber v-model:value="activeMasterTable.timeoutMs" :min="10" :step="50" />
          </NFormItem>
          <NFormItem label="重试">
            <NInputNumber v-model:value="activeMasterTable.retries" :min="0" :step="1" />
          </NFormItem>
        </div>
      </section>

      <section class="modbus-panel__section modbus-panel__collapsible">
        <button
          class="modbus-panel__collapse-header"
          data-testid="modbus-master-request-toggle"
          type="button"
          @click="masterRequestOpen = !masterRequestOpen"
        >
          <span class="modbus-panel__collapse-label">
            <span class="modbus-panel__section-title">请求</span>
            <span class="modbus-panel__collapse-state">{{ masterRequestOpen ? '收起' : '展开' }}</span>
          </span>
          <span class="modbus-panel__collapse-summary">
            FC {{ modbusStore.masterForm.functionCode }} · Unit {{ modbusStore.masterForm.unitId }}
          </span>
        </button>
        <div
          v-if="masterRequestOpen"
          class="modbus-panel__config-strip modbus-panel__request-form"
          data-testid="modbus-master-request-form"
        >
          <NFormItem label="Unit ID">
            <NInputNumber v-model:value="modbusStore.masterForm.unitId" :min="1" :max="247" data-testid="modbus-master-request-unit" />
          </NFormItem>
          <NFormItem label="功能码">
            <NSelect
              v-model:value="modbusStore.masterForm.functionCode"
              :options="functionOptions"
              data-testid="modbus-master-request-function"
              data-functions="1,2,3,4,5,6,15,16"
            />
          </NFormItem>
          <NFormItem label="地址模式">
            <NSelect v-model:value="modbusStore.masterForm.addressMode" :options="addressModeOptions" data-testid="modbus-master-request-address-mode" />
          </NFormItem>
          <NFormItem label="地址">
            <NInputNumber v-model:value="modbusStore.masterForm.address" :min="0" data-testid="modbus-master-request-address" />
          </NFormItem>
          <NFormItem v-if="isReadFunction || isMultiCoilWrite || isMultiRegisterWrite" label="数量">
            <NInputNumber v-model:value="modbusStore.masterForm.quantity" :min="1" :max="isMultiCoilWrite ? 1968 : 125" data-testid="modbus-master-request-quantity" />
          </NFormItem>
          <NFormItem v-if="isSingleWrite" label="值">
            <NInputNumber v-model:value="modbusStore.masterForm.value" :min="0" :max="65535" data-testid="modbus-master-request-value" />
          </NFormItem>
          <NFormItem v-if="isMultiCoilWrite" label="线圈值">
            <NInput v-model:value="modbusStore.masterForm.coilValues" placeholder="1 0 true off" data-testid="modbus-master-request-coil-values" />
          </NFormItem>
          <NFormItem v-if="isMultiRegisterWrite" label="寄存器值">
            <NInput v-model:value="modbusStore.masterForm.registerValues" placeholder="24 42 0x2a" data-testid="modbus-master-request-register-values" />
          </NFormItem>
          <NFormItem label="超时 ms">
            <NInputNumber v-model:value="modbusStore.masterForm.timeoutMs" :min="10" :step="50" data-testid="modbus-master-request-timeout" />
          </NFormItem>
          <NFormItem label="重试">
            <NInputNumber v-model:value="modbusStore.masterForm.retries" :min="0" :step="1" data-testid="modbus-master-request-retries" />
          </NFormItem>
          <div class="modbus-panel__request-actions">
            <NButton
              type="primary"
              size="small"
              :loading="sending"
              :disabled="!activeSession || activeSession.SlaveRunning"
              data-testid="modbus-master-request-send"
              @click="sendRequest"
            >
              发送请求
            </NButton>
          </div>
          <div v-if="lastTransaction" class="modbus-panel__frame-preview modbus-panel__request-result" data-testid="modbus-master-request-result">
            <div><span>TX</span><code>{{ lastTransaction.RequestFrameHex }}</code></div>
            <div><span>RX</span><code>{{ lastTransaction.ResponseFrameHex || lastTransaction.Error }}</code></div>
          </div>
        </div>
      </section>

      <div class="modbus-panel__register-grid">
        <section
          v-for="table in modbusStore.masterRegisterTables"
          :key="table.type"
          class="modbus-panel__section modbus-panel__register-card"
          :class="{ 'modbus-panel__register-card--active': table.type === modbusStore.masterGrid.registerType }"
          :data-testid="`modbus-master-card-${table.type}`"
          tabindex="0"
          @click.self="selectMasterTable(table)"
          @focusin="selectMasterTable(table)"
        >
          <div class="modbus-panel__section-header">
            <button class="modbus-panel__section-title modbus-panel__title-button" type="button" @click="selectMasterTable(table)">
              {{ registerTypeLabel(table.type) }}
            </button>
            <div class="modbus-panel__toolbar-group">
              <NButton size="tiny" secondary :type="!modbusStore.masterGrid.littleEndian ? 'primary' : 'default'" @click="toggleMasterEndian(false)">BE</NButton>
              <NButton size="tiny" secondary :type="modbusStore.masterGrid.littleEndian ? 'primary' : 'default'" @click="toggleMasterEndian(true)">LE</NButton>
              <NButton size="tiny" @click="addMasterRow(table)">添加</NButton>
            </div>
          </div>
          <ResizableTable
            :columns="masterRegisterColumns"
            :table-class="[
              'modbus-panel__dense-table',
              { 'modbus-panel__bool-table': table.type === 'coils' || table.type === 'discrete_inputs' },
            ]"
            :data-testid="`modbus-master-table-${table.type}`"
            :min-width="1040"
          >
            <tr
              v-for="row in table.rows"
              :key="row.id"
              :class="[
                mappingClass(mappingForAddress(table, row.address)),
                { 'modbus-panel__register-row--readonly': isMappingContinuation(mappingForAddress(table, row.address), row.address) },
              ]"
              :data-readonly="isMappingContinuation(mappingForAddress(table, row.address), row.address) ? 'true' : 'false'"
              :data-testid="`modbus-master-row-${table.type}-${row.address}`"
              @click="selectMasterTable(table)"
            >
              <td class="modbus-panel__cell--address">
                <NInputNumber
                  :value="row.address"
                  :min="0"
                  :disabled="isMappingContinuation(mappingForAddress(table, row.address), row.address)"
                  @update:value="value => updateMasterRowAddress(table, row, Number(value))"
                />
              </td>
              <td class="modbus-panel__cell--alias">
                <NInput
                  v-if="mappingForAddress(table, row.address)"
                  :value="mappingForAddress(table, row.address)?.comment"
                  :disabled="isMappingContinuation(mappingForAddress(table, row.address), row.address)"
                  :data-testid="`modbus-mapping-alias-${table.type}-${row.address}`"
                  placeholder=""
                  @update:value="value => updateMappingComment(mappingForAddress(table, row.address), String(value))"
                />
              </td>
              <td class="modbus-panel__cell--value">{{ rowDisplayValue(row) }}</td>
              <td><code>{{ row.raw || '-' }}</code></td>
              <td>
                <span v-if="mappingForAddress(table, row.address)" class="modbus-panel__mapping-pill">
                  {{ mappingRole(mappingForAddress(table, row.address), row.address) }}
                  {{ mappingForAddress(table, row.address)?.dataType }}
                </span>
              </td>
              <td class="modbus-panel__cell--type">
                <NSelect
                  v-if="mappingForAddress(table, row.address)"
                  :value="mappingForAddress(table, row.address)?.dataType"
                  :options="dataTypeOptions"
                  :disabled="isMappingContinuation(mappingForAddress(table, row.address), row.address)"
                  :data-testid="`modbus-mapping-type-${table.type}-${row.address}`"
                  @update:value="value => updateMappingDataType(mappingForAddress(table, row.address)!, String(value))"
                />
              </td>
              <td class="modbus-panel__cell--word">
                <NSelect
                  v-if="mappingForAddress(table, row.address)"
                  :value="mappingForAddress(table, row.address)?.wordOrder"
                  :options="wordOrderOptions"
                  :disabled="isMappingContinuation(mappingForAddress(table, row.address), row.address)"
                  :data-testid="`modbus-mapping-word-${table.type}-${row.address}`"
                  @update:value="value => updateMappingWordOrder(mappingForAddress(table, row.address)!, String(value))"
                />
              </td>
              <td class="modbus-panel__cell--number">
                <NInputNumber
                  v-if="mappingForAddress(table, row.address)"
                  :value="mappingForAddress(table, row.address)?.length"
                  :min="0"
                  :disabled="isMappingContinuation(mappingForAddress(table, row.address), row.address)"
                  :data-testid="`modbus-mapping-length-${table.type}-${row.address}`"
                  @update:value="value => updateMappingLength(mappingForAddress(table, row.address), Number(value))"
                />
              </td>
              <td class="modbus-panel__cell--number">
                <NInputNumber
                  v-if="mappingForAddress(table, row.address)"
                  :value="mappingForAddress(table, row.address)?.scalingFactor"
                  :step="0.1"
                  :disabled="isMappingContinuation(mappingForAddress(table, row.address), row.address)"
                  :data-testid="`modbus-mapping-scale-${table.type}-${row.address}`"
                  @update:value="value => updateMappingScalingFactor(mappingForAddress(table, row.address), Number(value))"
                />
              </td>
              <td>
                <div class="modbus-panel__row-actions">
                  <NButton
                    v-if="!mappingForAddress(table, row.address)"
                    size="tiny"
                    quaternary
                    :data-testid="`modbus-add-mapping-${table.type}-${row.address}`"
                    @click="addMappingRow(table, row.address)"
                  >
                    映射
                  </NButton>
                  <NButton
                    v-else-if="isMappingStart(mappingForAddress(table, row.address), row.address)"
                    size="tiny"
                    quaternary
                    :data-testid="`modbus-remove-mapping-${table.type}-${row.address}`"
                    @click="removeMappingRow(table, mappingForAddress(table, row.address)!)"
                  >
                    删除映射
                  </NButton>
                  <NButton
                    size="tiny"
                    quaternary
                    :disabled="isMappingContinuation(mappingForAddress(table, row.address), row.address)"
                    :data-testid="`modbus-remove-row-${table.type}-${row.address}`"
                    @click="removeMasterRow(table, row)"
                  >
                    删除
                  </NButton>
                </div>
              </td>
            </tr>
          </ResizableTable>
        </section>
      </div>

      <section class="modbus-panel__section" v-if="registerMappedValues.length || registerRawValues.length || registerBits.length">
        <div class="modbus-panel__section-title">最后读取结果</div>
        <div v-if="modbusStore.masterGrid.rawVisible" class="modbus-panel__raw-line">
          <span>Raw</span>
          <code v-if="registerRawValues.length">{{ registerRawValues.join(', ') }}</code>
          <code v-else-if="registerBits.length">{{ registerBits.map(value => value ? 1 : 0).join(', ') }}</code>
          <code v-else>暂无</code>
        </div>
        <ResizableTable :columns="mappedResultColumns" table-class="modbus-panel__dense-table" :min-width="600">
          <tr v-for="row in registerMappedValues" :key="`${row.Address}-${row.Mapping.DataType}`">
            <td>{{ row.Address }}</td>
            <td>{{ row.Mapping.DataType }}</td>
            <td>{{ row.Error || row.Value.Display }}</td>
            <td>{{ row.Mapping.Comment }}</td>
          </tr>
        </ResizableTable>
      </section>

      <section class="modbus-panel__section">
        <div class="modbus-panel__section-title">扫描</div>
        <div class="modbus-panel__scan-grid">
          <NFormItem label="Unit IDs">
            <NInput v-model:value="modbusStore.unitScanForm.unitIds" placeholder="1-10, 20" />
          </NFormItem>
          <NFormItem label="扫描起始">
            <NInputNumber v-model:value="modbusStore.registerScanForm.startAddress" :min="0" />
          </NFormItem>
          <NFormItem label="扫描结束">
            <NInputNumber v-model:value="modbusStore.registerScanForm.endAddress" :min="0" />
          </NFormItem>
          <NFormItem label="块大小">
            <NInputNumber v-model:value="modbusStore.registerScanForm.chunkSize" :min="1" :max="125" />
          </NFormItem>
        </div>
        <div class="modbus-panel__scan-line">
          <span>在线 Unit</span>
          <code>{{ activeUnitIds.join(', ') || '无' }}</code>
        </div>
        <div class="modbus-panel__scan-line">
          <span>寄存器</span>
          <code>{{ registerScanValues.map(row => `${row.Address}=${row.Value}`).join(', ') || '无' }}</code>
        </div>
      </section>

      <section v-if="modbusStore.masterGrid.logVisible" class="modbus-panel__section">
        <div class="modbus-panel__section-title">日志</div>
        <div v-if="modbusStore.history.length === 0" class="modbus-panel__empty">暂无事务</div>
        <button v-for="tx in modbusStore.history" :key="tx.ID" class="modbus-panel__history" type="button">
          <span>{{ tx.RequestFrameHex }}</span>
          <code>{{ responseSummary(tx) }}</code>
        </button>
        <div v-if="lastTransaction" class="modbus-panel__frame-preview">
          <div><span>TX</span><code>{{ lastTransaction.RequestFrameHex }}</code></div>
          <div><span>RX</span><code>{{ lastTransaction.ResponseFrameHex || lastTransaction.Error }}</code></div>
        </div>
      </section>
    </section>

    <section v-if="variant === 'tab' && isSlaveSession" class="modbus-panel__workbench">
      <div class="modbus-panel__toolbar">
        <div class="modbus-panel__toolbar-group">
          <NButton size="small" :loading="slaveLoading" data-testid="modbus-add-slave-unit" @click="openAddSlaveUnitDialog">添加 Unit</NButton>
          <NButton size="small" :disabled="modbusStore.slaveUnitGrids.length <= 1" :loading="slaveLoading" @click="removeSlaveUnit">删除 Unit</NButton>
        </div>
        <div class="modbus-panel__toolbar-group">
          <NButton size="small" secondary :type="!modbusStore.masterGrid.littleEndian ? 'primary' : 'default'" @click="toggleMasterEndian(false)">BE</NButton>
          <NButton size="small" secondary :type="modbusStore.masterGrid.littleEndian ? 'primary' : 'default'" @click="toggleMasterEndian(true)">LE</NButton>
          <NButton size="small" :disabled="!activeSession" :loading="slaveLoading" @click="applySlaveData">应用数据</NButton>
          <NButton v-if="activeSession?.SlaveRunning" size="small" :loading="slaveLoading" @click="stopSlave">停止从站</NButton>
          <NButton v-else type="primary" size="small" :disabled="!activeSession" :loading="slaveLoading" @click="startSlave">启动从站</NButton>
        </div>
      </div>

      <div class="modbus-panel__unit-tabs" data-testid="modbus-slave-unit-tabs">
        <span class="modbus-panel__unit-tabs-label">从站 Unit</span>
        <div class="modbus-panel__unit-tab-list">
          <button
            v-for="unit in modbusStore.slaveUnitGrids"
            :key="unit.unitId"
            class="modbus-panel__unit-tab"
            :class="{ 'modbus-panel__unit-tab--active': unit.unitId === modbusStore.activeSlaveUnitId }"
            :data-testid="`modbus-slave-unit-tab-${unit.unitId}`"
            type="button"
            @click="modbusStore.selectSlaveUnit(unit.unitId)"
          >
            Unit {{ unit.unitId }}
          </button>
        </div>
      </div>

      <div class="modbus-panel__hint" data-testid="modbus-slave-model-hint">
        Backend-backed slave fields are addresses and values only. dataType, comment, and word order are UI-only until the backend data model is expanded.
      </div>

      <section class="modbus-panel__section modbus-panel__collapsible">
        <button
          class="modbus-panel__collapse-header"
          type="button"
          @click="slaveConfigOpen = !slaveConfigOpen"
        >
          <span class="modbus-panel__collapse-label">
            <span class="modbus-panel__section-title">配置</span>
            <span class="modbus-panel__collapse-state">{{ slaveConfigOpen ? '收起' : '展开' }}</span>
          </span>
          <span class="modbus-panel__collapse-summary" data-testid="modbus-slave-config-summary">
            Unit {{ modbusStore.activeSlaveUnitId }} · {{ activeSession?.SlaveRunning ? '运行中' : '未启动' }}
          </span>
        </button>
        <div v-if="slaveConfigOpen" class="modbus-panel__config-strip">
          <NFormItem label="字序">
            <NSelect
              :value="modbusStore.masterGrid.littleEndian ? 'little' : 'big'"
              :options="wordOrderOptions"
              @update:value="value => toggleMasterEndian(value === 'little')"
            />
          </NFormItem>
        </div>
      </section>

      <div class="modbus-panel__slave-grid">
        <section class="modbus-panel__section">
          <div class="modbus-panel__section-header">
            <span class="modbus-panel__section-title">Coils</span>
            <NButton size="tiny" @click="addBoolRow('coils')">添加</NButton>
          </div>
          <ResizableTable
            :columns="slaveBoolColumns"
            table-class="modbus-panel__dense-table modbus-panel__bool-table"
            data-testid="modbus-slave-table-coils"
            :min-width="450"
          >
            <tr v-for="row in activeSlaveGrid.coils" :key="row.id">
              <td><NInputNumber v-model:value="row.address" :min="0" /></td>
              <td><NButton size="tiny" secondary @click="row.value = !row.value">{{ row.value ? 'ON' : 'OFF' }}</NButton></td>
              <td class="modbus-panel__empty">-</td>
              <td>
                <div class="modbus-panel__row-actions">
                  <NButton size="tiny" quaternary @click="removeBoolRow('coils', row)">删除</NButton>
                </div>
              </td>
            </tr>
          </ResizableTable>
        </section>

        <section class="modbus-panel__section">
          <div class="modbus-panel__section-header">
            <span class="modbus-panel__section-title">Discrete Inputs</span>
            <NButton size="tiny" @click="addBoolRow('discreteInputs')">添加</NButton>
          </div>
          <ResizableTable
            :columns="slaveBoolColumns"
            table-class="modbus-panel__dense-table modbus-panel__bool-table"
            data-testid="modbus-slave-table-discreteInputs"
            :min-width="450"
          >
            <tr v-for="row in activeSlaveGrid.discreteInputs" :key="row.id">
              <td><NInputNumber v-model:value="row.address" :min="0" /></td>
              <td><NButton size="tiny" secondary @click="row.value = !row.value">{{ row.value ? 'ON' : 'OFF' }}</NButton></td>
              <td class="modbus-panel__empty">-</td>
              <td>
                <div class="modbus-panel__row-actions">
                  <NButton size="tiny" quaternary @click="removeBoolRow('discreteInputs', row)">删除</NButton>
                </div>
              </td>
            </tr>
          </ResizableTable>
        </section>

        <section class="modbus-panel__section">
          <div class="modbus-panel__section-header">
            <span class="modbus-panel__section-title">Input Registers</span>
            <NButton size="tiny" @click="addRegisterRow('inputRegisters')">添加</NButton>
          </div>
          <ResizableTable
            :columns="slaveRegisterColumns"
            table-class="modbus-panel__dense-table"
            data-testid="modbus-slave-table-inputRegisters"
            :min-width="670"
          >
            <tr v-for="row in activeSlaveGrid.inputRegisters" :key="row.id" :class="mappingClass(mappingForSlaveRegister(row))">
              <td><NInputNumber v-model:value="row.address" :min="0" /></td>
              <td><NInputNumber v-model:value="row.value" :min="0" :max="65535" /></td>
              <td><NSelect :value="row.dataType" :options="dataTypeOptions" @update:value="value => updateSlaveRegisterDataType(row, String(value))" /></td>
              <td><span class="modbus-panel__mapping-pill">{{ row.dataType }}</span></td>
              <td><NInput v-model:value="row.comment" /></td>
              <td>
                <div class="modbus-panel__row-actions">
                  <NButton size="tiny" quaternary>映射</NButton>
                  <NButton size="tiny" quaternary @click="removeRegisterRow('inputRegisters', row)">删除</NButton>
                </div>
              </td>
            </tr>
          </ResizableTable>
        </section>

        <section class="modbus-panel__section">
          <div class="modbus-panel__section-header">
            <span class="modbus-panel__section-title">Holding Registers</span>
            <NButton size="tiny" @click="addRegisterRow('holdingRegisters')">添加</NButton>
          </div>
          <ResizableTable
            :columns="slaveRegisterColumns"
            table-class="modbus-panel__dense-table"
            data-testid="modbus-slave-table-holdingRegisters"
            :min-width="670"
          >
            <tr v-for="row in activeSlaveGrid.holdingRegisters" :key="row.id" :class="mappingClass(mappingForSlaveRegister(row))">
              <td><NInputNumber v-model:value="row.address" :min="0" /></td>
              <td><NInputNumber v-model:value="row.value" :min="0" :max="65535" /></td>
              <td><NSelect :value="row.dataType" :options="dataTypeOptions" @update:value="value => updateSlaveRegisterDataType(row, String(value))" /></td>
              <td><span class="modbus-panel__mapping-pill">{{ row.dataType }}</span></td>
              <td><NInput v-model:value="row.comment" /></td>
              <td>
                <div class="modbus-panel__row-actions">
                  <NButton size="tiny" quaternary>映射</NButton>
                  <NButton size="tiny" quaternary @click="removeRegisterRow('holdingRegisters', row)">删除</NButton>
                </div>
              </td>
            </tr>
          </ResizableTable>
        </section>
      </div>
    </section>

    <div v-if="masterUnitDialogOpen" class="modbus-panel__dialog-backdrop">
      <div class="modbus-panel__dialog" data-testid="modbus-master-unit-dialog">
        <div class="modbus-panel__section-title">添加主站 Unit</div>
        <NFormItem label="Unit ID">
          <NInputNumber
            v-model:value="pendingMasterUnitId"
            :min="1"
            :max="247"
            data-testid="modbus-new-master-unit-id"
          />
        </NFormItem>
        <div class="modbus-panel__dialog-actions">
          <NButton size="small" @click="masterUnitDialogOpen = false">取消</NButton>
          <NButton type="primary" size="small" data-testid="modbus-confirm-master-unit" @click="confirmAddMasterUnit">
            添加
          </NButton>
        </div>
      </div>
    </div>

    <div v-if="slaveUnitDialogOpen" class="modbus-panel__dialog-backdrop">
      <div class="modbus-panel__dialog" data-testid="modbus-slave-unit-dialog">
        <div class="modbus-panel__section-title">添加从站 Unit</div>
        <NFormItem label="Unit ID">
          <NInputNumber
            v-model:value="pendingSlaveUnitId"
            :min="1"
            :max="247"
            data-testid="modbus-new-slave-unit-id"
          />
        </NFormItem>
        <div class="modbus-panel__dialog-actions">
          <NButton size="small" @click="slaveUnitDialogOpen = false">取消</NButton>
          <NButton type="primary" size="small" :loading="slaveLoading" data-testid="modbus-confirm-slave-unit" @click="confirmAddSlaveUnit">
            添加
          </NButton>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.modbus-panel {
  display: flex;
  flex-direction: column;
  gap: 12px;
  height: 100%;
  padding: 12px;
  overflow: auto;
  color: var(--app-text);
  background: var(--app-surface);
}
.modbus-panel--tab {
  background: var(--app-bg);
}
.modbus-panel__header,
.modbus-panel__status,
.modbus-panel__toggle-row,
.modbus-panel__unit-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}
.modbus-panel__unit-row {
  align-items: stretch;
}
.modbus-panel__unit-row .n-select {
  min-width: 0;
  flex: 1;
}
.modbus-panel__section {
  display: flex;
  flex-direction: column;
  gap: 8px;
  min-width: 0;
  border-top: 1px solid var(--app-border);
  padding-top: 10px;
}
.modbus-panel > .modbus-panel__section:first-of-type {
  border-top: 0;
  padding-top: 0;
}
.modbus-panel__section-title {
  font-size: 12px;
  font-weight: 600;
  color: var(--app-text);
}
.modbus-panel__workbench {
  display: flex;
  flex-direction: column;
  gap: 10px;
  min-height: 0;
}
.modbus-panel__toolbar,
.modbus-panel__section-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  min-width: 0;
}
.modbus-panel__toolbar {
  padding: 8px 0;
  border-top: 1px solid var(--app-border);
  border-bottom: 1px solid var(--app-border);
}
.modbus-panel__toolbar-group {
  display: flex;
  align-items: center;
  gap: 6px;
  min-width: 0;
  flex-wrap: wrap;
}
.modbus-panel__unit-tabs {
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
  min-height: 40px;
  padding: 6px 8px;
  border: 1px solid var(--app-border);
  border-radius: 6px;
  background: var(--app-surface);
}
.modbus-panel__unit-tabs-label {
  flex: 0 0 auto;
  font-size: 12px;
  font-weight: 600;
  color: var(--app-text-muted);
}
.modbus-panel__unit-tab-list {
  display: flex;
  align-items: center;
  gap: 4px;
  min-width: 0;
  flex: 1 1 auto;
  overflow-x: auto;
}
.modbus-panel__unit-tab-actions {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  flex: 0 0 auto;
}
.modbus-panel__unit-tab {
  min-height: 28px;
  padding: 0 10px;
  border: 1px solid var(--app-border);
  border-radius: 5px;
  background: transparent;
  color: var(--app-text-muted);
  cursor: pointer;
  font-size: 12px;
  white-space: nowrap;
}
.modbus-panel__unit-tab:hover {
  background: var(--app-hover-bg);
  color: var(--app-text);
}
.modbus-panel__unit-tab--active {
  border-color: var(--app-accent);
  background: color-mix(in srgb, var(--app-accent) 12%, var(--app-surface));
  color: var(--app-text);
}
.modbus-panel__config-strip,
.modbus-panel__scan-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(110px, 1fr));
  gap: 0 8px;
  min-width: 0;
}
.modbus-panel__collapsible {
  padding: 0;
  border: 1px solid var(--app-border);
  border-radius: 6px;
  overflow: hidden;
}
.modbus-panel__collapse-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  width: 100%;
  min-height: 36px;
  padding: 8px 10px;
  border: 0;
  background: var(--app-surface);
  color: inherit;
  text-align: left;
  cursor: pointer;
}
.modbus-panel__collapse-label,
.modbus-panel__collapse-summary {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
}
.modbus-panel__collapse-label {
  flex: 0 0 auto;
}
.modbus-panel__collapse-state,
.modbus-panel__collapse-summary {
  font-size: 12px;
  color: var(--app-text-muted);
}
.modbus-panel__collapse-summary {
  justify-content: flex-end;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.modbus-panel__collapse-header:hover {
  background: var(--app-hover-bg);
}
.modbus-panel__collapsible .modbus-panel__config-strip {
  padding: 0 10px 8px;
}
.modbus-panel__content-grid,
.modbus-panel__slave-grid {
  display: grid;
  grid-template-columns: minmax(360px, 1fr) minmax(360px, 1.2fr);
  gap: 12px;
  min-width: 0;
}
.modbus-panel__register-grid {
  display: grid;
  grid-template-columns: minmax(0, 1fr);
  gap: 12px;
  min-width: 0;
}
.modbus-panel__register-card {
  padding: 10px;
  border: 1px solid var(--app-border);
  border-radius: 6px;
  background: var(--app-surface);
  overflow-x: auto;
}
.modbus-panel__register-card--active {
  border-color: var(--app-accent);
}
.modbus-panel__title-button {
  padding: 0;
  border: 0;
  background: transparent;
  cursor: pointer;
}
.modbus-panel__row-actions {
  display: inline-flex;
  gap: 4px;
  opacity: 0;
  transition: opacity 120ms ease;
}
.modbus-panel tr:hover .modbus-panel__row-actions,
.modbus-panel tr:focus-within .modbus-panel__row-actions {
  opacity: 1;
}
.modbus-panel__mapping-pill {
  display: inline-flex;
  align-items: center;
  min-height: 18px;
  padding: 0 5px;
  border-radius: 4px;
  background: color-mix(in srgb, var(--mapping-color, var(--app-accent)) 28%, transparent);
  color: var(--app-text);
  font-size: 11px;
}
.modbus-panel__mapping-band {
  background:
    linear-gradient(
      90deg,
      color-mix(in srgb, var(--mapping-color) 16%, transparent),
      color-mix(in srgb, var(--mapping-color) 5%, transparent) 72%,
      transparent
    );
  box-shadow: inset 4px 0 0 var(--mapping-color);
}
.modbus-panel__register-row--readonly :deep(.n-input),
.modbus-panel__register-row--readonly :deep(.n-input-number),
.modbus-panel__register-row--readonly :deep(.n-base-selection) {
  opacity: 0.72;
}
.modbus-panel__mapping-band--1 { --mapping-color: #4cc9f0; }
.modbus-panel__mapping-band--2 { --mapping-color: #80ed99; }
.modbus-panel__mapping-band--3 { --mapping-color: #ffd166; }
.modbus-panel__mapping-band--4 { --mapping-color: #f4978e; }
.modbus-panel__mapping-band--5 { --mapping-color: #c77dff; }
.modbus-panel__mapping-band--6 { --mapping-color: #90dbf4; }
.modbus-panel__cell--address,
.modbus-panel__cell--number {
  width: 84px;
}
.modbus-panel__cell--alias {
  min-width: 130px;
}
.modbus-panel__cell--type {
  width: 118px;
}
.modbus-panel__cell--word {
  width: 82px;
}
.modbus-panel__cell--value {
  min-width: 72px;
  overflow-wrap: anywhere;
}
.modbus-panel :deep(.modbus-panel__dense-table th),
.modbus-panel :deep(.modbus-panel__dense-table td) {
  padding: 3px 6px;
  line-height: 1.2;
}
.modbus-panel :deep(.modbus-panel__dense-table .n-input),
.modbus-panel :deep(.modbus-panel__dense-table .n-input-number),
.modbus-panel :deep(.modbus-panel__dense-table .n-base-selection) {
  min-height: 24px;
  font-size: 12px;
}
.modbus-panel :deep(.modbus-panel__dense-table .n-input__input-el),
.modbus-panel :deep(.modbus-panel__dense-table .n-base-selection-label),
.modbus-panel :deep(.modbus-panel__dense-table .n-base-selection-input) {
  height: 24px;
  min-height: 24px;
  line-height: 24px;
}
.modbus-panel :deep(.modbus-panel__dense-table .n-button) {
  min-height: 22px;
  font-size: 12px;
}
.modbus-panel :deep(.modbus-panel__bool-table tbody tr:nth-child(odd) td) {
  background: color-mix(in srgb, var(--app-accent) 7%, transparent);
}
.modbus-panel :deep(.modbus-panel__bool-table tbody tr:nth-child(even) td) {
  background: color-mix(in srgb, var(--app-text-muted) 5%, transparent);
}
.modbus-panel :deep(.modbus-panel__bool-table tbody tr:hover td) {
  background: color-mix(in srgb, var(--app-accent) 13%, transparent);
}
.modbus-panel__unit-select {
  min-width: 120px;
}
.modbus-panel__dialog-backdrop {
  position: fixed;
  inset: 0;
  z-index: 1000;
  display: grid;
  place-items: center;
  background: color-mix(in srgb, var(--app-bg) 62%, transparent);
}
.modbus-panel__dialog {
  display: grid;
  gap: 12px;
  width: min(320px, calc(100vw - 32px));
  padding: 14px;
  border: 1px solid var(--app-border);
  border-radius: 6px;
  background: var(--app-surface);
  color: var(--app-text);
}
.modbus-panel__dialog-actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
}
.modbus-panel__raw-line {
  display: grid;
  grid-template-columns: 40px minmax(0, 1fr);
  gap: 8px;
  align-items: start;
  font-size: 12px;
}
.modbus-panel__raw-line span {
  color: var(--app-text-muted);
}
.modbus-panel__raw-line code {
  overflow-wrap: anywhere;
  color: var(--app-accent);
}
.modbus-panel__bool-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(72px, 1fr));
  gap: 6px;
}
.modbus-panel__bool-cell {
  display: grid;
  gap: 2px;
  min-height: 46px;
  padding: 6px;
  border: 1px solid var(--app-border);
  border-radius: 6px;
  background: var(--app-surface);
  color: inherit;
  text-align: left;
  cursor: pointer;
}
.modbus-panel__bool-cell:hover {
  background: var(--app-hover-bg);
}
.modbus-panel__bool-cell span {
  font-size: 11px;
  color: var(--app-text-muted);
}
.modbus-panel__bool-cell strong {
  font-size: 12px;
}
.modbus-panel__grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 0 8px;
}
.modbus-panel__frame-preview {
  display: grid;
  gap: 6px;
  font-size: 12px;
}
.modbus-panel__frame-preview div,
.modbus-panel__result div,
.modbus-panel__scan-line {
  display: grid;
  grid-template-columns: 34px minmax(0, 1fr);
  gap: 8px;
  align-items: start;
}
.modbus-panel__result {
  display: grid;
  gap: 8px;
  min-width: 0;
  font-size: 12px;
}
.modbus-panel__scan-line {
  font-size: 12px;
}
.modbus-panel__frame-preview span,
.modbus-panel__result span,
.modbus-panel__scan-line span {
  color: var(--app-text-muted);
}
.modbus-panel__frame-preview code,
.modbus-panel__history code,
.modbus-panel__result code,
.modbus-panel__scan-line code {
  white-space: pre-wrap;
  overflow-wrap: anywhere;
  color: var(--app-accent);
}
.modbus-panel__history {
  display: grid;
  gap: 4px;
  width: 100%;
  padding: 8px 0;
  border: 0;
  border-top: 1px solid var(--app-border);
  background: transparent;
  color: inherit;
  text-align: left;
  cursor: pointer;
}
.modbus-panel__history:hover {
  background: var(--app-hover-bg);
}
.modbus-panel__history span {
  overflow-wrap: anywhere;
  font-family: Menlo, Monaco, Consolas, monospace;
  font-size: 12px;
}
.modbus-panel__empty {
  color: var(--app-text-muted);
  font-size: 12px;
}
.modbus-panel :deep(.resizable-table) {
  min-width: 1040px;
  color: var(--app-text);
  background: transparent;
}
.modbus-panel :deep(.resizable-table th) {
  background: var(--app-table-header);
  color: var(--app-text-muted);
}
.modbus-panel :deep(.resizable-table td) {
  background: transparent;
  color: var(--app-text);
}
.modbus-panel :deep(.resizable-table th),
.modbus-panel :deep(.resizable-table td) {
  overflow: hidden;
  border-bottom: 1px solid var(--app-border);
  border-color: var(--app-border);
  text-align: left;
  vertical-align: middle;
}
.modbus-panel :deep(.n-form-item-label) {
  color: var(--app-text-muted);
}
.modbus-panel :deep(.n-input),
.modbus-panel :deep(.n-input-number),
.modbus-panel :deep(.n-base-selection) {
  background-color: var(--app-surface) !important;
  color: var(--app-text) !important;
}
.modbus-panel :deep(.n-input__input-el),
.modbus-panel :deep(.n-base-selection-input),
.modbus-panel :deep(.n-base-selection-input__content) {
  color: var(--app-text) !important;
}
.modbus-panel :deep(.n-input__border),
.modbus-panel :deep(.n-input__state-border),
.modbus-panel :deep(.n-base-selection__border),
.modbus-panel :deep(.n-base-selection__state-border) {
  border: 1px solid var(--app-border) !important;
}
.modbus-panel :deep(.n-base-selection-label) {
  background-color: var(--app-surface) !important;
}
.modbus-panel :deep(.n-input:hover),
.modbus-panel :deep(.n-base-selection:hover) {
  background-color: var(--app-hover-bg) !important;
}
.modbus-panel :deep(.n-button--default-type:not(.n-button--primary-type)) {
  --n-color: var(--app-surface) !important;
  --n-color-hover: var(--app-hover-bg) !important;
  --n-color-pressed: var(--app-hover-bg) !important;
  --n-color-focus: var(--app-hover-bg) !important;
  --n-text-color: var(--app-text) !important;
  --n-text-color-hover: var(--app-text) !important;
  --n-text-color-pressed: var(--app-text) !important;
  --n-text-color-focus: var(--app-text) !important;
  --n-border: 1px solid var(--app-border) !important;
  --n-border-hover: 1px solid var(--app-border) !important;
  --n-border-pressed: 1px solid var(--app-border) !important;
  --n-border-focus: 1px solid var(--app-border) !important;
}
@media (max-width: 960px) {
  .modbus-panel__config-strip,
  .modbus-panel__scan-grid,
  .modbus-panel__register-grid,
  .modbus-panel__content-grid,
  .modbus-panel__slave-grid {
    grid-template-columns: 1fr;
  }
}
</style>
