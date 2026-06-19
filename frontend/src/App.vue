<script setup lang="ts">
import { computed } from 'vue'
import { NConfigProvider, darkTheme, type GlobalThemeOverrides } from 'naive-ui'
import ActivityBar from './shell/ActivityBar.vue'
import Sidebar from './shell/Sidebar.vue'
import EditorGroups from './shell/EditorGroups.vue'
import Panel from './shell/Panel.vue'
import StatusBar from './shell/StatusBar.vue'
import { useRegistry } from './core/registry'

const registry = useRegistry()
const contributions = computed(() => registry.list())
const activeId = registry.active
const activeViewId = registry.activeView
const activeViewVersion = registry.activeViewVersion

const themeOverrides: GlobalThemeOverrides = {
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
</script>

<template>
  <NConfigProvider :theme="darkTheme" :theme-overrides="themeOverrides">
    <div class="app-shell">
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
  background: #1e1e1e;
  /* macOS 窗口控制按钮安全区域 */
  padding-top: 50px;
  box-sizing: border-box;
}
.app-shell__body {
  flex: 1;
  display: flex;
  min-height: 0;
  overflow: hidden;
}
</style>
