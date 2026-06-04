<template>
  <div class="toolbar">
    <!-- Logo -->
    <div class="logo">
      <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <rect x="3" y="3" width="18" height="18" rx="2"/>
        <line x1="3" y1="9" x2="21" y2="9"/>
        <line x1="3" y1="15" x2="21" y2="15"/>
        <line x1="9" y1="3" x2="9" y2="21"/>
      </svg>
      <span class="logo-text">OpenCSV</span>
    </div>

    <div class="divider-v" />

    <!-- File Operations -->
    <div class="toolbar-group">
      <button class="btn btn-ghost tooltip" data-tip="Open File (⌘O)" @click="openFile">
        <FolderOpen :size="14" />
        <span>Open</span>
      </button>
      <input ref="fileInput" type="file" accept=".csv,.tsv,.txt,.xlsx" style="display:none" @change="onFileInput" multiple />

      <button class="btn btn-ghost tooltip" data-tip="Save (⌘S)" :disabled="!activeTab" @click="saveFile">
        <Save :size="14" />
      </button>

      <button class="btn btn-ghost tooltip" data-tip="Save As" :disabled="!activeTab" @click="saveAs">
        <SaveAll :size="14" />
      </button>
    </div>

    <div class="divider-v" />

    <!-- Edit Operations -->
    <div class="toolbar-group">
      <button class="btn btn-ghost tooltip" data-tip="Undo (⌘Z)" :disabled="!canUndo" @click="undo">
        <Undo2 :size="14" />
      </button>
      <button class="btn btn-ghost tooltip" data-tip="Redo (⌘⇧Z)" :disabled="!canRedo" @click="redo">
        <Redo2 :size="14" />
      </button>
    </div>

    <div class="divider-v" />

    <!-- Data Operations -->
    <div class="toolbar-group">
      <button class="btn btn-ghost tooltip" data-tip="Find & Replace (⌘F)" :disabled="!activeTab" @click="emit('findReplace')">
        <Search :size="14" />
      </button>
      <button class="btn btn-ghost tooltip" data-tip="Sort" :disabled="!activeTab" @click="emit('sort')">
        <ArrowUpDown :size="14" />
      </button>
      <button class="btn btn-ghost tooltip" data-tip="Filter" :disabled="!activeTab" @click="emit('filter')">
        <Filter :size="14" />
      </button>
      <button class="btn btn-ghost tooltip" data-tip="SQL Console" :disabled="!activeTab" @click="emit('sql')">
        <Terminal :size="14" />
      </button>
    </div>

    <div class="divider-v" />

    <!-- Row/Col Operations -->
    <div class="toolbar-group">
      <button class="btn btn-ghost tooltip" data-tip="Insert Row Below" :disabled="!activeTab" @click="emit('insertRow')">
        <Plus :size="14" />
        R
      </button>
      <button class="btn btn-ghost tooltip" data-tip="Delete Selected Rows" :disabled="!activeTab" @click="emit('deleteRows')">
        <Trash2 :size="14" />
        R
      </button>
    </div>

    <div class="divider-v" />

    <!-- Export Operations -->
    <div class="toolbar-group export-group">
      <button class="btn btn-ghost tooltip" data-tip="Export as Excel (.xlsx)" :disabled="!activeTab" @click="exportExcel">
        <FileSpreadsheet :size="14" />
      </button>
      <div class="export-dropdown">
        <button class="btn btn-ghost tooltip" data-tip="Download as..." :disabled="!activeTab" @click="showExportMenu = !showExportMenu">
          <Download :size="14" />
        </button>
        <div v-if="showExportMenu" class="export-menu" @mouseleave="showExportMenu = false">
          <button @click="downloadFormat('json')">JSON</button>
          <button @click="downloadFormat('markdown')">Markdown</button>
          <button @click="downloadFormat('html')">HTML</button>
          <button @click="downloadFormat('sql')">SQL</button>
          <button @click="downloadFormat('latex')">LaTeX</button>
        </div>
      </div>
    </div>

    <div class="flex-1" />

    <!-- Theme Toggle -->
    <div class="toolbar-group">
      <button class="btn-icon tooltip" :data-tip="themeLabel" @click="cycleTheme">
        <Sun v-if="settings.theme === 'light'" :size="14" />
        <Moon v-else-if="settings.theme === 'dark'" :size="14" />
        <Monitor v-else :size="14" />
      </button>
    </div>

    <!-- Command Palette hint -->
    <button class="btn btn-ghost cmd-hint" @click="openCommandPalette">
      <Command :size="12" />
      <span>P</span>
    </button>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, inject, onMounted, onUnmounted } from 'vue'
import {
  FolderOpen, Save, SaveAll, Undo2, Redo2,
  Search, ArrowUpDown, Filter, Terminal,
  Plus, Trash2, Sun, Moon, Monitor, Command,
  FileSpreadsheet, Download
} from 'lucide-vue-next'
import { useTabsStore } from '@/stores/tabs'
import { useHistoryStore } from '@/stores/history'
import { useSettingsStore } from '@/stores/settings'
import { fileApi } from '@/api/file'
import { exportApi } from '@/api/data'
import { useFileOpener } from '@/composables/useFileOpener'
import { saveSession } from '@/utils/fileSystem'

const emit = defineEmits(['findReplace', 'sort', 'filter', 'sql', 'insertRow', 'deleteRows'])

const tabsStore = useTabsStore()
const historyStore = useHistoryStore()
const settings = useSettingsStore()
const notify = inject<(type: string, msg: string) => void>('notify')
const openCommandPalette = inject<() => void>('openCommandPalette')

const { fileInput, openFile, onFileInput } = useFileOpener()
const activeTab = computed(() => tabsStore.activeTab)
const showExportMenu = ref(false)

function onCmdOpen() { openFile() }
onMounted(() => window.addEventListener('cmd:open', onCmdOpen))
onUnmounted(() => window.removeEventListener('cmd:open', onCmdOpen))
const canUndo = computed(() => activeTab.value ? historyStore.canUndo(activeTab.value.id) : false)
const canRedo = computed(() => activeTab.value ? historyStore.canRedo(activeTab.value.id) : false)

const themeLabel = computed(() => ({
  light: 'Light Theme',
  dark: 'Dark Theme',
  system: 'System Theme'
}[settings.theme]))

function cycleTheme() {
  const order = ['light', 'dark', 'system'] as const
  const idx = order.indexOf(settings.theme)
  settings.setTheme(order[(idx + 1) % 3])
}

async function saveFile() {
  const tab = activeTab.value
  if (!tab) return
  try {
    const where = await saveSession(tab.session.id)
    tabsStore.markModified(tab.session.id, false)
    notify?.('success', where === 'disk' ? 'Saved to file' : 'File saved')
  } catch (err: any) {
    notify?.('error', err.message)
  }
}

async function saveAs() {
  const tab = activeTab.value
  if (!tab) return
  const path = prompt('Save as path:', tab.session.filePath)
  if (!path) return
  try {
    await fileApi.save(tab.session.id, path)
    tabsStore.markModified(tab.session.id, false)
    notify?.('success', 'Saved to ' + path)
  } catch (err: any) {
    notify?.('error', err.message)
  }
}

function undo() { window.dispatchEvent(new CustomEvent('grid:undo')) }
function redo() { window.dispatchEvent(new CustomEvent('grid:redo')) }

async function exportExcel() {
  const tab = activeTab.value
  if (!tab) return
  const defaultPath = tab.session.filePath.replace(/\.(csv|tsv|txt)$/i, '.xlsx')
  const path = prompt('Export as Excel to:', defaultPath)
  if (!path) return
  try {
    await exportApi.toExcel(tab.session.id, path)
    notify?.('success', 'Exported to ' + path)
  } catch (err: any) { notify?.('error', err.message) }
}

function downloadFormat(format: string) {
  showExportMenu.value = false
  window.dispatchEvent(new CustomEvent('cmd:downloadFormat', { detail: format }))
}
</script>

<style scoped>
.toolbar {
  height: var(--toolbar-h);
  background: var(--bg-surface);
  border-bottom: 1px solid var(--border);
  display: flex;
  align-items: center;
  padding: 0 8px;
  gap: 4px;
  flex-shrink: 0;
  user-select: none;
}
.logo {
  display: flex;
  align-items: center;
  gap: 7px;
  padding: 0 6px;
  color: var(--accent);
}
.logo-text {
  font-family: var(--font-mono);
  font-size: 13px;
  font-weight: 600;
  color: var(--text-primary);
  letter-spacing: -0.3px;
}
.toolbar-group {
  display: flex;
  align-items: center;
  gap: 2px;
}
.divider-v {
  width: 1px;
  height: 18px;
  background: var(--border);
  margin: 0 4px;
}
.cmd-hint {
  font-family: var(--font-mono);
  font-size: 11px;
  gap: 2px;
  padding: 3px 6px;
  color: var(--text-muted);
  border-color: var(--border);
}

.export-group { position: relative; }
.export-dropdown { position: relative; }
.export-menu {
  position: absolute;
  top: calc(100% + 4px);
  left: 0;
  background: var(--bg-surface);
  border: 1px solid var(--border);
  border-radius: var(--radius);
  box-shadow: var(--shadow-lg);
  z-index: 1000;
  min-width: 120px;
  padding: 4px;
}
.export-menu button {
  display: block;
  width: 100%;
  text-align: left;
  padding: 6px 10px;
  font-size: 12px;
  color: var(--text-primary);
  background: none;
  border: none;
  border-radius: var(--radius-sm);
  cursor: pointer;
}
.export-menu button:hover { background: var(--bg-hover); }
</style>
