<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { NConfigProvider, darkTheme, lightTheme, type GlobalThemeOverrides } from 'naive-ui'
import ActivityBar from './shell/ActivityBar.vue'
import Sidebar from './shell/Sidebar.vue'
import EditorGroups from './shell/EditorGroups.vue'
import Panel from './shell/Panel.vue'
import StatusBar from './shell/StatusBar.vue'
import { useRegistry } from './core/registry'
import { useSettingsStore } from './settings'
import { useWorkspaceFileStore } from './workspace/stores/workspaceFileStore'
import { useSerialGraphStore } from './serial/stores/graphStore'

const registry = useRegistry()
const settings = useSettingsStore()
const workspaceFile = useWorkspaceFileStore()
const graphStore = useSerialGraphStore()
const contributions = computed(() => registry.list())
const activeId = registry.active
const activeViewId = registry.activeView
const activeViewVersion = registry.activeViewVersion
const workspaceReady = ref(false)
const sidebarExpanded = ref(true)
const windowActions = [
  { id: 'new-graph', label: '新建', title: '新建拓扑图', action: createGraph },
  { id: 'open-graph', label: '打开', title: '打开拓扑图配置文件', action: openGraph },
  { id: 'save-graph', label: '保存', title: '保存当前拓扑图', action: saveGraph },
  { id: 'save-as-graph', label: '另存为', title: '将当前拓扑图另存为文件', action: saveGraphAs },
]
const systemTheme = ref<'dark' | 'light'>(resolveSystemTheme())
let systemThemeMedia: MediaQueryList | null = null

const effectiveTheme = computed(() => {
  if (settings.global.Theme === 'system') {
    return systemTheme.value
  }
  return settings.global.Theme
})

const naiveTheme = computed(() => effectiveTheme.value === 'light' ? lightTheme : darkTheme)

const darkThemeOverrides: GlobalThemeOverrides = {
  common: {
    primaryColor: '#007acc',
    primaryColorHover: '#1a86d1',
    primaryColorPressed: '#005a9e',
    borderColor: '#2d2d2d',
    textColor1: '#d4d4d4',
    textColor2: '#cccccc',
    textColor3: '#858585',
    bodyColor: '#1e1e1e',
    popoverColor: '#252526',
    cardColor: '#252526',
    inputColor: 'rgba(255, 255, 255, 0.1)',
    dividerColor: '#2d2d2d',
  },
  Tabs: {
    tabTextColorActiveBar: '#d4d4d4',
    tabTextColorHoverBar: '#d4d4d4',
    tabColorSegment: '#252526',
    tabBorderColor: '#2d2d2d',
  },
}

const lightThemeOverrides: GlobalThemeOverrides = {
  common: {
    primaryColor: '#007acc',
    primaryColorHover: '#1a86d1',
    primaryColorPressed: '#005a9e',
    borderColor: '#d0d0d0',
    textColor1: '#1f2328',
    textColor2: '#333333',
    textColor3: '#666666',
    bodyColor: '#f5f5f5',
    popoverColor: '#ffffff',
    cardColor: '#ffffff',
    inputColor: '#ffffff',
    dividerColor: '#d0d0d0',
  },
  Tabs: {
    tabTextColorActiveBar: '#1f2328',
    tabTextColorHoverBar: '#1f2328',
    tabColorSegment: '#ffffff',
    tabBorderColor: '#d0d0d0',
  },
}

const themeOverrides = computed(() => effectiveTheme.value === 'light' ? lightThemeOverrides : darkThemeOverrides)

onMounted(async () => {
  systemThemeMedia = window.matchMedia?.('(prefers-color-scheme: light)') ?? null
  syncSystemTheme()
  if (systemThemeMedia) {
    addSystemThemeListener(systemThemeMedia)
  }
  try {
    const loaded = await workspaceFile.loadLast()
    if (!loaded) {
      workspaceFile.syncAllGraphSnapshots()
    }
  } catch {
    workspaceFile.syncAllGraphSnapshots()
  } finally {
    workspaceReady.value = true
  }
})

onBeforeUnmount(() => {
  if (systemThemeMedia) {
    removeSystemThemeListener(systemThemeMedia)
  }
  systemThemeMedia = null
})

watch(
  () => graphStore.exportState(),
  () => {
    if (!workspaceReady.value) return
    workspaceFile.syncAllGraphSnapshots()
  },
  { deep: true }
)

watch(
  () => settings.serial,
  () => {
    if (!workspaceReady.value || !graphStore.activeGraphId) return
    workspaceFile.syncGraphSnapshot(graphStore.activeGraphId, { useCurrentSerialSettings: true })
  },
  { deep: true }
)

watch(
  () => graphStore.activeGraphId,
  graphId => {
    if (!workspaceReady.value || !graphId) return
    workspaceFile.applyGraphSerialSettings(graphId)
  }
)

function resolveSystemTheme(): 'dark' | 'light' {
  if (typeof window === 'undefined') return 'dark'
  return window.matchMedia?.('(prefers-color-scheme: light)').matches ? 'light' : 'dark'
}

function syncSystemTheme() {
  systemTheme.value = resolveSystemTheme()
}

function handleSystemThemeChange(event: MediaQueryListEvent) {
  systemTheme.value = event.matches ? 'light' : 'dark'
}

function addSystemThemeListener(media: MediaQueryList) {
  if (typeof media.addEventListener === 'function') {
    media.addEventListener('change', handleSystemThemeChange)
  } else {
    media.addListener?.(handleSystemThemeChange)
  }
}

function removeSystemThemeListener(media: MediaQueryList) {
  if (typeof media.removeEventListener === 'function') {
    media.removeEventListener('change', handleSystemThemeChange)
  } else {
    media.removeListener?.(handleSystemThemeChange)
  }
}

function handleActivitySelect(id: string) {
  if (id === activeId.value) {
    sidebarExpanded.value = !sidebarExpanded.value
    return
  }
  registry.setActive(id)
  sidebarExpanded.value = true
}

function createGraph() {
  const graph = graphStore.createGraph(undefined, true)
  workspaceFile.applyGraphSerialSettings(graph.id)
}

async function openGraph() {
  const path = await workspaceFile.selectOpenPath()
  if (!path) return
  await workspaceFile.openFromPath(path)
}

async function saveGraph() {
  const graphId = graphStore.activeGraphId ?? graphStore.graphList[0]?.id
  if (!graphId) return
  await workspaceFile.saveGraph(graphId)
}

async function saveGraphAs() {
  const graphId = graphStore.activeGraphId ?? graphStore.graphList[0]?.id
  if (!graphId) return
  await workspaceFile.saveGraphAs(graphId)
}
</script>

<template>
  <NConfigProvider :theme="naiveTheme" :theme-overrides="themeOverrides">
    <div
      class="app-shell"
      :class="`app-shell--${effectiveTheme}`"
    >
      <div class="app-shell__titlebar" data-app-region="drag">
        <div class="app-shell__titlebar-actions" data-app-region="no-drag">
          <button
            v-for="action in windowActions"
            :key="action.id"
            type="button"
            class="app-shell__titlebar-action"
            :data-testid="`app-titlebar-${action.id}`"
            :title="action.title"
            @click="action.action"
          >
            {{ action.label }}
          </button>
        </div>
      </div>
      <div class="app-shell__body">
        <ActivityBar
          :contributions="contributions"
          :active-id="activeId"
          @select="handleActivitySelect"
        />
        <Sidebar
          v-if="sidebarExpanded"
          :contributions="contributions"
          :active-id="activeId"
          :active-view-id="activeViewId"
          @select-view="registry.setActiveView($event)"
        />
        <EditorGroups
          :active-view-id="activeViewId"
          :active-view-version="activeViewVersion"
        />
      </div>
      <Panel />
      <StatusBar :active-id="activeId" />
    </div>
  </NConfigProvider>
</template>

<style scoped>
.app-shell {
  display: flex;
  flex-direction: column;
  width: 100vw;
  height: 100vh;
  overflow: hidden;
  background: var(--app-bg);
  color: var(--app-text);
  padding-top: 0;
  box-sizing: border-box;
}
.app-shell__titlebar {
  height: 34px;
  flex: 0 0 34px;
  display: flex;
  align-items: center;
  justify-content: flex-start;
  padding-left: 86px;
  padding-right: 10px;
  background: var(--app-surface);
  border-bottom: 1px solid var(--app-border);
}
.app-shell__titlebar-actions {
  display: inline-flex;
  align-items: center;
  gap: 8px;
}
.app-shell__titlebar-action {
  height: 22px;
  padding: 0 10px;
  border: 1px solid var(--app-border);
  background: var(--app-hover-bg);
  color: var(--app-text);
  font: inherit;
  font-size: 12px;
  line-height: 20px;
  cursor: pointer;
}
.app-shell--dark {
  --app-bg: #1e1e1e;
  --app-surface: #252526;
  --app-border: #2d2d2d;
  --app-text: #d4d4d4;
  --app-text-muted: #858585;
  --app-active: #094771;
  --app-hover-bg: #2a2d2e;
  --app-table-header: #252526;
  --app-accent: #dcdcaa;
}
.app-shell--light {
  --app-bg: #f5f5f5;
  --app-surface: #ffffff;
  --app-border: #d0d0d0;
  --app-text: #1f2328;
  --app-text-muted: #666666;
  --app-active: #dbeafe;
  --app-hover-bg: #f0f3f6;
  --app-table-header: #f6f8fa;
  --app-accent: #8250df;
}
.app-shell__body {
  flex: 1;
  display: flex;
  min-height: 0;
  overflow: hidden;
}
</style>
