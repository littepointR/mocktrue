<script setup lang="ts">
import { computed, onUnmounted, ref, watch } from 'vue'
import { NButton, NForm, NFormItem, NSelect } from 'naive-ui'
import { useSerialStore } from '../stores/serialStore'
import DataDisplay from './DataDisplay.vue'
import SendPanel from './SendPanel.vue'
import StatsPanel from './StatsPanel.vue'
import { useSerialWorkspaceStore } from '../stores/workspaceStore'

const props = defineProps<{
  handleId: string
}>()

const store = useSerialStore()
const workspaceStore = useSerialWorkspaceStore()
const handle = computed(() => store.handles.get(props.handleId))
const showConfig = computed({
  get: () => workspaceStore.tabState(props.handleId).showConfig,
  set: value => workspaceStore.updateTabState(props.handleId, { showConfig: value }),
})
const configSaving = ref(false)
const baudRate = ref(115200)
const dataBits = ref(8)
const stopBits = ref('1')
const parity = ref('none')
const flowMode = ref('none')

const baudOptions = [9600, 19200, 38400, 57600, 115200, 230400, 460800, 921600, 1000000, 2000000, 4000000].map(v => ({
  label: String(v),
  value: v,
}))
const dataBitsOptions = [5, 6, 7, 8].map(v => ({
  label: String(v),
  value: v,
}))
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

const minDisplayHeight = 120
const minSendHeight = 48
const sendHeight = computed({
  get: () => workspaceStore.tabState(props.handleId).sendHeight,
  set: value => workspaceStore.updateTabState(props.handleId, { sendHeight: value }),
})
const rootEl = ref<HTMLElement | null>(null)
const isResizing = ref(false)

let resizeStartY = 0
let resizeStartSendHeight = 0

function clampSendHeight(value: number): number {
  const rootHeight = rootEl.value?.clientHeight ?? 0
  const statsHeight = rootEl.value?.querySelector<HTMLElement>('.serial-tab-content__stats')?.offsetHeight ?? 0
  const handleHeight = rootEl.value?.querySelector<HTMLElement>('.serial-tab-content__resize-handle')?.offsetHeight ?? 0
  const maxSendHeight = Math.max(minSendHeight, rootHeight - statsHeight - handleHeight - minDisplayHeight)
  return Math.min(Math.max(value, minSendHeight), maxSendHeight)
}

function resizeTo(clientY: number) {
  sendHeight.value = clampSendHeight(resizeStartSendHeight + resizeStartY - clientY)
}

function stopResize() {
  if (!isResizing.value) return
  isResizing.value = false
  window.removeEventListener('pointermove', handlePointerMove)
  window.removeEventListener('pointerup', stopResize)
  window.removeEventListener('pointercancel', stopResize)
  document.body.style.cursor = ''
  document.body.style.userSelect = ''
}

function handlePointerMove(event: PointerEvent) {
  resizeTo(event.clientY)
}

function startResize(event: PointerEvent) {
  event.preventDefault()
  resizeStartY = event.clientY
  resizeStartSendHeight = sendHeight.value
  isResizing.value = true
  document.body.style.cursor = 'row-resize'
  document.body.style.userSelect = 'none'
  window.addEventListener('pointermove', handlePointerMove)
  window.addEventListener('pointerup', stopResize)
  window.addEventListener('pointercancel', stopResize)
}

function handleResizeKeydown(event: KeyboardEvent) {
  const step = event.shiftKey ? 40 : 12
  if (event.key === 'ArrowUp') {
    event.preventDefault()
    sendHeight.value = clampSendHeight(sendHeight.value + step)
  } else if (event.key === 'ArrowDown') {
    event.preventDefault()
    sendHeight.value = clampSendHeight(sendHeight.value - step)
  }
}

watch(
  handle,
  (value) => {
    if (!value) return
    baudRate.value = value.Config.BaudRate
    dataBits.value = value.Config.DataBits
    stopBits.value = value.Config.StopBits
    parity.value = value.Config.Parity
    flowMode.value = value.Config.FlowMode
  },
  { immediate: true }
)

async function applyConfig() {
  if (!handle.value) return
  configSaving.value = true
  try {
    await store.updatePortConfig(props.handleId, {
      ...handle.value.Config,
      BaudRate: baudRate.value,
      DataBits: dataBits.value,
      StopBits: stopBits.value,
      Parity: parity.value,
      FlowMode: flowMode.value,
    })
  } finally {
    configSaving.value = false
  }
}

onUnmounted(stopResize)
</script>

<template>
  <div
    ref="rootEl"
    class="serial-tab-content"
    :class="{ 'serial-tab-content--resizing': isResizing }"
  >
    <div v-if="handle" class="serial-tab-config">
      <button
        class="serial-tab-config__summary"
        type="button"
        :aria-expanded="showConfig"
        @click="showConfig = !showConfig"
      >
        <span class="serial-tab-config__title">串口配置</span>
        <span class="serial-tab-config__port">{{ handle.Config.PortName }}</span>
        <span class="serial-tab-config__meta">{{ handle.Config.BaudRate }} bps</span>
        <span class="serial-tab-config__chevron">{{ showConfig ? '⌃' : '⌄' }}</span>
      </button>
      <div v-if="showConfig" class="serial-tab-config__details">
        <NForm
          class="serial-tab-config__form"
          label-placement="top"
          size="small"
        >
          <div class="serial-tab-config__field serial-tab-config__field--port">
            <span>端口</span>
            <strong>{{ handle.Config.PortName }}</strong>
          </div>
          <NFormItem class="serial-tab-config__field" label="波特率">
            <NSelect v-model:value="baudRate" :options="baudOptions" :disabled="configSaving" />
          </NFormItem>
          <NFormItem class="serial-tab-config__field" label="数据位">
            <NSelect v-model:value="dataBits" :options="dataBitsOptions" :disabled="configSaving" />
          </NFormItem>
          <NFormItem class="serial-tab-config__field" label="停止位">
            <NSelect v-model:value="stopBits" :options="stopBitsOptions" :disabled="configSaving" />
          </NFormItem>
          <NFormItem class="serial-tab-config__field" label="校验">
            <NSelect v-model:value="parity" :options="parityOptions" :disabled="configSaving" />
          </NFormItem>
          <NFormItem class="serial-tab-config__field" label="流控">
            <NSelect v-model:value="flowMode" :options="flowOptions" :disabled="configSaving" />
          </NFormItem>
          <NButton
            class="serial-tab-config__apply"
            type="primary"
            size="small"
            :loading="configSaving"
            @click="applyConfig"
          >
            应用配置
          </NButton>
        </NForm>
      </div>
    </div>
    <div class="serial-tab-content__display">
      <DataDisplay :handle-id="handleId" />
    </div>
    <div
      class="serial-tab-content__resize-handle"
      role="separator"
      aria-label="调整接收和发送区域大小"
      aria-orientation="horizontal"
      tabindex="0"
      @pointerdown="startResize"
      @keydown="handleResizeKeydown"
    />
    <div
      class="serial-tab-content__send"
      :style="{ flexBasis: `${sendHeight}px` }"
    >
      <SendPanel :handle-id="handleId" />
    </div>
    <div class="serial-tab-content__stats">
      <StatsPanel :handle-id="handleId" />
    </div>
  </div>
</template>

<style scoped>
.serial-tab-content {
  display: flex;
  flex-direction: column;
  width: 100%;
  height: 100%;
  overflow: hidden;
}
.serial-tab-config {
  flex: 0 0 auto;
  border-bottom: 1px solid #2d2d2d;
  background: #252526;
}
.serial-tab-config__summary {
  display: grid;
  grid-template-columns: auto minmax(0, 1fr) auto auto;
  align-items: center;
  gap: 10px;
  width: 100%;
  height: 30px;
  padding: 0 10px;
  border: 0;
  background: transparent;
  color: #cccccc;
  cursor: pointer;
  font: inherit;
  text-align: left;
}
.serial-tab-config__summary:hover {
  background: #2d2d2d;
}
.serial-tab-config__title {
  color: #858585;
  font-size: 12px;
}
.serial-tab-config__port,
.serial-tab-config__meta {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 12px;
}
.serial-tab-config__meta,
.serial-tab-config__chevron {
  color: #858585;
}
.serial-tab-config__details {
  padding: 8px 10px 10px;
  border-top: 1px solid #2d2d2d;
}
.serial-tab-config__form {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(120px, 1fr));
  align-items: end;
  gap: 8px 12px;
}
.serial-tab-config__field {
  min-width: 0;
  font-size: 12px;
}
.serial-tab-config__field :deep(.n-form-item-label) {
  color: #858585;
  font-size: 12px;
}
.serial-tab-config__field--port span {
  display: block;
  margin-bottom: 2px;
  color: #858585;
}
.serial-tab-config__field--port strong {
  display: block;
  overflow: hidden;
  color: #d4d4d4;
  font-weight: 500;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.serial-tab-config__apply {
  align-self: end;
}
.serial-tab-content__display {
  flex: 1;
  min-height: 0;
  overflow: hidden;
}
.serial-tab-content__resize-handle {
  position: relative;
  flex: 0 0 6px;
  border-top: 1px solid #2d2d2d;
  border-bottom: 1px solid #2d2d2d;
  background: #252526;
  cursor: row-resize;
}
.serial-tab-content__resize-handle::before {
  content: '';
  position: absolute;
  top: 2px;
  left: 50%;
  width: 36px;
  height: 2px;
  border-radius: 1px;
  background: #5a5a5a;
  transform: translateX(-50%);
}
.serial-tab-content__resize-handle:hover::before,
.serial-tab-content__resize-handle:focus-visible::before,
.serial-tab-content--resizing .serial-tab-content__resize-handle::before {
  background: #007acc;
}
.serial-tab-content__send {
  flex: 0 0 180px;
  min-height: 48px;
  background: #252526;
  overflow: hidden;
}
.serial-tab-content__stats {
  flex: 0 0 auto;
  border-top: 1px solid #2d2d2d;
  background: #252526;
}
</style>
