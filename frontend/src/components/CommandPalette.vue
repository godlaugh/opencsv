<template>
  <Teleport to="body">
    <div class="dialog-overlay" @click.self="emit('close')">
      <div class="palette">
        <div class="palette-search">
          <Search :size="14" class="palette-search-icon" />
          <input
            ref="inputRef"
            class="palette-input"
            v-model="query"
            placeholder="Search commands..."
            @keydown.escape="emit('close')"
            @keydown.arrow-down.prevent="moveDown"
            @keydown.arrow-up.prevent="moveUp"
            @keydown.enter.prevent="runSelected"
          />
          <span class="palette-esc">ESC</span>
        </div>
        <div class="palette-list" ref="listRef">
          <div
            v-for="(cmd, i) in filtered"
            :key="cmd.id"
            class="palette-item"
            :class="{ active: i === selectedIdx }"
            @click="runCommand(cmd)"
            @mouseenter="selectedIdx = i"
          >
            <component :is="cmd.icon" :size="14" class="palette-item-icon" />
            <div class="palette-item-content">
              <span class="palette-item-label">{{ cmd.label }}</span>
              <span v-if="cmd.desc" class="palette-item-desc">{{ cmd.desc }}</span>
            </div>
            <span v-if="cmd.shortcut" class="palette-item-shortcut">{{ cmd.shortcut }}</span>
          </div>
          <div v-if="filtered.length === 0" class="palette-empty">No commands found</div>
        </div>
      </div>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import {
  Search, FolderOpen, Save, Undo2, Redo2,
  ArrowUpDown, Filter, Terminal, FileText,
  FileSpreadsheet, Copy, Trash2, RefreshCw,
  Sun, Moon, Monitor, SplitSquareHorizontal,
  Type, Columns, Rows
} from 'lucide-vue-next'
import { useTabsStore } from '@/stores/tabs'
import { fileApi } from '@/api/file'

const emit = defineEmits<{ close: [] }>()

const tabsStore = useTabsStore()
const inputRef = ref<HTMLInputElement | null>(null)
const listRef = ref<HTMLElement | null>(null)
const query = ref('')
const selectedIdx = ref(0)

interface Command {
  id: string
  label: string
  desc?: string
  icon: any
  shortcut?: string
  action: () => void
}

const allCommands: Command[] = [
  { id: 'open', label: 'Open File', desc: 'Open a CSV or Excel file', icon: FolderOpen, shortcut: '⌘O', action: () => window.dispatchEvent(new CustomEvent('cmd:open')) },
  { id: 'save', label: 'Save File', icon: Save, shortcut: '⌘S', action: () => window.dispatchEvent(new CustomEvent('cmd:save')) },
  { id: 'close', label: 'Close Tab', icon: Trash2, shortcut: '⌘W', action: () => { const t = tabsStore.activeTab; if (t) tabsStore.removeTab(t.id) } },
  { id: 'find', label: 'Find & Replace', icon: Search, shortcut: '⌘F', action: () => window.dispatchEvent(new CustomEvent('cmd:findReplace')) },
  { id: 'sort', label: 'Sort Data', icon: ArrowUpDown, action: () => window.dispatchEvent(new CustomEvent('cmd:sort')) },
  { id: 'filter', label: 'Filter Data', icon: Filter, action: () => window.dispatchEvent(new CustomEvent('cmd:filter')) },
  { id: 'sql', label: 'Open SQL Console', icon: Terminal, action: () => window.dispatchEvent(new CustomEvent('cmd:sql')) },
  { id: 'undo', label: 'Undo', icon: Undo2, shortcut: '⌘Z', action: () => window.dispatchEvent(new CustomEvent('grid:undo')) },
  { id: 'redo', label: 'Redo', icon: Redo2, shortcut: '⌘⇧Z', action: () => window.dispatchEvent(new CustomEvent('grid:redo')) },
  { id: 'exportExcel', label: 'Export as Excel', icon: FileSpreadsheet, action: () => window.dispatchEvent(new CustomEvent('cmd:exportExcel')) },
  { id: 'exportMarkdown', label: 'Copy as Markdown', icon: Copy, action: () => window.dispatchEvent(new CustomEvent('cmd:copyMarkdown')) },
  { id: 'exportJson', label: 'Copy as JSON', icon: Copy, action: () => window.dispatchEvent(new CustomEvent('cmd:copyJson')) },
  { id: 'transpose', label: 'Transpose (Swap Rows/Columns)', icon: SplitSquareHorizontal, action: () => window.dispatchEvent(new CustomEvent('cmd:transpose')) },
  { id: 'transform', label: 'Transform Text...', icon: Type, action: () => window.dispatchEvent(new CustomEvent('cmd:transform')) },
  { id: 'light', label: 'Switch to Light Theme', icon: Sun, action: () => window.dispatchEvent(new CustomEvent('cmd:theme', { detail: 'light' })) },
  { id: 'dark', label: 'Switch to Dark Theme', icon: Moon, action: () => window.dispatchEvent(new CustomEvent('cmd:theme', { detail: 'dark' })) },
  { id: 'system', label: 'Use System Theme', icon: Monitor, action: () => window.dispatchEvent(new CustomEvent('cmd:theme', { detail: 'system' })) },
]

const filtered = computed(() => {
  const q = query.value.toLowerCase()
  if (!q) return allCommands
  return allCommands.filter(c =>
    c.label.toLowerCase().includes(q) ||
    c.desc?.toLowerCase().includes(q) ||
    c.id.includes(q)
  )
})

watch(filtered, () => { selectedIdx.value = 0 })

onMounted(() => inputRef.value?.focus())

function moveDown() {
  selectedIdx.value = Math.min(filtered.value.length - 1, selectedIdx.value + 1)
  scrollToSelected()
}
function moveUp() {
  selectedIdx.value = Math.max(0, selectedIdx.value - 1)
  scrollToSelected()
}
function scrollToSelected() {
  nextTick(() => {
    const el = listRef.value?.children[selectedIdx.value] as HTMLElement
    el?.scrollIntoView({ block: 'nearest' })
  })
}
function runSelected() {
  const cmd = filtered.value[selectedIdx.value]
  if (cmd) runCommand(cmd)
}
function runCommand(cmd: Command) {
  emit('close')
  cmd.action()
}

import { nextTick } from 'vue'
</script>

<style scoped>
.palette {
  width: 560px;
  background: var(--bg-surface);
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  box-shadow: var(--shadow-lg);
  overflow: hidden;
}
.palette-search {
  display: flex;
  align-items: center;
  padding: 10px 14px;
  border-bottom: 1px solid var(--border);
  gap: 8px;
}
.palette-search-icon { color: var(--text-muted); flex-shrink: 0; }
.palette-input {
  flex: 1;
  border: none;
  background: transparent;
  font-family: var(--font-sans);
  font-size: 14px;
  color: var(--text-primary);
  outline: none;
}
.palette-esc {
  font-size: 10px;
  background: var(--bg-surface-3);
  border: 1px solid var(--border);
  border-radius: 3px;
  padding: 1px 5px;
  color: var(--text-muted);
  font-family: var(--font-mono);
}
.palette-list { max-height: 340px; overflow-y: auto; padding: 4px; }
.palette-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 8px 10px;
  border-radius: var(--radius-sm);
  cursor: pointer;
  transition: background 0.1s;
}
.palette-item.active { background: var(--accent-light); }
.palette-item-icon { color: var(--text-muted); flex-shrink: 0; }
.palette-item.active .palette-item-icon { color: var(--accent); }
.palette-item-content { flex: 1; min-width: 0; }
.palette-item-label { font-size: 13px; color: var(--text-primary); }
.palette-item-desc { font-size: 11px; color: var(--text-muted); display: block; }
.palette-item-shortcut { font-size: 10px; color: var(--text-muted); font-family: var(--font-mono); }
.palette-empty { padding: 20px; text-align: center; color: var(--text-muted); font-size: 13px; }
</style>
