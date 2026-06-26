<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import {
  NAlert,
  NButton,
  NForm,
  NFormItem,
  NInput,
  NInputNumber,
  NRadioButton,
  NRadioGroup,
  NSelect,
  NSpace,
  NSwitch,
  NTag,
} from 'naive-ui'
import { useSerialStore } from '../stores/serialStore'
import ResizableTable, { type ResizableTableColumn } from '../components/ResizableTable.vue'
import {
  fecbusFunctionOptions,
  useFecbusStore,
} from '../stores/fecbusStore'
import { FrameType, FunctionCode, SessionRole, StatusCode } from '../../../bindings/github.com/littepointR/portweave/internal/modules/serial/fecbus/models.js'

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
const fecbusStore = useFecbusStore()
const loading = ref(false)
const sending = ref(false)
const slaveLoading = ref(false)
const historyLoading = ref(false)
const selectedFrameSeq = ref<number | null>(null)
const newCustomFieldKey = ref('value')

const baudOptions = [4800, 9600, 19200, 38400, 57600, 115200].map(v => ({ label: String(v), value: v }))
const dataBitsOptions = [8].map(v => ({ label: String(v), value: v }))
const stopBitsOptions = [{ label: '1', value: '1' }]
const parityOptions = [{ label: '无', value: 'none' }]
const flowOptions = [
  { label: '无', value: 'none' },
  { label: '硬件 (RTS/CTS)', value: 'hw_rtscts' },
]
const roleOptions = [
  { label: '主控节点', value: 'master' },
  { label: '电气控制装置', value: 'slave' },
]
const frameTypeOptions = [
  { label: '请求 00H', value: FrameType.FrameTypeRequest },
  { label: '应答 01H', value: FrameType.FrameTypeAnswer },
]
const priorityOptions = [
  { label: '0 最高', value: 0 },
  { label: '1', value: 1 },
  { label: '2', value: 2 },
  { label: '3 最低', value: 3 },
]
const statusOptions = [
  { label: '00 分组结束', value: StatusCode.StatusGroupEnd },
  { label: '01 CRC 校验错', value: StatusCode.StatusCRCError },
  { label: '02 无效服务', value: StatusCode.StatusInvalidService },
  { label: '03 单元故障', value: StatusCode.StatusUnitFault },
  { label: '04 忙', value: StatusCode.StatusBusy },
  { label: '05 命令不识别', value: StatusCode.StatusUnrecognizedCommand },
  { label: '06 地址不存在', value: StatusCode.StatusAddressNotFound },
  { label: '07 参数错误', value: StatusCode.StatusParameterError },
  { label: '08 处理中', value: StatusCode.StatusProcessing },
  { label: '09 事件结束', value: StatusCode.StatusEventEnd },
  { label: '0A 接收正确', value: StatusCode.StatusReceivedOK },
]
const directionOptions = [
  { label: '全部', value: '' },
  { label: '发送', value: 'tx' },
  { label: '接收', value: 'rx' },
]
const frameHistoryColumns: ResizableTableColumn[] = [
  { key: 'seq', label: '#', width: 56, minWidth: 44 },
  { key: 'direction', label: '方向', width: 72, minWidth: 56 },
  { key: 'ft', label: 'FT', width: 56, minWidth: 44 },
  { key: 'da', label: 'DA', width: 56, minWidth: 44 },
  { key: 'sa', label: 'SA', width: 56, minWidth: 44 },
  { key: 'mn', label: 'MN', width: 56, minWidth: 44 },
  { key: 'function', label: '功能', width: 180, minWidth: 120 },
  { key: 'hex', label: 'HEX', width: 280, minWidth: 160 },
  { key: 'error', label: '错误', width: 140, minWidth: 90 },
]
const dataFieldColumns: ResizableTableColumn[] = [
  { key: 'field', label: '字段', width: 150, minWidth: 90 },
  { key: 'range', label: '范围', width: 80, minWidth: 64 },
  { key: 'hex', label: 'HEX', width: 160, minWidth: 100 },
  { key: 'value', label: '值', width: 160, minWidth: 100 },
  { key: 'meaning', label: '含义', width: 220, minWidth: 120 },
]
const functionOptions = fecbusFunctionOptions.map(option => ({
  label: option.label,
  value: option.value,
}))
const portOptions = computed(() =>
  serialStore.ports.map(p => ({
    label: p.FriendlyName || p.Name,
    value: p.Name,
  }))
)
const targetSessionId = computed(() => props.sessionId || fecbusStore.activeSessionId)
const activeSession = computed(() =>
  targetSessionId.value ? fecbusStore.sessions.get(targetSessionId.value) ?? null : null
)
const sessionTitle = computed(() => {
  const session = activeSession.value
  if (!session) return 'FECbus'
  const role = session.Role === SessionRole.SessionRoleSlave ? 'Slave' : 'Master'
  const suffix = session.Name || session.Config.PortName
  return `FECbus ${role}${suffix ? ` · ${suffix}` : ''}`
})
const currentFilter = computed(() => {
  const id = targetSessionId.value
  return id ? fecbusStore.frameFilters[id] ?? { direction: '', search: '' } : { direction: '', search: '' }
})
const framePage = computed(() => {
  const id = targetSessionId.value
  return id ? fecbusStore.framePages[id] : null
})
const rows = computed(() => framePage.value?.Frames ?? [])
const lastTransaction = computed(() => fecbusStore.history[0] ?? null)
const selectedFrame = computed(() => rows.value.find(row => row.Seq === selectedFrameSeq.value) ?? rows.value[0] ?? null)
const selectedFunctionOption = computed(() => fecbusFunctionOptions.find(option => option.value === fecbusStore.sendForm.functionCode))
const structuredFields = computed(() => {
  const custom = fecbusStore.customFunctions.find(item => Number(item.Code) === fecbusStore.sendForm.functionCode)?.Fields ?? []
  if (custom.length > 0) return custom
  if (fecbusStore.sendForm.functionCode === FunctionCode.FunctionQueryDeviceStatus || fecbusStore.sendForm.functionCode === FunctionCode.FunctionQueryConfig) {
    return [
      { Key: 'controller_id', Label: '控制器编号', Offset: 1, Length: 1, Type: 'uint8', Endian: 'little', Enum: null, Meaning: '' },
      { Key: 'unit_id', Label: '单元编号', Offset: 2, Length: 1, Type: 'uint8', Endian: 'little', Enum: null, Meaning: '' },
      { Key: 'device_id', Label: '设备编号', Offset: 3, Length: 1, Type: 'uint8', Endian: 'little', Enum: null, Meaning: '' },
      { Key: 'channel_id', Label: '通道编号', Offset: 4, Length: 1, Type: 'uint8', Endian: 'little', Enum: null, Meaning: '' },
    ]
  }
  if (fecbusStore.sendForm.functionCode === FunctionCode.FunctionQueryProtocolVersion) {
    return [{ Key: 'controller_id', Label: '控制器编号', Offset: 1, Length: 1, Type: 'uint8', Endian: 'little', Enum: null, Meaning: '' }]
  }
  return []
})

onMounted(() => {
  selectTargetSession()
  serialStore.refreshPorts()
  fecbusStore.refreshSessions()
  if (targetSessionId.value) {
    fecbusStore.queryFrames(targetSessionId.value).catch(() => undefined)
  }
})

watch(
  () => props.sessionId,
  () => selectTargetSession()
)

function selectTargetSession() {
  if (props.sessionId) {
    fecbusStore.setActiveSession(props.sessionId)
  }
}

async function openSession() {
  loading.value = true
  try {
    const id = await fecbusStore.openSession()
    if (id) emit('opened', id)
  } finally {
    loading.value = false
  }
}

async function refresh() {
  loading.value = true
  try {
    await Promise.all([serialStore.refreshPorts(), fecbusStore.refreshSessions()])
  } finally {
    loading.value = false
  }
}

async function sendRequest() {
  selectTargetSession()
  sending.value = true
  try {
    await fecbusStore.sendRequest()
  } finally {
    sending.value = false
  }
}

async function startSlave() {
  selectTargetSession()
  slaveLoading.value = true
  try {
    await fecbusStore.startSlave()
  } finally {
    slaveLoading.value = false
  }
}

async function stopSlave() {
  selectTargetSession()
  slaveLoading.value = true
  try {
    await fecbusStore.stopSlave()
  } finally {
    slaveLoading.value = false
  }
}

async function updateSlaveState() {
  selectTargetSession()
  slaveLoading.value = true
  try {
    await fecbusStore.updateSlaveState()
  } finally {
    slaveLoading.value = false
  }
}

async function addSlaveUnit() {
  selectTargetSession()
  slaveLoading.value = true
  try {
    await fecbusStore.addSlaveUnit({ ...fecbusStore.slaveForm })
  } finally {
    slaveLoading.value = false
  }
}

async function removeSlaveUnit(address: number) {
  selectTargetSession()
  slaveLoading.value = true
  try {
    await fecbusStore.removeSlaveUnit(address)
  } finally {
    slaveLoading.value = false
  }
}

async function queryFrames() {
  selectTargetSession()
  historyLoading.value = true
  try {
    await fecbusStore.queryFrames(targetSessionId.value)
  } finally {
    historyLoading.value = false
  }
}

async function clearFrames() {
  selectTargetSession()
  await fecbusStore.clearFrames(targetSessionId.value)
}

function setDirection(value: string) {
  const id = targetSessionId.value
  if (!id) return
  fecbusStore.setFrameFilter(id, { direction: value })
}

function setSearch(value: string) {
  const id = targetSessionId.value
  if (!id) return
  fecbusStore.setFrameFilter(id, { search: value })
}

function selectFrame(seq: number) {
  selectedFrameSeq.value = seq
}

function groupClass(index?: number) {
  if (index === undefined || index < 0) return ''
  return `fecbus-panel__frame-row--group-${index % 8}`
}

function addCustomFunction() {
  const code = Number(fecbusStore.sendForm.functionCode)
  fecbusStore.upsertCustomFunction({
    Code: code as FunctionCode,
    Name: selectedFunctionOption.value?.custom ? '用户自定义' : `${selectedFunctionOption.value?.name ?? '功能'}扩展`,
    Description: '',
    Direction: selectedFunctionOption.value?.direction ?? 'custom',
    Answer: true,
    Fields: [],
  })
}

function addCustomField(code: number) {
  const definition = fecbusStore.customFunctions.find(item => Number(item.Code) === code)
  if (!definition) return
  const fields = definition.Fields ?? []
  const key = newCustomFieldKey.value.trim() || `field_${fields.length + 1}`
  fecbusStore.upsertCustomFunction({
    ...definition,
    Fields: [
      ...fields,
      { Key: key, Label: key, Offset: fields.length + 1, Length: 1, Type: 'uint8', Endian: 'little', Enum: null, Meaning: '' },
    ],
  })
  newCustomFieldKey.value = 'value'
}
</script>

<template>
  <section
    class="fecbus-panel"
    :class="{ 'fecbus-panel--tab': variant === 'tab' }"
  >
    <header class="fecbus-panel__header">
      <div>
        <h2>{{ variant === 'tab' ? sessionTitle : 'FECbus 调试' }}</h2>
        <p v-if="variant === 'tab' && activeSession">
          {{ activeSession.Role === SessionRole.SessionRoleSlave ? '电气控制装置' : '主控节点' }}
          · {{ activeSession.Config.BaudRate }} 8N1
          · RX {{ activeSession.RxBytes }} / TX {{ activeSession.TxBytes }}
        </p>
      </div>
      <NSpace>
        <NButton size="small" @click="refresh">刷新</NButton>
      </NSpace>
    </header>

    <NAlert
      v-if="fecbusStore.error"
      type="error"
      class="fecbus-panel__alert"
    >
      {{ fecbusStore.error }}
    </NAlert>

    <template v-if="variant !== 'tab'">
      <NForm
        class="fecbus-panel__form"
        label-placement="top"
      >
        <NFormItem label="会话 ID">
          <NInput v-model:value="fecbusStore.portForm.sessionId" />
        </NFormItem>
        <NFormItem label="名称">
          <NInput v-model:value="fecbusStore.portForm.name" />
        </NFormItem>
        <NFormItem label="串口">
          <NSelect
            v-model:value="fecbusStore.portForm.port"
            :options="portOptions"
            filterable
            tag
          />
        </NFormItem>
        <NFormItem label="角色">
          <NSelect
            v-model:value="fecbusStore.portForm.role"
            :options="roleOptions"
          />
        </NFormItem>
        <div class="fecbus-panel__grid">
          <NFormItem label="波特率">
            <NSelect
              v-model:value="fecbusStore.portForm.baudRate"
              :options="baudOptions"
            />
          </NFormItem>
          <NFormItem label="数据位">
            <NSelect
              v-model:value="fecbusStore.portForm.dataBits"
              :options="dataBitsOptions"
            />
          </NFormItem>
          <NFormItem label="停止位">
            <NSelect
              v-model:value="fecbusStore.portForm.stopBits"
              :options="stopBitsOptions"
            />
          </NFormItem>
          <NFormItem label="校验">
            <NSelect
              v-model:value="fecbusStore.portForm.parity"
              :options="parityOptions"
            />
          </NFormItem>
        </div>
        <NFormItem label="流控">
          <NSelect
            v-model:value="fecbusStore.portForm.flowMode"
            :options="flowOptions"
          />
        </NFormItem>
        <div class="fecbus-panel__grid">
          <NFormItem label="超时 ms">
            <NInputNumber
              v-model:value="fecbusStore.portForm.timeoutMs"
              :min="10"
            />
          </NFormItem>
          <NFormItem label="重试">
            <NInputNumber
              v-model:value="fecbusStore.portForm.retries"
              :min="0"
            />
          </NFormItem>
        </div>
        <NButton
          type="primary"
          :loading="loading"
          @click="openSession"
        >
          打开
        </NButton>
      </NForm>
    </template>

    <template v-else-if="activeSession">
      <div class="fecbus-panel__workbench">
        <section
          class="fecbus-panel__section"
          data-testid="fecbus-send-form"
        >
          <div class="fecbus-panel__section-header">
            <h3>发送帧</h3>
            <NButton
              size="small"
              type="primary"
              :loading="sending"
              @click="sendRequest"
            >
              发送
            </NButton>
          </div>
          <div class="fecbus-panel__form-grid">
            <label>
              <span>帧类型</span>
              <NSelect
                v-model:value="fecbusStore.sendForm.frameType"
                :options="frameTypeOptions"
              />
            </label>
            <label>
              <span>目标地址</span>
              <NInputNumber
                v-model:value="fecbusStore.sendForm.targetAddress"
                :min="0"
                :max="63"
              />
            </label>
            <label>
              <span>源地址</span>
              <NInputNumber
                v-model:value="fecbusStore.sendForm.sourceAddress"
                :min="1"
                :max="63"
              />
            </label>
            <label>
              <span>优先级</span>
              <NSelect
                v-model:value="fecbusStore.sendForm.priority"
                :options="priorityOptions"
              />
            </label>
            <label>
              <span>报文编号</span>
              <NInputNumber
                v-model:value="fecbusStore.sendForm.messageNumber"
                :min="1"
                :max="63"
              />
            </label>
            <label>
              <span>分组编号</span>
              <NInputNumber
                v-model:value="fecbusStore.sendForm.groupNumber"
                :min="0"
                :max="127"
              />
            </label>
            <label>
              <span>功能码</span>
              <NSelect
                v-model:value="fecbusStore.sendForm.functionCode"
                :options="functionOptions"
                filterable
              />
            </label>
            <label>
              <span>应答</span>
              <NSwitch v-model:value="fecbusStore.sendForm.expectAnswer" />
            </label>
          </div>
          <label class="fecbus-panel__wide-field">
            <span>数据输入</span>
            <NRadioGroup
              v-model:value="fecbusStore.sendForm.inputMode"
              size="small"
            >
              <NRadioButton value="hex">HEX</NRadioButton>
              <NRadioButton value="structured">结构化</NRadioButton>
            </NRadioGroup>
          </label>
          <div
            v-if="fecbusStore.sendForm.inputMode === 'structured' && structuredFields.length"
            class="fecbus-panel__form-grid"
          >
            <label
              v-for="field in structuredFields"
              :key="field.Key"
            >
              <span>{{ field.Label || field.Key }}</span>
              <NInput
                v-model:value="fecbusStore.sendForm.structuredFields[field.Key]"
                :placeholder="`${field.Type || 'uint8'} · ${field.Length} 字节`"
              />
            </label>
          </div>
          <label
            v-else
            class="fecbus-panel__wide-field"
          >
            <span>数据 HEX</span>
            <NInput
              v-model:value="fecbusStore.sendForm.payloadHex"
              type="textarea"
              placeholder="功能码后的数据字节，按 xx 空格格式输入"
            />
          </label>
          <div
            v-if="lastTransaction"
            class="fecbus-panel__transaction"
          >
            <span>{{ lastTransaction.RequestFrameHex }}</span>
            <span v-if="lastTransaction.ResponseFrameHex">{{ lastTransaction.ResponseFrameHex }}</span>
            <span v-if="lastTransaction.Error">{{ lastTransaction.Error }}</span>
          </div>
        </section>

        <section class="fecbus-panel__section">
          <div class="fecbus-panel__section-header">
            <h3>设备从站</h3>
            <NSpace>
              <NButton
                size="small"
                :loading="slaveLoading"
                @click="addSlaveUnit"
              >
                添加/更新
              </NButton>
              <NButton
                size="small"
                type="primary"
                :loading="slaveLoading"
                @click="startSlave"
              >
                启动
              </NButton>
              <NButton
                size="small"
                :loading="slaveLoading"
                @click="stopSlave"
              >
                停止
              </NButton>
            </NSpace>
          </div>
          <div class="fecbus-panel__form-grid">
            <label>
              <span>设备地址</span>
              <NInputNumber
                v-model:value="fecbusStore.slaveForm.address"
                :min="1"
                :max="63"
              />
            </label>
            <label>
              <span>状态应答</span>
              <NSelect
                v-model:value="fecbusStore.slaveForm.statusCode"
                :options="statusOptions"
              />
            </label>
            <label>
              <span>自动应答</span>
              <NSwitch v-model:value="fecbusStore.slaveForm.autoStatusAnswer" />
            </label>
            <label>
              <span>接受广播</span>
              <NSwitch v-model:value="fecbusStore.slaveForm.acceptBroadcast" />
            </label>
          </div>
          <div class="fecbus-panel__unit-list">
            <button
              v-for="unit in fecbusStore.slaveUnits"
              :key="unit.address"
              type="button"
              class="fecbus-panel__unit"
              :class="{ 'fecbus-panel__unit--active': unit.address === fecbusStore.slaveForm.address }"
              @click="fecbusStore.slaveForm = { ...unit }"
            >
              <span>地址 {{ unit.address }}</span>
              <span>{{ unit.statusCode }}</span>
              <NButton
                size="tiny"
                quaternary
                @click.stop="removeSlaveUnit(unit.address)"
              >
                删除
              </NButton>
            </button>
          </div>
        </section>

        <section
          class="fecbus-panel__section fecbus-panel__history"
          data-testid="fecbus-frame-history"
        >
          <div class="fecbus-panel__section-header">
            <h3>帧历史</h3>
            <NSpace>
              <NSelect
                :value="currentFilter.direction"
                :options="directionOptions"
                size="small"
                @update:value="setDirection"
              />
              <NInput
                :value="currentFilter.search"
                size="small"
                placeholder="搜索"
                @update:value="setSearch"
              />
              <NButton
                size="small"
                :loading="historyLoading"
                @click="queryFrames"
              >
                查询
              </NButton>
              <NButton
                size="small"
                @click="clearFrames"
              >
                清空
              </NButton>
            </NSpace>
          </div>
          <ResizableTable :columns="frameHistoryColumns" table-class="fecbus-panel__resizable-table" :min-width="950">
            <tr
              v-for="row in rows"
              :key="row.Seq"
              class="fecbus-panel__frame-row"
              :class="groupClass(row.Annotated?.GroupColorIndex)"
              @click="selectFrame(row.Seq)"
            >
              <td>{{ row.Seq }}</td>
              <td>
                <NTag size="small">{{ row.Direction === 'tx' ? '发送' : '接收' }}</NTag>
              </td>
              <td>{{ row.Frame?.Type ?? '' }}</td>
              <td>{{ row.Frame?.TargetAddress ?? '' }}</td>
              <td>{{ row.Frame?.SourceAddress ?? '' }}</td>
              <td>{{ row.Frame?.MessageNumber ?? '' }}</td>
              <td>{{ row.Annotated?.Function?.Name || (row.Frame?.Data ? row.Frame.Data.slice(0, 2) : '') }}</td>
              <td class="fecbus-panel__hex">{{ row.Hex }}</td>
              <td>{{ row.Error }}</td>
            </tr>
          </ResizableTable>
          <div
            v-if="selectedFrame?.Annotated"
            class="fecbus-panel__frame-detail"
          >
            <div class="fecbus-panel__segment-list">
              <span
                v-for="segment in selectedFrame.Annotated.Segments"
                :key="`${segment.Key}-${segment.Start}`"
                class="fecbus-panel__segment"
              >
                {{ segment.Label }} {{ segment.Hex }} {{ segment.Meaning }}
              </span>
            </div>
            <ResizableTable
              v-if="selectedFrame.Annotated.DataFields?.length"
              :columns="dataFieldColumns"
              table-class="fecbus-panel__resizable-table"
              :min-width="770"
            >
              <tr
                v-for="field in selectedFrame.Annotated.DataFields"
                :key="`${field.Key}-${field.Start}`"
              >
                <td>{{ field.Label }}</td>
                <td>{{ field.Start }}-{{ field.End }}</td>
                <td class="fecbus-panel__hex">{{ field.Hex }}</td>
                <td>{{ field.ValueText }}</td>
                <td>{{ field.Meaning }}</td>
              </tr>
            </ResizableTable>
          </div>
        </section>

        <section class="fecbus-panel__section">
          <div class="fecbus-panel__section-header">
            <h3>功能码与自定义数据段</h3>
            <NButton
              size="small"
              @click="addCustomFunction"
            >
              添加当前功能码定义
            </NButton>
          </div>
          <div class="fecbus-panel__function-grid">
            <span
              v-for="option in fecbusFunctionOptions"
              :key="option.value"
              class="fecbus-panel__function-chip"
            >
              {{ option.label }}
            </span>
          </div>
          <div class="fecbus-panel__custom-functions">
            <div
              v-for="definition in fecbusStore.customFunctions"
              :key="definition.Code"
              class="fecbus-panel__custom-function"
            >
              <div class="fecbus-panel__custom-function-header">
                <strong>{{ Number(definition.Code).toString(16).toUpperCase().padStart(2, '0') }} {{ definition.Name }}</strong>
                <NSpace>
                  <NInput
                    v-model:value="newCustomFieldKey"
                    size="small"
                    placeholder="字段 key"
                  />
                  <NButton
                    size="small"
                    @click="addCustomField(Number(definition.Code))"
                  >
                    添加字段
                  </NButton>
                  <NButton
                    size="small"
                    @click="fecbusStore.removeCustomFunction(Number(definition.Code))"
                  >
                    删除
                  </NButton>
                </NSpace>
              </div>
              <div class="fecbus-panel__form-grid">
                <template
                  v-for="field in definition.Fields"
                  :key="field.Key"
                >
                  <label>
                    <span>字段</span>
                    <NInput v-model:value="field.Label" />
                  </label>
                  <label>
                    <span>偏移</span>
                    <NInputNumber
                      v-model:value="field.Offset"
                      :min="1"
                      :max="7"
                    />
                  </label>
                  <label>
                    <span>长度</span>
                    <NInputNumber
                      v-model:value="field.Length"
                      :min="1"
                      :max="7"
                    />
                  </label>
                  <label>
                    <span>类型</span>
                    <NSelect
                      v-model:value="field.Type"
                      :options="[
                        { label: 'uint8', value: 'uint8' },
                        { label: 'uint16', value: 'uint16' },
                        { label: 'HEX', value: 'hex' },
                        { label: 'UTF-8', value: 'utf8' },
                      ]"
                    />
                  </label>
                </template>
              </div>
            </div>
          </div>
        </section>
      </div>
    </template>

    <div
      v-else
      class="fecbus-panel__empty"
    >
      选择或打开 FECbus 会话
    </div>
  </section>
</template>

<style scoped>
.fecbus-panel {
  height: 100%;
  min-height: 0;
  display: flex;
  flex-direction: column;
  background: var(--app-surface, #252526);
  color: var(--app-text, #cccccc);
  overflow: hidden;
}
.fecbus-panel--tab {
  background: var(--app-bg, #1e1e1e);
}
.fecbus-panel__header {
  flex: 0 0 auto;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 12px 14px;
  border-bottom: 1px solid var(--app-border, #2d2d2d);
}
.fecbus-panel__header h2,
.fecbus-panel__section h3 {
  margin: 0;
  font-size: 14px;
  font-weight: 600;
}
.fecbus-panel__header p {
  margin: 4px 0 0;
  color: var(--app-text-muted, #858585);
  font-size: 12px;
}
.fecbus-panel__alert {
  margin: 10px 12px 0;
}
.fecbus-panel__form {
  padding: 12px;
  overflow: auto;
}
.fecbus-panel__grid,
.fecbus-panel__form-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 10px;
}
.fecbus-panel__workbench {
  flex: 1;
  min-height: 0;
  overflow: auto;
  padding: 12px;
}
.fecbus-panel__section {
  margin-bottom: 12px;
  border: 1px solid var(--app-border, #2d2d2d);
  border-radius: 6px;
  background: var(--app-surface, #252526);
}
.fecbus-panel__section-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 8px 10px;
  border-bottom: 1px solid var(--app-border, #2d2d2d);
}
.fecbus-panel__form-grid {
  padding: 10px;
  grid-template-columns: repeat(4, minmax(120px, 1fr));
}
.fecbus-panel__form-grid label,
.fecbus-panel__wide-field {
  display: flex;
  flex-direction: column;
  gap: 4px;
  min-width: 0;
  color: var(--app-text-muted, #858585);
  font-size: 12px;
}
.fecbus-panel__wide-field {
  padding: 0 10px 10px;
}
.fecbus-panel__transaction {
  display: flex;
  flex-direction: column;
  gap: 4px;
  padding: 0 10px 10px;
  color: var(--app-text-muted, #858585);
  font-family: var(--serial-terminal-font-family, Menlo, Consolas, monospace);
  font-size: 12px;
}
.fecbus-panel__unit-list,
.fecbus-panel__custom-functions,
.fecbus-panel__frame-detail {
  padding: 0 10px 10px;
}
.fecbus-panel__unit-list {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}
.fecbus-panel__unit {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  border: 1px solid var(--app-border, #2d2d2d);
  border-radius: 4px;
  padding: 4px 6px;
  background: transparent;
  color: var(--app-text, #cccccc);
  font-size: 12px;
  cursor: pointer;
}
.fecbus-panel__unit--active {
  border-color: var(--app-accent, #409eff);
}
.fecbus-panel :deep(.fecbus-panel__resizable-table) {
  width: 100%;
  color: var(--app-text, #cccccc);
  font-size: 12px;
}
.fecbus-panel :deep(.fecbus-panel__resizable-table th),
.fecbus-panel :deep(.fecbus-panel__resizable-table td) {
  overflow: hidden;
  padding: 5px 8px;
  border-bottom: 1px solid var(--app-border, #2d2d2d);
  text-align: left;
  text-overflow: ellipsis;
  vertical-align: middle;
}
.fecbus-panel :deep(.fecbus-panel__resizable-table th) {
  background: var(--app-table-header, #252526);
  color: var(--app-text-muted, #858585);
  font-weight: 500;
}
.fecbus-panel__frame-row {
  cursor: pointer;
}
.fecbus-panel__frame-row--group-0 { background: color-mix(in srgb, #3b82f6 16%, transparent); }
.fecbus-panel__frame-row--group-1 { background: color-mix(in srgb, #22c55e 16%, transparent); }
.fecbus-panel__frame-row--group-2 { background: color-mix(in srgb, #f59e0b 16%, transparent); }
.fecbus-panel__frame-row--group-3 { background: color-mix(in srgb, #ef4444 16%, transparent); }
.fecbus-panel__frame-row--group-4 { background: color-mix(in srgb, #14b8a6 16%, transparent); }
.fecbus-panel__frame-row--group-5 { background: color-mix(in srgb, #a855f7 16%, transparent); }
.fecbus-panel__frame-row--group-6 { background: color-mix(in srgb, #84cc16 16%, transparent); }
.fecbus-panel__frame-row--group-7 { background: color-mix(in srgb, #ec4899 16%, transparent); }
.fecbus-panel__hex {
  font-family: var(--serial-terminal-font-family, Menlo, Consolas, monospace);
  word-break: break-all;
}
.fecbus-panel__segment-list {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  margin-bottom: 8px;
}
.fecbus-panel__segment {
  border: 1px solid var(--app-border, #2d2d2d);
  border-radius: 4px;
  padding: 3px 5px;
  color: var(--app-text-muted, #858585);
  font-family: var(--serial-terminal-font-family, Menlo, Consolas, monospace);
  font-size: 11px;
}
.fecbus-panel__function-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(150px, 1fr));
  gap: 6px;
  padding: 10px;
}
.fecbus-panel__function-chip {
  padding: 4px 6px;
  border: 1px solid var(--app-border, #2d2d2d);
  border-radius: 4px;
  color: var(--app-text, #cccccc);
  font-size: 12px;
}
.fecbus-panel__custom-function {
  border-top: 1px solid var(--app-border, #2d2d2d);
  padding-top: 10px;
  margin-top: 10px;
}
.fecbus-panel__custom-function-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  margin-bottom: 8px;
  font-size: 12px;
}
.fecbus-panel__empty {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--app-text-muted, #858585);
}
@media (max-width: 900px) {
  .fecbus-panel__form-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}
</style>
