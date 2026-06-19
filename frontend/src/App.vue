<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { NConfigProvider, darkTheme, lightTheme, type GlobalThemeOverrides } from 'naive-ui'
import ActivityBar from './shell/ActivityBar.vue'
import Sidebar from './shell/Sidebar.vue'
import EditorGroups from './shell/EditorGroups.vue'
import Panel from './shell/Panel.vue'
import StatusBar from './shell/StatusBar.vue'
import { useRegistry } from './core/registry'
import { useSettingsStore } from './settings'
import { useWorkspaceFileStore } from './workspace/stores/workspaceFileStore'
import { buildWorkspaceSnapshot } from './workspace/workspaceSession'

const registry = useRegistry()
const settings = useSettingsStore()
const workspaceFile = useWorkspaceFileStore()
const contributions = computed(() => registry.list())
const activeId = registry.active
const activeViewId = registry.activeView
const activeViewVersion = registry.activeViewVersion
const workspaceReady = ref(false)

const effectiveTheme = computed(() => {
  if (settings.global.Theme === 'system') {
    return window.matchMedia?.('(prefers-color-scheme: light)').matches ? 'light' : 'dark'
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
  try {
    const loaded = await workspaceFile.loadLast()
    if (!loaded) {
      workspaceFile.markClean('', buildWorkspaceSnapshot())
    }
  } catch {
    workspaceFile.markClean('', buildWorkspaceSnapshot())
  } finally {
    workspaceReady.value = true
  }
})

watch(
  () => buildWorkspaceSnapshot(),
  snapshot => {
    if (!workspaceReady.value) return
    workspaceFile.updateCurrentSnapshot(snapshot)
  },
  { deep: true }
)
</script>

<template>
  <NConfigProvider :theme="naiveTheme" :theme-overrides="themeOverrides">
    <div
      class="app-shell"
      :class="`app-shell--${effectiveTheme}`"
    >
      <div class="app-shell__body">
        <ActivityBar
          :contributions="contributions"
          :active-id="activeId"
          @select="registry.setActive($event)"
        />
        <Sidebar
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
      <StatusBar
        :active-id="activeId"
        :dirty="workspaceFile.isDirty"
        :config-path="workspaceFile.displayPath"
      />
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
  /* macOS 窗口控制按钮安全区域 */
  padding-top: 50px;
  box-sizing: border-box;
}
.app-shell--dark {
  --app-bg: #1e1e1e;
  --app-surface: #252526;
  --app-border: #2d2d2d;
  --app-text: #d4d4d4;
  --app-text-muted: #858585;
  --app-active: #094771;
}
.app-shell--light {
  --app-bg: #f5f5f5;
  --app-surface: #ffffff;
  --app-border: #d0d0d0;
  --app-text: #1f2328;
  --app-text-muted: #666666;
  --app-active: #dbeafe;
}
.app-shell__body {
  flex: 1;
  display: flex;
  min-height: 0;
  overflow: hidden;
}
</style>
