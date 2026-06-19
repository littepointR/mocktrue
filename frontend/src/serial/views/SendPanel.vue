<script setup lang="ts">
import { onUnmounted, ref, watch } from 'vue'
import { NInput, NButton, NSelect, NInputNumber, NSwitch } from 'naive-ui'
import { Send } from '../../../bindings/github.com/suyue/mocktrue/internal/modules/serial/service.js'
import { useSerialStore } from '../stores/serialStore'
import { asciiToHexText, formatHexInput, hexTextToAscii } from '../utils/bytes'

const props = defineProps<{
  handleId: string
}>()

const serialStore = useSerialStore()
const sendData = ref('')
const sendMode = ref<'ascii' | 'hex'>('ascii')
const autoSend = ref(false)
const sendIntervalMs = ref(1000)
const sendHistory = ref<Array<{ id: number; content: string; mode: 'ascii' | 'hex' }>>([])
const error = ref<string | null>(null)
let nextHistoryId = 1
let autoSendTimer: number | null = null
let skipNextSendDataWatch = false

async function sendContent(content: string, mode = sendMode.value, clearEditor = false) {
  if (!props.handleId || !content) return
  try {
    const sentBytes = await Send({
      PortID: props.handleId,
      Content: content,
      Mode: mode,
    })
    serialStore.addTxBytes(props.handleId, sentBytes)
    addHistory(content, mode)
    if (clearEditor) {
      sendData.value = ''
    }
    error.value = null
  } catch (e: any) {
    error.value = e?.message ?? 'Send failed'
  }
}

function addHistory(content: string, mode: 'ascii' | 'hex') {
  sendHistory.value = [
    { id: nextHistoryId++, content, mode },
    ...sendHistory.value.filter(item => item.content !== content || item.mode !== mode),
  ].slice(0, 20)
}

async function handleSend() {
  await sendContent(sendData.value, sendMode.value, false)
}

async function resendHistoryItem(item: { content: string; mode: 'ascii' | 'hex' }) {
  await sendContent(item.content, item.mode, false)
}

function stopAutoSend() {
  if (autoSendTimer !== null) {
    window.clearInterval(autoSendTimer)
    autoSendTimer = null
  }
}

function startAutoSend() {
  stopAutoSend()
  if (!autoSend.value || !sendData.value) return

  const interval = Math.max(10, sendIntervalMs.value || 10)
  autoSendTimer = window.setInterval(() => {
    void sendContent(sendData.value, sendMode.value, false)
  }, interval)
}

watch([autoSend, sendIntervalMs], () => {
  startAutoSend()
})

watch(sendData, (value) => {
  if (skipNextSendDataWatch) {
    skipNextSendDataWatch = false
    return
  }

  if (sendMode.value === 'hex') {
    const formatted = formatHexInput(value)
    if (formatted !== value) {
      setSendData(formatted)
      syncAutoSendForContent(formatted)
      return
    }
  }

  syncAutoSendForContent(value)
})

function syncAutoSendForContent(value: string) {
  if (!value && autoSend.value) {
    autoSend.value = false
  } else if (autoSend.value) {
    startAutoSend()
  }
}

watch(sendMode, (mode, previousMode) => {
  if (mode === previousMode) return
  try {
    if (mode === 'hex') {
      setSendData(asciiToHexText(sendData.value))
    } else {
      setSendData(hexTextToAscii(sendData.value))
    }
    error.value = null
  } catch (e: any) {
    error.value = e?.message ?? 'Convert send content failed'
  }
})

function setSendData(value: string) {
  if (sendData.value === value) return
  skipNextSendDataWatch = true
  sendData.value = value
}

onUnmounted(stopAutoSend)
</script>

<template>
  <div class="send-panel">
    <div class="send-panel__compose">
      <div class="send-panel__toolbar">
        <NSelect
          v-model:value="sendMode"
          class="send-panel__mode"
          :options="[
            { label: 'ASCII', value: 'ascii' },
            { label: 'HEX', value: 'hex' },
          ]"
          size="small"
          style="width: 100px"
        />
        <label class="send-panel__interval">
          <span>发送间隔</span>
          <NInputNumber
            v-model:value="sendIntervalMs"
            size="small"
            :min="10"
            :step="10"
            :show-button="false"
          />
          <span>ms</span>
        </label>
        <span class="send-panel__auto-send">
          <span>自动发送</span>
          <NSwitch
            v-model:value="autoSend"
            size="small"
            :disabled="!sendData"
          />
        </span>
        <NButton
          type="primary"
          size="small"
          :disabled="!sendData"
          @click="handleSend"
        >
          发送
        </NButton>
      </div>
      <NInput
        v-model:value="sendData"
        class="send-panel__editor"
        placeholder="输入要发送的数据"
        type="textarea"
        size="small"
        :autosize="false"
      />
      <div v-if="error" class="send-panel__error">{{ error }}</div>
    </div>

    <div class="send-panel__history">
      <button
        v-for="item in sendHistory"
        :key="item.id"
        class="send-panel__history-item"
        type="button"
        @click="resendHistoryItem(item)"
      >
        <span class="send-panel__history-mode">{{ item.mode.toUpperCase() }}</span>
        <span class="send-panel__history-content">{{ item.content }}</span>
      </button>
    </div>
  </div>
</template>

<style scoped>
.send-panel {
  display: flex;
  gap: 8px;
  height: 100%;
  padding: 8px 12px;
  background: #252526;
  overflow: hidden;
}
.send-panel__compose {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-width: 0;
  min-height: 0;
  gap: 6px;
}
.send-panel__toolbar {
  display: flex;
  align-items: center;
  justify-content: flex-start;
  gap: 8px;
  flex-shrink: 0;
  min-width: 0;
  flex-wrap: wrap;
}
.send-panel__interval,
.send-panel__auto-send {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  color: #858585;
  font-size: 12px;
}
.send-panel__interval :deep(.n-input-number) {
  width: 76px;
}
.send-panel__toolbar :deep(.n-button) {
  margin-left: auto;
}
.send-panel__editor {
  flex: 1;
  min-height: 76px;
}
.send-panel__editor :deep(.n-input),
.send-panel__editor :deep(.n-input-wrapper),
.send-panel__editor :deep(.n-input__textarea) {
  height: 100%;
}
.send-panel__editor :deep(textarea) {
  height: 100% !important;
  resize: none;
  font-family: 'Consolas', 'Monaco', 'Courier New', monospace;
}
.send-panel__history {
  width: 220px;
  flex: 0 0 220px;
  overflow-y: auto;
  border-left: 1px solid #2d2d2d;
  padding-left: 8px;
}
.send-panel__history-item {
  display: flex;
  align-items: flex-start;
  width: 100%;
  gap: 6px;
  min-height: 28px;
  padding: 5px 6px;
  border: 0;
  border-radius: 3px;
  background: transparent;
  color: #cccccc;
  cursor: pointer;
  text-align: left;
  font: inherit;
}
.send-panel__history-item:hover {
  background: #2d2d2d;
}
.send-panel__history-mode {
  flex: 0 0 auto;
  color: #858585;
  font-size: 11px;
  line-height: 18px;
}
.send-panel__history-content {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  line-height: 18px;
}
.send-panel__error {
  flex-shrink: 0;
  color: #f48771;
  font-size: 12px;
}
</style>
