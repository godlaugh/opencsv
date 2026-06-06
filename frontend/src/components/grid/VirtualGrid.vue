<template>
  <div class="grid-wrap" @keydown="onKeydown" tabindex="0" ref="wrapRef">
    <!-- Overlays / Panels -->
    <FindReplaceBar v-if="showFindReplace" :tab="tab" @close="showFindReplace = false" @matches="onFindMatches" />
    <SortDialog v-if="showSort" :columns="tab.session.columns" :initial="tab.sortKeys" @close="showSort = false" @sort="onSort" />
    <FilterDialog v-if="showFilter" :columns="tab.session.columns" :initial="tab.filterGroup" @close="showFilter = false" @filter="onFilter" />
    <SqlConsole v-if="showSql" :tab="tab" @close="showSql = false" />
    <ContextMenu v-if="ctxMenu" :items="ctxItems" :x="ctxMenu.x" :y="ctxMenu.y" @close="ctxMenu = null" @select="onCtxSelect" />
    <TransformMenu v-if="showTransform" @close="showTransform = false" @apply="onTransform" />
    <ColumnFilterMenu
      v-if="colFilterMenu"
      :session-id="tab.session.id"
      :col-index="colFilterMenu.colIndex"
      :col-name="colFilterMenu.colName"
      :anchor="colFilterMenu.anchor"
      :initial="columnInitialFilter(colFilterMenu.colIndex)"
      :active-sort="columnSortKey(colFilterMenu.colIndex)"
      @close="colFilterMenu = null"
      @apply="applyColFilter"
      @sort="applyColSort"
    />

    <!-- Active filter (WHERE) bar — shared by per-column + global filter -->
    <div v-if="tab.filterActive && filterExpr" class="filter-bar">
      <span class="filter-bar-where">WHERE</span>
      <span class="filter-bar-expr truncate" :title="filterExpr">{{ filterExpr }}</span>
      <span class="filter-bar-count">{{ tab.filteredIndices?.length ?? 0 }} rows</span>
      <button class="filter-bar-btn" title="Edit filter" @click="openFilter">
        <Pencil :size="13" />
      </button>
      <button class="filter-bar-btn" title="Clear filter" @click="clearAllFilters">
        <X :size="14" />
      </button>
    </div>

    <!-- Fixed header: row-num gutter + scrollable column headers -->
    <div class="grid-header-outer">
      <div class="grid-header-rownum" :style="{ width: colNumW + 'px' }" />
      <div class="grid-header-scroll-wrap" style="overflow: hidden; flex: 1;">
        <div class="grid-header-row" ref="headerRef" :style="{ width: totalWidth + 'px' }">
          <div
            v-for="(col, ci) in visibleCols"
            :key="col.index"
            class="grid-header-cell"
            :class="{ selected: isColSelected(col.index), sorted: !!columnSort(col.index), editing: editingHeader === col.index }"
            :style="{ width: getColWidth(col.index) + 'px' }"
            @click="editingHeader === col.index ? null : selectCol(col.index, $event)"
            @dblclick="startHeaderEdit(ci, col)"
            @contextmenu.prevent="editingHeader === col.index ? null : openColCtx($event, col.index)"
          >
            <template v-if="editingHeader === col.index">
              <input
                class="header-editor"
                :value="headerEditValue"
                @input="headerEditValue = ($event.target as HTMLInputElement).value"
                @keydown.enter.prevent="commitHeaderEdit(col.index)"
                @keydown.escape.prevent="editingHeader = null"
                @blur="commitHeaderEdit(col.index)"
                @click.stop
                @dblclick.stop
                @mousedown.stop
              />
            </template>
            <template v-else>
              <span class="header-name truncate">{{ col.name }}</span>
              <span v-if="columnSort(col.index)" class="sort-indicator">
                {{ columnSort(col.index)!.order === 'asc' ? '↑' : '↓' }}<sub v-if="(tab.sortKeys?.length ?? 0) > 1">{{ columnSort(col.index)!.priority + 1 }}</sub>
              </span>
              <button
                class="header-filter-btn"
                :class="{ active: columnHasFilter(col.index) }"
                :title="columnHasFilter(col.index) ? 'Filter active — click to edit' : 'Sort & Filter'"
                @click.stop="openColFilter($event, col.index, col.name)"
                @mousedown.stop
                @dblclick.stop
              >
                <Filter :size="11" :fill="columnHasFilter(col.index) ? 'currentColor' : 'none'" />
              </button>
              <div class="col-resize-handle" @mousedown.stop="startColResize($event, col.index)" />
            </template>
          </div>
        </div>
      </div>
    </div>

    <!-- Scrollable body -->
    <div class="grid-body" ref="bodyRef" @scroll="onScroll">
      <!-- Row number gutter -->
      <div class="grid-row-nums" :style="{ width: colNumW + 'px', height: totalHeight + 'px' }">
        <div
          v-for="ri in visibleRowRange"
          :key="ri"
          class="grid-row-num"
          :class="{ selected: isRowSelected(ri) }"
          :style="{ top: (ri * rowH) + 'px', height: rowH + 'px' }"
          @click="selectRow(ri, $event)"
          @contextmenu.prevent="openRowCtx($event, ri)"
        >{{ ri + 1 }}</div>
      </div>

      <!-- Cell content area -->
      <div class="grid-content" :style="{ height: totalHeight + 'px', width: totalWidth + 'px', marginLeft: colNumW + 'px' }">
        <div
          v-for="ri in visibleRowRange"
          :key="ri"
          class="grid-row"
          :class="{ even: ri % 2 === 0 }"
          :style="{ top: (ri * rowH) + 'px', height: rowH + 'px', width: totalWidth + 'px' }"
        >
          <div
            v-for="col in visibleCols"
            :key="col.index"
            class="grid-cell"
            :class="{
              selected: isCellSelected(ri, col.index),
              active: isActiveCell(ri, col.index),
              'find-match': isFindMatch(ri, col.index)
            }"
            :style="{ width: getColWidth(col.index) + 'px' }"
            @mousedown="onCellMousedown($event, ri, col.index)"
            @mouseenter="onCellMouseenter($event, ri, col.index)"
            @dblclick="startEdit(ri, col.index)"
            @contextmenu.prevent="openCellCtx($event, ri, col.index)"
          >
            <template v-if="editingCell?.row === ri && editingCell?.col === col.index">
              <input
                ref="editInputRef"
                class="cell-editor"
                :value="editValue"
                @input="editValue = ($event.target as HTMLInputElement).value"
                @keydown.enter.prevent="commitEdit"
                @keydown.escape="cancelEdit"
                @keydown.tab.prevent="commitEditAndMove('right')"
                @blur="commitEdit"
              />
            </template>
            <template v-else>
              <span class="cell-value truncate">{{ getCellValue(ri, col.index) }}</span>
            </template>
          </div>
        </div>
        <!-- Selection highlight overlay -->
        <div v-if="selectionRect" class="selection-overlay" :style="selectionRect" />
      </div>
    </div>

    <!-- Loading spinner -->
    <div v-if="loading" class="grid-loading">
      <div class="spinner" />
      <span>Loading...</span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, watch, onMounted, onUnmounted, nextTick, inject } from 'vue'
import type { Tab, Cell, Selection, FindMatch, AggregateResult, ColumnQuickFilter, SortKey, FilterGroup } from '@/types'
import { fileApi } from '@/api/file'
import { dataApi, exportApi } from '@/api/data'
import { useTabsStore } from '@/stores/tabs'
import { useHistoryStore } from '@/stores/history'
import FindReplaceBar from '@/components/dialogs/FindReplaceBar.vue'
import SortDialog from '@/components/dialogs/SortDialog.vue'
import FilterDialog from '@/components/dialogs/FilterDialog.vue'
import SqlConsole from '@/components/panels/SqlConsole.vue'
import ContextMenu from '@/components/ContextMenu.vue'
import TransformMenu from '@/components/dialogs/TransformMenu.vue'
import ColumnFilterMenu from '@/components/grid/ColumnFilterMenu.vue'
import { Filter, Pencil, X } from 'lucide-vue-next'

const props = defineProps<{ tab: Tab }>()
const tabsStore = useTabsStore()
const historyStore = useHistoryStore()
const notify = inject<(type: string, msg: string) => void>('notify')

// DOM refs
const wrapRef = ref<HTMLElement | null>(null)
const bodyRef = ref<HTMLElement | null>(null)
const headerRef = ref<HTMLElement | null>(null)
const editInputRef = ref<HTMLInputElement[]>([])

// Dimensions
const rowH = 28
const colNumW = 52
const defaultColWidth = 120
const MIN_COL_WIDTH = 40

// Column widths (reactive map)
const colWidths = ref<Record<number, number>>({})

function getColWidth(idx: number) {
  return colWidths.value[idx] ?? defaultColWidth
}

// Virtual scroll state
const scrollTop = ref(0)
const scrollLeft = ref(0)
const viewportH = ref(600)
const viewportW = ref(1000)
const BUFFER = 5

// Loading / data
const loading = ref(false)

// --- Row storage: hybrid eager / windowed ---
// Small files (<= ROW_THRESHOLD rows): eager mode — localRows holds the whole
// dataset, everything is local and instant.
// Large files (> ROW_THRESHOLD): windowed mode — only a sliding window of rows
// lives in the browser (rowCache, keyed by full-dataset index), fetched on
// demand and evicted, so tab memory stays bounded regardless of file size.
const ROW_THRESHOLD = 200_000
const windowed = computed(() => props.tab.session.totalRows > ROW_THRESHOLD)

const localRows = ref<string[][]>(props.tab.rows)          // eager dense store
const rowCache = reactive(new Map<number, string[]>())     // windowed sparse store: actualIndex -> row

// Map a display-row (filter-aware) to its full-dataset index.
function actualIndex(displayRow: number): number {
  if (props.tab.filterActive && props.tab.filteredIndices) return props.tab.filteredIndices[displayRow]
  return displayRow
}

// Read/write a row by its full-dataset index, in whichever mode is active.
function rowByActual(actual: number): string[] | undefined {
  return windowed.value ? rowCache.get(actual) : localRows.value[actual]
}
function setRowByActual(actual: number, row: string[]) {
  if (windowed.value) rowCache.set(actual, row)
  else localRows.value[actual] = row
}

// Number of display rows in the current view (filter-aware, mode-aware).
const totalRows = computed(() => {
  if (props.tab.filterActive && props.tab.filteredIndices) return props.tab.filteredIndices.length
  return windowed.value ? props.tab.session.totalRows : localRows.value.length
})
const totalHeight = computed(() => totalRows.value * rowH)

const columns = computed(() => props.tab.session.columns)
const totalWidth = computed(() => columns.value.reduce((acc, col) => acc + getColWidth(col.index), 0))

// Visible range
const visibleRowRange = computed(() => {
  const start = Math.max(0, Math.floor(scrollTop.value / rowH) - BUFFER)
  const end = Math.min(totalRows.value - 1, Math.ceil((scrollTop.value + viewportH.value) / rowH) + BUFFER)
  const range = []
  for (let i = start; i <= end; i++) range.push(i)
  return range
})

const visibleCols = computed(() => {
  let x = 0
  const visible = []
  for (const col of columns.value) {
    const w = getColWidth(col.index)
    // show columns in the scroll viewport + 300px buffer on each side
    if (x + w > scrollLeft.value - 300 && x < scrollLeft.value + viewportW.value + 300) {
      visible.push(col)
    }
    x += w
  }
  return visible
})

// Selection state
const sel = ref<Selection>({ startRow: 0, startCol: 0, endRow: 0, endCol: 0 })
const activeCell = ref({ row: 0, col: 0 })
const isDragging = ref(false)

const selMinRow = computed(() => Math.min(sel.value.startRow, sel.value.endRow))
const selMaxRow = computed(() => Math.max(sel.value.startRow, sel.value.endRow))
const selMinCol = computed(() => Math.min(sel.value.startCol, sel.value.endCol))
const selMaxCol = computed(() => Math.max(sel.value.startCol, sel.value.endCol))

function isCellSelected(r: number, c: number) {
  return r >= selMinRow.value && r <= selMaxRow.value && c >= selMinCol.value && c <= selMaxCol.value
}
function isActiveCell(r: number, c: number) {
  return r === activeCell.value.row && c === activeCell.value.col
}
function isRowSelected(r: number) {
  return r >= selMinRow.value && r <= selMaxRow.value && selMinCol.value === 0 && selMaxCol.value >= columns.value.length - 1
}
function isColSelected(c: number) {
  return c >= selMinCol.value && c <= selMaxCol.value && selMinRow.value === 0 && selMaxRow.value >= totalRows.value - 1
}

// Selection visual rect
const selectionRect = computed(() => {
  if (selMinRow.value === selMaxRow.value && selMinCol.value === selMaxCol.value) return null
  const top = selMinRow.value * rowH
  const left = getColLeft(selMinCol.value)
  const width = Array.from({ length: selMaxCol.value - selMinCol.value + 1 }, (_, i) => getColWidth(selMinCol.value + i)).reduce((a, b) => a + b, 0)
  const height = (selMaxRow.value - selMinRow.value + 1) * rowH
  return { top: top + 'px', left: left + 'px', width: width + 'px', height: height + 'px' }
})

function getColLeft(colIdx: number) {
  let x = 0
  for (const col of columns.value) {
    if (col.index === colIdx) break
    x += getColWidth(col.index)
  }
  return x
}

// Find matches
// Find matches come back as full-dataset {row, col}. Map them to DISPLAY rows
// so highlight + navigation are correct under filter/sort/windowing.
const findMatches = ref<FindMatch[]>([])
const findCurrent = ref(0)

const findActualToDisplay = computed(() => {
  if (props.tab.filterActive && props.tab.filteredIndices) {
    const m = new Map<number, number>()
    props.tab.filteredIndices.forEach((ai, d) => m.set(ai, d))
    return m
  }
  return null // identity: display row == actual index
})

const findMatchKeys = computed(() => {
  const set = new Set<string>()
  const map = findActualToDisplay.value
  for (const m of findMatches.value) {
    const d = map ? map.get(m.row) : m.row
    if (d !== undefined && d !== null) set.add(d + ':' + m.col)
  }
  return set
})

function isFindMatch(r: number, c: number) {
  return findMatchKeys.value.has(r + ':' + c)
}

function onFindMatches(matches: FindMatch[], current = 0) {
  findMatches.value = matches
  findCurrent.value = current
  const m = matches[current]
  if (!m) return
  const map = findActualToDisplay.value
  const d = map ? map.get(m.row) : m.row
  if (d !== undefined && d !== null) scrollToCell(d, m.col)
}

// Sort indicator — derived from the shared tab.sortKeys
function columnSort(colIndex: number): { order: 'asc' | 'desc'; priority: number } | null {
  const keys = props.tab.sortKeys ?? []
  const idx = keys.findIndex(k => k.colIndex === colIndex)
  if (idx === -1) return null
  return { order: keys[idx].order, priority: idx }
}

// The full SortKey for a column (for the per-column menu to reflect direction + type)
function columnSortKey(colIndex: number): { order: 'asc' | 'desc'; type: 'text' | 'number' | 'date' | 'length' } | null {
  const k = (props.tab.sortKeys ?? []).find(k => k.colIndex === colIndex)
  return k ? { order: k.order, type: k.type } : null
}

// Editing state
const editingCell = ref<{ row: number; col: number } | null>(null)
const editValue = ref('')

function getCellValue(row: number, col: number): string {
  const r = rowByActual(actualIndex(row))
  if (!r) return ''
  return r[col] ?? ''
}

function startEdit(row: number, col: number) {
  editingCell.value = { row, col }
  editValue.value = getCellValue(row, col)
  nextTick(() => {
    const input = editInputRef.value?.[0]
    input?.focus()
    input?.select()
  })
}

function commitEdit() {
  if (!editingCell.value) return
  const { row, col } = editingCell.value
  const oldVal = getCellValue(row, col)
  const newVal = editValue.value

  if (oldVal !== newVal) {
    // Get actual row index (accounting for filter)
    const actualRow = props.tab.filterActive && props.tab.filteredIndices
      ? props.tab.filteredIndices[row]
      : row

    // Update local state
    const rowData = [...(rowByActual(actualRow) || [])]
    while (rowData.length <= col) rowData.push('')
    rowData[col] = newVal
    setRowByActual(actualRow, rowData)

    // Push to undo history
    historyStore.push(props.tab.id, {
      type: 'cell',
      before: { cells: [{ row: actualRow, col, value: oldVal }] },
      after: { cells: [{ row: actualRow, col, value: newVal }] }
    })

    // Sync to server (debounced)
    debouncedSync([{ row: actualRow, col, value: newVal }])
    tabsStore.markModified(props.tab.id, true)
  }
  editingCell.value = null
}

function commitEditAndMove(dir: 'right' | 'down') {
  commitEdit()
  if (dir === 'right') moveActive(0, 1)
  else moveActive(1, 0)
  nextTick(() => startEdit(activeCell.value.row, activeCell.value.col))
}

function cancelEdit() {
  editingCell.value = null
}

// Debounced server sync
let syncTimer: ReturnType<typeof setTimeout> | null = null
const pendingCells: Map<string, Cell> = new Map()

function debouncedSync(cells: Cell[]) {
  for (const cell of cells) {
    pendingCells.set(`${cell.row}:${cell.col}`, cell)
  }
  if (syncTimer) clearTimeout(syncTimer)
  syncTimer = setTimeout(async () => {
    const toSync = Array.from(pendingCells.values())
    pendingCells.clear()
    try {
      await fileApi.updateCells(props.tab.session.id, toSync)
    } catch {}
  }, 500)
}

// Keyboard navigation
function onKeydown(e: KeyboardEvent) {
  if (editingCell.value) return

  const meta = e.metaKey || e.ctrlKey

  if (meta && e.key === 'z' && !e.shiftKey) { e.preventDefault(); undoAction(); return }
  if (meta && e.key === 'z' && e.shiftKey) { e.preventDefault(); redoAction(); return }
  if (meta && e.key === 'y') { e.preventDefault(); redoAction(); return }
  if (meta && e.key === 'c') { e.preventDefault(); copySelection(); return }
  if (meta && e.key === 'v') { e.preventDefault(); pasteFromClipboard(); return }
  if (meta && e.key === 'a') { e.preventDefault(); selectAll(); return }
  if (meta && e.key === 'f') { e.preventDefault(); showFindReplace.value = true; return }

  if (e.key === 'Delete' || e.key === 'Backspace') {
    e.preventDefault()
    clearSelection()
    return
  }

  if (e.key === 'F2') { e.preventDefault(); startEdit(activeCell.value.row, activeCell.value.col); return }
  if (e.key === 'Enter') { e.preventDefault(); startEdit(activeCell.value.row, activeCell.value.col); return }

  const moves: Record<string, [number, number]> = {
    ArrowUp: [-1, 0], ArrowDown: [1, 0], ArrowLeft: [0, -1], ArrowRight: [0, 1],
    Tab: [0, 1], Home: [0, -activeCell.value.col], End: [0, columns.value.length - 1 - activeCell.value.col]
  }
  if (moves[e.key]) {
    e.preventDefault()
    const [dr, dc] = moves[e.key]
    if (e.shiftKey) {
      extendSelection(dr, dc)
    } else {
      moveActive(dr, dc)
      sel.value = { startRow: activeCell.value.row, startCol: activeCell.value.col, endRow: activeCell.value.row, endCol: activeCell.value.col }
    }
    return
  }

  // Start typing to edit
  if (e.key.length === 1 && !meta && !e.ctrlKey) {
    editingCell.value = { row: activeCell.value.row, col: activeCell.value.col }
    editValue.value = e.key
    nextTick(() => {
      const input = editInputRef.value?.[0]
      if (input) {
        input.focus()
        const len = input.value.length
        input.setSelectionRange(len, len)
      }
    })
  }
}

function moveActive(dr: number, dc: number) {
  activeCell.value = {
    row: Math.max(0, Math.min(totalRows.value - 1, activeCell.value.row + dr)),
    col: Math.max(0, Math.min(columns.value.length - 1, activeCell.value.col + dc))
  }
  scrollToCell(activeCell.value.row, activeCell.value.col)
}

function extendSelection(dr: number, dc: number) {
  sel.value = {
    ...sel.value,
    endRow: Math.max(0, Math.min(totalRows.value - 1, sel.value.endRow + dr)),
    endCol: Math.max(0, Math.min(columns.value.length - 1, sel.value.endCol + dc))
  }
}

function selectAll() {
  sel.value = { startRow: 0, startCol: 0, endRow: totalRows.value - 1, endCol: columns.value.length - 1 }
}

function selectRow(r: number, e: MouseEvent) {
  if (e.shiftKey) {
    sel.value = { startRow: sel.value.startRow, startCol: 0, endRow: r, endCol: columns.value.length - 1 }
  } else {
    sel.value = { startRow: r, startCol: 0, endRow: r, endCol: columns.value.length - 1 }
    activeCell.value = { row: r, col: 0 }
  }
}

function selectCol(c: number, e: MouseEvent) {
  if (e.shiftKey) {
    sel.value = { startRow: 0, startCol: sel.value.startCol, endRow: totalRows.value - 1, endCol: c }
  } else {
    sel.value = { startRow: 0, startCol: c, endRow: totalRows.value - 1, endCol: c }
    activeCell.value = { row: 0, col: c }
  }
}

// Mouse selection
function onCellMousedown(e: MouseEvent, r: number, c: number) {
  if (e.button !== 0) return
  activeCell.value = { row: r, col: c }
  if (e.shiftKey) {
    sel.value = { ...sel.value, endRow: r, endCol: c }
  } else {
    sel.value = { startRow: r, startCol: c, endRow: r, endCol: c }
  }
  isDragging.value = true
  wrapRef.value?.focus()
}

function onCellMouseenter(e: MouseEvent, r: number, c: number) {
  if (isDragging.value && e.buttons === 1) {
    sel.value = { ...sel.value, endRow: r, endCol: c }
  }
}

onMounted(() => {
  document.addEventListener('mouseup', () => { isDragging.value = false })
})

// Clipboard
async function copySelection() {
  // In windowed mode the selection may span rows not currently cached; fetch
  // them first so we don't copy blanks.
  if (windowed.value) {
    const idxs: number[] = []
    for (let r = selMinRow.value; r <= selMaxRow.value; r++) idxs.push(actualIndex(r))
    await ensureActualRowsLoaded(idxs)
  }
  const rows = []
  for (let r = selMinRow.value; r <= selMaxRow.value; r++) {
    const row = []
    for (let c = selMinCol.value; c <= selMaxCol.value; c++) {
      row.push(getCellValue(r, c))
    }
    rows.push(row.join('\t'))
  }
  await navigator.clipboard.writeText(rows.join('\n'))
  notify?.('success', `Copied ${(selMaxRow.value - selMinRow.value + 1) * (selMaxCol.value - selMinCol.value + 1)} cells`)
}

async function pasteFromClipboard() {
  const text = await navigator.clipboard.readText()
  const pastedRows = text.split('\n').map(line => line.split('\t'))
  const cells: Cell[] = []
  const before: Cell[] = []

  for (let dr = 0; dr < pastedRows.length; dr++) {
    for (let dc = 0; dc < pastedRows[dr].length; dc++) {
      const r = activeCell.value.row + dr
      const c = activeCell.value.col + dc
      if (r >= totalRows.value || c >= columns.value.length) continue
      const actualRow = props.tab.filterActive && props.tab.filteredIndices
        ? props.tab.filteredIndices[r] : r
      before.push({ row: actualRow, col: c, value: getCellValue(r, c) })
      cells.push({ row: actualRow, col: c, value: pastedRows[dr][dc] })
      const rowData = [...(rowByActual(actualRow) || [])]
      while (rowData.length <= c) rowData.push('')
      rowData[c] = pastedRows[dr][dc]
      setRowByActual(actualRow, rowData)
    }
  }

  if (cells.length > 0) {
    historyStore.push(props.tab.id, { type: 'paste', before: { cells: before }, after: { cells } })
    debouncedSync(cells)
    tabsStore.markModified(props.tab.id, true)
  }
}

function clearSelection() {
  const cells: Cell[] = []
  const before: Cell[] = []

  for (let r = selMinRow.value; r <= selMaxRow.value; r++) {
    for (let c = selMinCol.value; c <= selMaxCol.value; c++) {
      const actualRow = props.tab.filterActive && props.tab.filteredIndices
        ? props.tab.filteredIndices[r] : r
      before.push({ row: actualRow, col: c, value: getCellValue(r, c) })
      cells.push({ row: actualRow, col: c, value: '' })
      const rowData = [...(rowByActual(actualRow) || [])]
      rowData[c] = ''
      setRowByActual(actualRow, rowData)
    }
  }

  historyStore.push(props.tab.id, { type: 'clear', before: { cells: before }, after: { cells } })
  debouncedSync(cells)
  tabsStore.markModified(props.tab.id, true)
}

// Undo / Redo
// Single coherent timeline: the frontend history stack orders all actions.
// Cell edits are reverted locally; structural ops are reverted on the backend
// (inverse stack) and the grid then refreshes.
function pushStructural() {
  historyStore.push(props.tab.id, { type: 'structural', structural: true })
}

// Refresh the grid after the backend changed the data (undo/redo of a
// structural op): pick up new columns/totalRows, re-run any active filter,
// and reload the visible rows.
async function applyServerEdit(res: { columns?: any[]; totalRows?: number }) {
  if (res.columns) props.tab.session.columns = res.columns
  if (typeof res.totalRows === 'number') props.tab.session.totalRows = res.totalRows
  if (props.tab.filterActive && props.tab.filterGroup && props.tab.filterGroup.conditions.length > 0) {
    const fr = await dataApi.filter(props.tab.session.id, props.tab.filterGroup)
    props.tab.filteredIndices = fr.matchIndices
  }
  if (windowed.value) {
    rowCache.clear()
    await refetchVisible()
  } else {
    localRows.value = []
    await ensureAllRowsLoaded()
  }
}

async function undoAction() {
  const entry = historyStore.undo(props.tab.id)
  if (!entry) return
  if (entry.structural) {
    try {
      const res = await fileApi.undo(props.tab.session.id)
      await applyServerEdit(res)
      tabsStore.markModified(props.tab.id, true)
    } catch (err: any) { notify?.('error', err.message) }
    return
  }
  for (const cell of entry.before!.cells) {
    const rowData = [...(rowByActual(cell.row) || [])]
    while (rowData.length <= cell.col) rowData.push('')
    rowData[cell.col] = cell.value
    setRowByActual(cell.row, rowData)
  }
  debouncedSync(entry.before!.cells)
}

async function redoAction() {
  const entry = historyStore.redo(props.tab.id)
  if (!entry) return
  if (entry.structural) {
    try {
      const res = await fileApi.redo(props.tab.session.id)
      await applyServerEdit(res)
      tabsStore.markModified(props.tab.id, true)
    } catch (err: any) { notify?.('error', err.message) }
    return
  }
  for (const cell of entry.after!.cells) {
    const rowData = [...(rowByActual(cell.row) || [])]
    while (rowData.length <= cell.col) rowData.push('')
    rowData[cell.col] = cell.value
    setRowByActual(cell.row, rowData)
  }
  debouncedSync(entry.after!.cells)
}

// Scroll — sync header horizontal scroll with body
function onScroll(e: Event) {
  const el = e.target as HTMLElement
  scrollTop.value = el.scrollTop
  scrollLeft.value = el.scrollLeft
  if (headerRef.value) {
    headerRef.value.style.transform = `translateX(-${el.scrollLeft}px)`
  }
}

function scrollToCell(row: number, col: number) {
  const el = bodyRef.value
  if (!el) return
  const top = row * rowH
  const left = getColLeft(col)
  const colW = getColWidth(col)

  if (top < el.scrollTop) el.scrollTop = top
  else if (top + rowH > el.scrollTop + viewportH.value) el.scrollTop = top + rowH - viewportH.value

  if (left < el.scrollLeft) el.scrollLeft = left
  else if (left + colW > el.scrollLeft + viewportW.value - colNumW) el.scrollLeft = left + colW - viewportW.value + colNumW
}

// Header inline editing
const editingHeader = ref<number | null>(null)
const headerEditValue = ref('')

function startHeaderEdit(_ci: number, col: { index: number; name: string }) {
  editingHeader.value = col.index
  headerEditValue.value = col.name
  nextTick(() => {
    const input = wrapRef.value?.querySelector('.header-editor') as HTMLInputElement | null
    input?.focus()
    input?.select()
  })
}

async function commitHeaderEdit(colIdx: number) {
  if (editingHeader.value === null) return
  editingHeader.value = null
  const col = columns.value.find(c => c.index === colIdx)
  if (!col || col.name === headerEditValue.value) return
  const newCols = columns.value.map(c =>
    c.index === colIdx ? { ...c, name: headerEditValue.value } : c
  )
  props.tab.session.columns = newCols
  try {
    await fileApi.updateColumns(props.tab.session.id, newCols)
    pushStructural()
    tabsStore.markModified(props.tab.id, true)
  } catch {}
}

// Column resize
let resizingCol: number | null = null
let resizeStartX = 0
let resizeStartW = 0

function startColResize(e: MouseEvent, colIdx: number) {
  resizingCol = colIdx
  resizeStartX = e.clientX
  resizeStartW = getColWidth(colIdx)
  document.addEventListener('mousemove', onColResizeMove)
  document.addEventListener('mouseup', onColResizeUp)
}

function onColResizeMove(e: MouseEvent) {
  if (resizingCol === null) return
  const newW = Math.max(MIN_COL_WIDTH, resizeStartW + (e.clientX - resizeStartX))
  colWidths.value[resizingCol] = newW
}

function onColResizeUp() {
  resizingCol = null
  document.removeEventListener('mousemove', onColResizeMove)
  document.removeEventListener('mouseup', onColResizeUp)
}

// Viewport resize observer
let resizeObserver: ResizeObserver | null = null
onMounted(() => {
  const el = bodyRef.value
  if (!el) return
  resizeObserver = new ResizeObserver(entries => {
    for (const entry of entries) {
      viewportH.value = entry.contentRect.height
      viewportW.value = entry.contentRect.width
    }
  })
  resizeObserver.observe(el)
  wrapRef.value?.focus()

  // Windowed mode: seed the cache with the initial rows the tab opened with,
  // then load whatever is actually visible.
  if (windowed.value) {
    props.tab.rows.forEach((r, i) => rowCache.set(i, r))
    refetchVisible()
  }

  // Grid action events from toolbar
  window.addEventListener('grid:undo', undoAction)
  window.addEventListener('grid:redo', redoAction)

  // Command palette events
  window.addEventListener('cmd:findReplace', _cmdFindReplace)
  window.addEventListener('cmd:sort', _cmdSort)
  window.addEventListener('cmd:filter', _cmdFilter)
  window.addEventListener('cmd:sql', _cmdSql)
  window.addEventListener('cmd:transform', _cmdTransform)
  window.addEventListener('cmd:transpose', _cmdTranspose)
  window.addEventListener('cmd:copyMarkdown', _cmdCopyMarkdown)
  window.addEventListener('cmd:copyJson', _cmdCopyJson)
  window.addEventListener('cmd:exportExcel', _cmdExportExcel)
  window.addEventListener('cmd:downloadFormat', _cmdDownloadFormat)
})

onUnmounted(() => {
  resizeObserver?.disconnect()
  window.removeEventListener('grid:undo', undoAction)
  window.removeEventListener('grid:redo', redoAction)
  window.removeEventListener('cmd:findReplace', _cmdFindReplace)
  window.removeEventListener('cmd:sort', _cmdSort)
  window.removeEventListener('cmd:filter', _cmdFilter)
  window.removeEventListener('cmd:sql', _cmdSql)
  window.removeEventListener('cmd:transform', _cmdTransform)
  window.removeEventListener('cmd:transpose', _cmdTranspose)
  window.removeEventListener('cmd:copyMarkdown', _cmdCopyMarkdown)
  window.removeEventListener('cmd:copyJson', _cmdCopyJson)
  window.removeEventListener('cmd:exportExcel', _cmdExportExcel)
  window.removeEventListener('cmd:downloadFormat', _cmdDownloadFormat)
})

// Sort — shared runner used by both the global dialog and per-column buttons.
// tab.sortKeys is the single source of truth.
async function runSort(keys: SortKey[]) {
  loading.value = true
  try {
    props.tab.sortKeys = keys
    if (keys.length > 0) {
      await dataApi.sort(props.tab.session.id, keys)
      pushStructural()
    }
    if (windowed.value) {
      // Windowed: cached rows are keyed by index whose data just changed → drop.
      rowCache.clear()
      // Re-run the filter against the new order so indices line up.
      if (props.tab.filterActive && props.tab.filterGroup && props.tab.filterGroup.conditions.length > 0) {
        const result = await dataApi.filter(props.tab.session.id, props.tab.filterGroup)
        props.tab.filteredIndices = result.matchIndices
      }
      await refetchVisible()
    } else {
      // Eager: reload the whole dataset in the new order.
      localRows.value = []
      await ensureAllRowsLoaded()
      if (props.tab.filterActive && props.tab.filterGroup && props.tab.filterGroup.conditions.length > 0) {
        const result = await dataApi.filter(props.tab.session.id, props.tab.filterGroup)
        props.tab.filteredIndices = result.matchIndices
      }
    }
    notify?.('success', keys.length > 1 ? `Sorted by ${keys.length} columns` : 'Sorted')
  } catch (err: any) {
    notify?.('error', err.message)
  } finally {
    loading.value = false
  }
}

// Global Sort dialog → replace the shared sort keys.
async function onSort(keys: SortKey[]) {
  showSort.value = false
  await runSort(keys)
}

// Global filter dialog → set the shared filter group, then run.
async function onFilter(group: FilterGroup) {
  showFilter.value = false
  props.tab.filterGroup = group && group.conditions.length > 0 ? group : null
  await applyFilterGroup()
}

// Ensure the full source dataset is loaded into localRows. Required before
// filtering/sorting, because filteredIndices reference the complete dataset
// but rows are otherwise lazy-loaded in chunks. Paginates because the backend
// caps /rows at 5000 per request.
async function ensureAllRowsLoaded() {
  const total = props.tab.session.totalRows
  if (localRows.value.length >= total) return
  loading.value = true
  try {
    const all: string[][] = []
    let offset = 0
    const CHUNK = 5000
    while (offset < total) {
      const { rows } = await fileApi.getRows(props.tab.session.id, offset, CHUNK)
      if (!rows.length) break
      for (const r of rows) all.push(r)
      offset += rows.length
    }
    localRows.value = all
    tabsStore.updateTabRows(props.tab.id, all)
  } finally {
    loading.value = false
  }
}

// ---- Windowed mode: fetch only the visible window of rows on demand ----
const EVICT_KEEP = 3000 // display rows of slack to keep cached around the viewport
let winSeq = 0

// Ensure rows for display range [start, end] are present in rowCache.
async function ensureWindow(start: number, end: number) {
  if (!windowed.value) return
  const need: number[] = []
  for (let d = start; d <= end; d++) {
    if (d < 0 || d >= totalRows.value) continue
    const ai = actualIndex(d)
    if (ai == null || ai < 0) continue
    if (!rowCache.has(ai)) need.push(ai)
  }
  if (need.length === 0) return
  const seq = ++winSeq
  try {
    if (props.tab.filterActive && props.tab.filteredIndices) {
      // Scattered indices → fetch by explicit list
      const { rows } = await fileApi.getRowsByIndices(props.tab.session.id, need)
      if (seq !== winSeq) return
      need.forEach((ai, k) => { if (rows[k]) rowCache.set(ai, rows[k]) })
    } else {
      // Contiguous range → cheaper offset/limit fetch
      const offset = need[0]
      const limit = need[need.length - 1] - need[0] + 1
      const { rows } = await fileApi.getRows(props.tab.session.id, offset, limit)
      if (seq !== winSeq) return
      rows.forEach((r, k) => rowCache.set(offset + k, r))
    }
    evictFar(start, end)
  } catch { /* leave gaps; they render blank and refetch on next scroll */ }
}

// Drop cached rows far from the current viewport to keep memory bounded.
function evictFar(start: number, end: number) {
  if (rowCache.size <= EVICT_KEEP * 3) return
  const keep = new Set<number>()
  const lo = Math.max(0, start - EVICT_KEEP)
  const hi = Math.min(totalRows.value - 1, end + EVICT_KEEP)
  for (let d = lo; d <= hi; d++) keep.add(actualIndex(d))
  for (const k of rowCache.keys()) if (!keep.has(k)) rowCache.delete(k)
}

// Ensure specific full-dataset rows are in the cache (windowed mode). Used by
// operations that read row data outside the visible window, e.g. copying a
// large selection. Fetches by explicit index in chunks.
async function ensureActualRowsLoaded(indices: number[]) {
  if (!windowed.value) return
  const missing: number[] = []
  for (const i of indices) {
    if (i != null && i >= 0 && !rowCache.has(i)) missing.push(i)
  }
  const CHUNK = 5000
  for (let s = 0; s < missing.length; s += CHUNK) {
    const batch = missing.slice(s, s + CHUNK)
    const { rows } = await fileApi.getRowsByIndices(props.tab.session.id, batch)
    batch.forEach((ai, k) => { if (rows[k]) rowCache.set(ai, rows[k]) })
  }
}

// Fetch the rows currently in view (used after filter/sort reset the view).
async function refetchVisible() {
  const start = Math.max(0, Math.floor(scrollTop.value / rowH) - BUFFER)
  const end = Math.min(totalRows.value - 1, Math.ceil((scrollTop.value + viewportH.value) / rowH) + BUFFER)
  await ensureWindow(start, end)
}

// After a structural change (insert/delete/transpose/dedup/transform) in
// windowed mode the cache is stale (indices shifted / data changed) — drop it
// and refetch the visible window.
async function refreshAfterStructuralChange() {
  if (!windowed.value) return
  rowCache.clear()
  await refetchVisible()
}

// ---- Per-column quick filter (Excel-style header dropdown) ----
const colFilterMenu = ref<{
  colIndex: number
  colName: string
  anchor: { x: number; y: number; width: number }
} | null>(null)

function openColFilter(e: MouseEvent, colIndex: number, colName: string) {
  const btn = e.currentTarget as HTMLElement
  const rect = btn.getBoundingClientRect()
  colFilterMenu.value = {
    colIndex,
    colName,
    anchor: { x: rect.right - 380, y: rect.bottom + 6, width: rect.width }
  }
}

// --- Unified filter state (shared by per-column menu + global dialog) ---

// Does the active filter group reference this column?
function columnHasFilter(colIndex: number): boolean {
  const g = props.tab.filterGroup
  return !!g && g.conditions.some(c => c.colIndex === colIndex)
}

function colName(colIndex: number): string {
  return columns.value.find(c => c.index === colIndex)?.name ?? `Col ${colIndex + 1}`
}

// Human-readable, SQL-like WHERE expression for the filter bar.
const filterExpr = computed(() => {
  const g = props.tab.filterGroup
  if (!g || g.conditions.length === 0) return ''
  const quote = (v: string) => `'${v.replace(/'/g, "''")}'`
  const parts = g.conditions.map(c => {
    const name = colName(c.colIndex)
    switch (c.operator) {
      case 'in': return `${name} IN (${(c.values ?? []).map(quote).join(', ')})`
      case 'notIn': return `${name} NOT IN (${(c.values ?? []).map(quote).join(', ')})`
      case 'eq': return `${name} = ${quote(c.value)}`
      case 'ne': return `${name} != ${quote(c.value)}`
      case 'contains': return `${name} contains ${quote(c.value)}`
      case 'notContains': return `${name} not contains ${quote(c.value)}`
      case 'startsWith': return `${name} starts with ${quote(c.value)}`
      case 'notStartsWith': return `${name} not starts with ${quote(c.value)}`
      case 'endsWith': return `${name} ends with ${quote(c.value)}`
      case 'notEndsWith': return `${name} not ends with ${quote(c.value)}`
      case 'like': return `${name} LIKE ${quote(c.value)}`
      case 'notLike': return `${name} NOT LIKE ${quote(c.value)}`
      case 'gt': return `${name} > ${c.value}`
      case 'gte': return `${name} >= ${c.value}`
      case 'lt': return `${name} < ${c.value}`
      case 'lte': return `${name} <= ${c.value}`
      case 'between': return `${name} BETWEEN ${quote(c.values?.[0] ?? '')} AND ${quote(c.values?.[1] ?? '')}`
      case 'notBetween': return `${name} NOT BETWEEN ${quote(c.values?.[0] ?? '')} AND ${quote(c.values?.[1] ?? '')}`
      case 'empty': return `${name} is empty`
      case 'notEmpty': return `${name} is not empty`
      case 'regex': return `${name} matches /${c.value}/`
      default: return `${name} ${c.operator} ${quote(c.value)}`
    }
  })
  return parts.join(g.logic === 'OR' ? ' OR ' : ' AND ')
})

function clearAllFilters() {
  props.tab.filterGroup = null
  props.tab.filterActive = false
  props.tab.filteredIndices = null
  if (windowed.value) refetchVisible()
  notify?.('info', 'Filter cleared')
}

// Convert the per-column condition (if any) into the shape ColumnFilterMenu wants.
function columnInitialFilter(colIndex: number): ColumnQuickFilter | undefined {
  const g = props.tab.filterGroup
  if (!g) return undefined
  const cond = g.conditions.find(c => c.colIndex === colIndex)
  if (!cond) return undefined
  if (cond.operator === 'in') {
    return { mode: 'values', selectedValues: cond.values ?? [] }
  }
  if (cond.operator === 'between' || cond.operator === 'notBetween') {
    return { mode: 'condition', operator: cond.operator, value: cond.values?.[0] ?? '', value2: cond.values?.[1] ?? '' }
  }
  return { mode: 'condition', operator: cond.operator, value: cond.value }
}

// Re-run whatever is currently in tab.filterGroup against the backend.
async function applyFilterGroup() {
  const group = props.tab.filterGroup
  if (!group || group.conditions.length === 0) {
    props.tab.filterGroup = null
    props.tab.filterActive = false
    props.tab.filteredIndices = null
    if (windowed.value) refetchVisible()
    notify?.('info', 'Filter cleared')
    return
  }
  try {
    loading.value = true
    // Eager mode maps filteredIndices into a fully-loaded localRows, so it must
    // load everything first. Windowed mode fetches matching rows on demand.
    if (!windowed.value) await ensureAllRowsLoaded()
    const result = await dataApi.filter(props.tab.session.id, group)
    props.tab.filterActive = true
    props.tab.filteredIndices = result.matchIndices
    if (windowed.value) {
      rowCache.clear()
      scrollTop.value = 0
      if (bodyRef.value) bodyRef.value.scrollTop = 0
      await refetchVisible()
    }
    notify?.('info', `${result.matchCount} rows match`)
  } catch (err: any) {
    notify?.('error', err.message)
  } finally {
    loading.value = false
  }
}

// Per-column menu applies/clears the condition for one column, then re-runs.
async function applyColFilter(filter: ColumnQuickFilter | null) {
  const colIndex = colFilterMenu.value?.colIndex
  if (colIndex === undefined) return

  const group: FilterGroup = props.tab.filterGroup
    ? { logic: props.tab.filterGroup.logic, conditions: [...props.tab.filterGroup.conditions] }
    : { logic: 'AND', conditions: [] }

  // Drop existing conditions for this column
  group.conditions = group.conditions.filter(c => c.colIndex !== colIndex)

  // Add the new condition for this column (if any)
  if (filter) {
    if (filter.mode === 'values' && filter.selectedValues && filter.selectedValues.length > 0) {
      group.conditions.push({ colIndex, operator: 'in', value: '', values: filter.selectedValues })
    } else if (filter.mode === 'condition' && filter.operator) {
      if (filter.operator === 'between' || filter.operator === 'notBetween') {
        group.conditions.push({ colIndex, operator: filter.operator, value: '', values: [filter.value ?? '', filter.value2 ?? ''] })
      } else {
        group.conditions.push({ colIndex, operator: filter.operator, value: filter.value ?? '' })
      }
    }
  }

  props.tab.filterGroup = group.conditions.length > 0 ? group : null
  await applyFilterGroup()
}

// Per-column sort button → replace the shared sort with this single key.
async function applyColSort(key: SortKey) {
  await runSort([key])
}

// Transform
async function onTransform(transformType: string) {
  showTransform.value = false
  const cells: Cell[] = []
  const before: Cell[] = []

  for (let r = selMinRow.value; r <= selMaxRow.value; r++) {
    for (let c = selMinCol.value; c <= selMaxCol.value; c++) {
      const actualRow = props.tab.filterActive && props.tab.filteredIndices
        ? props.tab.filteredIndices[r] : r
      before.push({ row: actualRow, col: c, value: getCellValue(r, c) })
      cells.push({ row: actualRow, col: c, value: '' })
    }
  }

  try {
    await dataApi.transform(props.tab.session.id, cells, transformType)
    pushStructural()
    if (windowed.value) {
      await refreshAfterStructuralChange()
    } else {
      // Refresh local rows
      const { rows } = await fileApi.getRows(props.tab.session.id, 0, Math.max(localRows.value.length, 1000))
      localRows.value = rows
    }
    notify?.('success', `Applied ${transformType} transform`)
  } catch (err: any) {
    notify?.('error', err.message)
  }
}

// Context menus
const ctxMenu = ref<{ x: number; y: number } | null>(null)
const ctxItems = ref<{ id: string; label: string; icon?: string; danger?: boolean }[]>([])
let ctxContext: { type: string; row?: number; col?: number } = { type: '' }

function openCellCtx(e: MouseEvent, r: number, c: number) {
  if (!isCellSelected(r, c)) {
    sel.value = { startRow: r, startCol: c, endRow: r, endCol: c }
    activeCell.value = { row: r, col: c }
  }
  ctxContext = { type: 'cell', row: r, col: c }
  ctxItems.value = [
    { id: 'copy', label: 'Copy' },
    { id: 'paste', label: 'Paste' },
    { id: 'clear', label: 'Clear' },
    { id: 'sep' as any, label: '---' },
    { id: 'insertRowBelow', label: 'Insert Row Below' },
    { id: 'insertRowAbove', label: 'Insert Row Above' },
    { id: 'deleteRows', label: 'Delete Selected Rows', danger: true },
    { id: 'sep2' as any, label: '---' },
    { id: 'transform', label: 'Transform...' },
    { id: 'sep3' as any, label: '---' },
    { id: 'copyAs', label: 'Copy as Markdown' },
    { id: 'copyAsJson', label: 'Copy as JSON' },
  ]
  ctxMenu.value = { x: e.clientX, y: e.clientY }
}

function openRowCtx(e: MouseEvent, r: number) {
  if (!isRowSelected(r)) selectRow(r, e)
  ctxContext = { type: 'row', row: r }
  ctxItems.value = [
    { id: 'insertRowAbove', label: 'Insert Row Above' },
    { id: 'insertRowBelow', label: 'Insert Row Below' },
    { id: 'deleteRows', label: 'Delete Row', danger: true },
  ]
  ctxMenu.value = { x: e.clientX, y: e.clientY }
}

function openColCtx(e: MouseEvent, c: number) {
  ctxContext = { type: 'col', col: c }
  ctxItems.value = [
    { id: 'insertColLeft', label: 'Insert Column Left' },
    { id: 'insertColRight', label: 'Insert Column Right' },
    { id: 'deleteCol', label: 'Delete Column', danger: true },
    { id: 'sep' as any, label: '---' },
    { id: 'sortAsc', label: 'Sort Ascending' },
    { id: 'sortDesc', label: 'Sort Descending' },
    { id: 'sep2' as any, label: '---' },
    { id: 'deduplicate', label: 'Deduplicate by this Column' },
  ]
  ctxMenu.value = { x: e.clientX, y: e.clientY }
}

async function onCtxSelect(id: string) {
  ctxMenu.value = null
  const id_ = props.tab.session.id

  switch (id) {
    case 'copy': await copySelection(); break
    case 'paste': await pasteFromClipboard(); break
    case 'clear': clearSelection(); break
    case 'insertRowBelow': {
      const r = ctxContext.row ?? selMaxRow.value
      const res = await dataApi.insertRows(id_, r, 1); pushStructural(); pushStructural()
      if (windowed.value) {
        props.tab.session.totalRows = res.totalRows
        await refreshAfterStructuralChange()
      } else {
        localRows.value.splice(r + 1, 0, Array(columns.value.length).fill(''))
        tabsStore.updateTabRows(props.tab.id, localRows.value)
      }
      break
    }
    case 'insertRowAbove': {
      const r = (ctxContext.row ?? selMinRow.value) - 1
      const res = await dataApi.insertRows(id_, r, 1); pushStructural(); pushStructural()
      if (windowed.value) {
        props.tab.session.totalRows = res.totalRows
        await refreshAfterStructuralChange()
      } else {
        localRows.value.splice(r + 1, 0, Array(columns.value.length).fill(''))
        tabsStore.updateTabRows(props.tab.id, localRows.value)
      }
      break
    }
    case 'deleteRows': {
      const rows: number[] = []
      for (let r = selMinRow.value; r <= selMaxRow.value; r++) {
        const actualRow = props.tab.filterActive && props.tab.filteredIndices
          ? props.tab.filteredIndices[r] : r
        rows.push(actualRow)
      }
      const res = await dataApi.deleteRows(id_, rows); pushStructural()
      if (windowed.value) {
        props.tab.session.totalRows = res.totalRows
        // a filtered view's indices are now stale; re-run the filter
        if (props.tab.filterActive && props.tab.filterGroup) {
          const fr = await dataApi.filter(id_, props.tab.filterGroup)
          props.tab.filteredIndices = fr.matchIndices
        }
        await refreshAfterStructuralChange()
      } else {
        localRows.value = localRows.value.filter((_, i) => !rows.includes(i))
        tabsStore.updateTabRows(props.tab.id, localRows.value)
      }
      break
    }
    case 'insertColLeft': {
      const c = (ctxContext.col ?? selMinCol.value) - 1
      const res = await dataApi.insertCols(id_, c, 1); pushStructural()
      props.tab.session.columns = res.columns
      if (windowed.value) {
        await refreshAfterStructuralChange()
      } else {
        localRows.value = localRows.value.map(row => {
          const newRow = [...row]; newRow.splice(c + 1, 0, ''); return newRow
        })
      }
      break
    }
    case 'insertColRight': {
      const c = ctxContext.col ?? selMaxCol.value
      const res = await dataApi.insertCols(id_, c, 1); pushStructural()
      props.tab.session.columns = res.columns
      if (windowed.value) {
        await refreshAfterStructuralChange()
      } else {
        localRows.value = localRows.value.map(row => {
          const newRow = [...row]; newRow.splice(c + 1, 0, ''); return newRow
        })
      }
      break
    }
    case 'deleteCol': {
      const c = ctxContext.col ?? selMinCol.value
      const res = await dataApi.deleteCols(id_, [c]); pushStructural()
      props.tab.session.columns = res.columns
      if (windowed.value) {
        await refreshAfterStructuralChange()
      } else {
        localRows.value = localRows.value.map(row => row.filter((_, i) => i !== c))
      }
      break
    }
    case 'sortAsc': {
      const c = ctxContext.col ?? 0
      await onSort([{ colIndex: c, order: 'asc', type: 'text' }])
      break
    }
    case 'sortDesc': {
      const c = ctxContext.col ?? 0
      await onSort([{ colIndex: c, order: 'desc', type: 'text' }])
      break
    }
    case 'deduplicate': {
      const c = ctxContext.col ?? 0
      const res = await dataApi.deduplicate(id_, [c]); pushStructural()
      notify?.('success', `Removed ${res.removed} duplicate rows`)
      props.tab.session.totalRows = res.totalRows
      if (windowed.value) {
        if (props.tab.filterActive && props.tab.filterGroup) {
          const fr = await dataApi.filter(id_, props.tab.filterGroup)
          props.tab.filteredIndices = fr.matchIndices
        }
        await refreshAfterStructuralChange()
      } else {
        const { rows } = await fileApi.getRows(id_, 0, 5000)
        localRows.value = rows
        tabsStore.updateTabRows(props.tab.id, rows)
      }
      break
    }
    case 'transform': showTransform.value = true; break
    case 'copyAs': {
      const cells: Cell[] = []
      for (let r = selMinRow.value; r <= selMaxRow.value; r++)
        for (let c = selMinCol.value; c <= selMaxCol.value; c++)
          cells.push({ row: actualIndex(r), col: c, value: '' })
      const res = await exportApi.toFormat(id_, 'markdown', cells)
      await navigator.clipboard.writeText(res.content)
      notify?.('success', 'Copied as Markdown')
      break
    }
    case 'copyAsJson': {
      const cells: Cell[] = []
      for (let r = selMinRow.value; r <= selMaxRow.value; r++)
        for (let c = selMinCol.value; c <= selMaxCol.value; c++)
          cells.push({ row: actualIndex(r), col: c, value: '' })
      const res = await exportApi.toFormat(id_, 'json', cells)
      await navigator.clipboard.writeText(res.content)
      notify?.('success', 'Copied as JSON')
      break
    }
  }
}

// Overlay panels
const showFindReplace = ref(false)
const showSort = ref(false)
const showFilter = ref(false)
const showSql = ref(false)
const showTransform = ref(false)

// Expose control methods for toolbar
function openFindReplace() { showFindReplace.value = true }
function openSort() { showSort.value = true }
function openFilter() { showFilter.value = true }
function openSql() { showSql.value = true }

async function insertRowAtActive() {
  const r = activeCell.value.row
  const id_ = props.tab.session.id
  const res = await dataApi.insertRows(id_, r, 1); pushStructural()
  if (windowed.value) {
    props.tab.session.totalRows = res.totalRows
    await refreshAfterStructuralChange()
  } else {
    localRows.value.splice(r + 1, 0, Array(columns.value.length).fill(''))
    tabsStore.updateTabRows(props.tab.id, localRows.value)
  }
  notify?.('success', 'Row inserted')
}

async function deleteSelectedRows() {
  const rows: number[] = []
  for (let r = selMinRow.value; r <= selMaxRow.value; r++) {
    const actualRow = props.tab.filterActive && props.tab.filteredIndices
      ? props.tab.filteredIndices[r] : r
    rows.push(actualRow)
  }
  const id_ = props.tab.session.id
  const res = await dataApi.deleteRows(id_, rows); pushStructural()
  if (windowed.value) {
    props.tab.session.totalRows = res.totalRows
    if (props.tab.filterActive && props.tab.filterGroup) {
      const fr = await dataApi.filter(id_, props.tab.filterGroup)
      props.tab.filteredIndices = fr.matchIndices
    }
    await refreshAfterStructuralChange()
  } else {
    localRows.value = localRows.value.filter((_, i) => !rows.includes(i))
    tabsStore.updateTabRows(props.tab.id, localRows.value)
  }
  sel.value = { startRow: 0, startCol: 0, endRow: 0, endCol: 0 }
  notify?.('success', `Deleted ${rows.length} row(s)`)
}

// ---- Export / format helpers ----

function downloadText(content: string, filename: string, mime = 'text/plain') {
  const blob = new Blob([content], { type: mime })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url; a.download = filename
  document.body.appendChild(a); a.click()
  document.body.removeChild(a); URL.revokeObjectURL(url)
}

async function handleTranspose() {
  loading.value = true
  try {
    await dataApi.transpose(props.tab.session.id)
    pushStructural()
    const info = await fileApi.getInfo(props.tab.session.id)
    props.tab.session.columns = info.columns
    props.tab.session.totalRows = info.totalRows
    // transpose clears any filter/sort (shape changed)
    props.tab.filterActive = false
    props.tab.filteredIndices = null
    if (windowed.value) {
      await refreshAfterStructuralChange()
    } else {
      const { rows } = await fileApi.getRows(props.tab.session.id, 0, Math.max(localRows.value.length, 1000))
      localRows.value = rows
      tabsStore.updateTabRows(props.tab.id, rows)
    }
    notify?.('success', 'Transposed')
  } catch (err: any) { notify?.('error', err.message) }
  finally { loading.value = false }
}

async function handleExportExcel() {
  const defaultPath = props.tab.session.filePath.replace(/\.(csv|tsv|txt)$/i, '.xlsx')
  const path = prompt('Export as Excel to:', defaultPath)
  if (!path) return
  try {
    await exportApi.toExcel(props.tab.session.id, path)
    notify?.('success', 'Exported to ' + path)
  } catch (err: any) { notify?.('error', err.message) }
}

async function handleDownloadFormat(format: string) {
  try {
    const res = await exportApi.toFormat(props.tab.session.id, format, [])
    const ext: Record<string, string> = { markdown: 'md', html: 'html', json: 'json', sql: 'sql', latex: 'tex', csv: 'csv' }
    const base = props.tab.session.fileName.replace(/\.[^.]+$/, '')
    downloadText(res.content, `${base}.${ext[format] ?? format}`)
    notify?.('success', `Downloaded as ${format}`)
  } catch (err: any) { notify?.('error', err.message) }
}

async function handleCopyMarkdownAll() {
  try {
    const res = await exportApi.toFormat(props.tab.session.id, 'markdown', [])
    await navigator.clipboard.writeText(res.content)
    notify?.('success', 'Copied entire file as Markdown')
  } catch (err: any) { notify?.('error', err.message) }
}

async function handleCopyJsonAll() {
  try {
    const res = await exportApi.toFormat(props.tab.session.id, 'json', [])
    await navigator.clipboard.writeText(res.content)
    notify?.('success', 'Copied entire file as JSON')
  } catch (err: any) { notify?.('error', err.message) }
}

// cmd: window event listeners (from command palette)
const _cmdFindReplace = () => openFindReplace()
const _cmdSort = () => openSort()
const _cmdFilter = () => openFilter()
const _cmdSql = () => openSql()
const _cmdTransform = () => { showTransform.value = true }
const _cmdTranspose = () => handleTranspose()
const _cmdCopyMarkdown = () => handleCopyMarkdownAll()
const _cmdCopyJson = () => handleCopyJsonAll()
const _cmdExportExcel = () => handleExportExcel()
const _cmdDownloadFormat = (e: Event) => handleDownloadFormat((e as CustomEvent).detail)

defineExpose({ openFindReplace, openSort, openFilter, openSql, insertRowAtActive, deleteSelectedRows, handleDownloadFormat })

// Aggregate stats — emit to window so StatusBar can pick up regardless of tree position
const aggregateResult = ref<AggregateResult | null>(null)
const selectionCount = computed(() => (selMaxRow.value - selMinRow.value + 1) * (selMaxCol.value - selMinCol.value + 1))

// Debounce aggregate calc on selection change
let aggTimer: ReturnType<typeof setTimeout> | null = null
watch([selMinRow, selMaxRow, selMinCol, selMaxCol], () => {
  if (aggTimer) clearTimeout(aggTimer)
  aggTimer = setTimeout(async () => {
    const count = selectionCount.value
    if (count < 2 || count > 10000) { aggregateResult.value = null; window.dispatchEvent(new CustomEvent('grid:aggregate', { detail: null })); return }
    const cells: Cell[] = []
    for (let r = selMinRow.value; r <= selMaxRow.value; r++)
      for (let c = selMinCol.value; c <= selMaxCol.value; c++)
        cells.push({ row: actualIndex(r), col: c, value: '' })
    try {
      const res = await dataApi.aggregate(props.tab.session.id, cells)
      aggregateResult.value = res
      window.dispatchEvent(new CustomEvent('grid:aggregate', { detail: { ...res, count: cells.length } }))
    } catch {}
  }, 200)
})

// Sync localRows when tab changes externally
watch(() => props.tab.rows, (newRows) => {
  localRows.value = newRows
  if (windowed.value) {
    rowCache.clear()
    newRows.forEach((r, i) => rowCache.set(i, r))
    refetchVisible()
  }
}, { deep: false })

// Windowed mode: keep the visible window of rows loaded as the user scrolls.
watch(visibleRowRange, (range) => {
  if (!windowed.value || range.length === 0) return
  ensureWindow(range[0], range[range.length - 1])
})

// Lazy load more rows when scrolling (eager mode only: append-only, offset-based)
let lazyLoading = false
watch(scrollTop, async (val) => {
  if (windowed.value) return // windowed mode handled by visibleRowRange watcher
  // When a filter is active the full dataset is already loaded; the scroll
  // position is in filtered-display space, so the append heuristic below
  // would be meaningless. Skip it.
  if (props.tab.filterActive) return
  const visibleEnd = Math.ceil((val + viewportH.value) / rowH) + BUFFER
  if (lazyLoading) return
  if (visibleEnd >= localRows.value.length - 50 && localRows.value.length < props.tab.session.totalRows) {
    lazyLoading = true
    try {
      const offset = localRows.value.length
      const { rows: newRows } = await fileApi.getRows(props.tab.session.id, offset, 1000)
      if (newRows.length > 0) localRows.value = [...localRows.value, ...newRows]
    } finally {
      lazyLoading = false
    }
  }
})
</script>

<style scoped>
.grid-wrap {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  outline: none;
  position: relative;
  background: var(--bg-surface);
}

/* Header */
.grid-header-outer {
  display: flex;
  background: var(--bg-grid-header);
  border-bottom: 2px solid var(--border-strong);
  flex-shrink: 0;
  z-index: 10;
  position: relative;
  overflow: hidden;
}
.grid-header-rownum {
  flex-shrink: 0;
  border-right: 2px solid var(--border-strong);
  background: var(--bg-grid-header);
}
.grid-header-row {
  display: flex;
  will-change: transform;
}
.grid-header-cell {
  display: flex;
  align-items: center;
  padding: 0 8px;
  height: var(--header-h);
  border-right: 1px solid var(--border);
  cursor: pointer;
  font-size: 11px;
  font-weight: 600;
  color: var(--text-secondary);
  letter-spacing: 0.3px;
  text-transform: uppercase;
  position: relative;
  user-select: none;
  flex-shrink: 0;
  gap: 4px;
  overflow: hidden;
}
.grid-header-cell:hover { background: var(--bg-hover); color: var(--text-primary); }
.grid-header-cell.selected { background: var(--bg-selected); color: var(--accent); }
.grid-header-cell.sorted { color: var(--accent); }
.header-name { flex: 1; min-width: 0; }
.sort-indicator { color: var(--accent); font-family: var(--font-mono); font-size: 12px; }

.col-resize-handle {
  position: absolute;
  right: 0;
  top: 0;
  bottom: 0;
  width: 4px;
  cursor: col-resize;
  background: transparent;
}
.col-resize-handle:hover { background: var(--accent); }

/* WHERE filter bar */
.filter-bar {
  display: flex;
  align-items: center;
  gap: 8px;
  height: 32px;
  padding: 0 10px;
  background: color-mix(in srgb, var(--accent) 8%, var(--bg-surface));
  border-bottom: 1px solid var(--border);
  flex-shrink: 0;
  font-size: 12px;
}
.filter-bar-where {
  font-family: var(--font-mono);
  font-weight: 700;
  font-size: 11px;
  color: var(--accent);
  letter-spacing: 0.5px;
  flex-shrink: 0;
}
.filter-bar-expr {
  font-family: var(--font-mono);
  color: var(--text-primary);
  flex: 1;
  min-width: 0;
}
.filter-bar-count {
  font-size: 11px;
  color: var(--text-muted);
  flex-shrink: 0;
  font-variant-numeric: tabular-nums;
}
.filter-bar-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 24px;
  height: 24px;
  border: none;
  background: transparent;
  color: var(--text-secondary);
  border-radius: var(--radius-sm);
  cursor: pointer;
  flex-shrink: 0;
  transition: background 0.1s, color 0.1s;
}
.filter-bar-btn:hover { background: var(--bg-hover); color: var(--accent); }

.header-filter-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 18px;
  height: 18px;
  margin-right: 6px;
  padding: 0;
  border: none;
  background: transparent;
  color: var(--text-muted);
  border-radius: 3px;
  cursor: pointer;
  opacity: 0;
  transition: opacity 0.12s, background 0.12s, color 0.12s;
  flex-shrink: 0;
}
.grid-header-cell:hover .header-filter-btn { opacity: 0.7; }
.header-filter-btn:hover { background: var(--bg-hover); color: var(--accent); opacity: 1 !important; }
.header-filter-btn.active {
  opacity: 1;
  color: var(--accent);
  background: color-mix(in srgb, var(--accent) 15%, transparent);
}

.header-editor {
  position: absolute;
  inset: 0;
  border: 2px solid var(--accent);
  background: var(--bg-surface);
  color: var(--text-primary);
  font-family: var(--font-sans);
  font-size: 11px;
  font-weight: 600;
  padding: 0 6px;
  outline: none;
  z-index: 20;
  width: 100%;
}
.grid-header-cell.editing { overflow: visible; }

/* Body */
.grid-body {
  flex: 1;
  overflow: auto;
  position: relative;
}
.grid-row-nums {
  position: absolute;
  left: 0;
  top: 0;
  z-index: 5;
  background: var(--bg-grid-header);
  border-right: 2px solid var(--border-strong);
}
.grid-row-num {
  position: absolute;
  left: 0;
  right: 0;
  display: flex;
  align-items: center;
  justify-content: flex-end;
  padding-right: 8px;
  font-family: var(--font-mono);
  font-size: 10px;
  color: var(--text-muted);
  cursor: pointer;
  user-select: none;
  border-bottom: 1px solid var(--border);
}
.grid-row-num:hover { background: var(--bg-hover); color: var(--text-secondary); }
.grid-row-num.selected { background: var(--bg-selected); color: var(--accent); }

/* Content */
.grid-content {
  position: absolute;
  top: 0;
  left: 0;
}
.grid-row {
  position: absolute;
  display: flex;
  border-bottom: 1px solid var(--border);
}
.grid-row.even { background: var(--bg-stripe); }

/* Cells */
.grid-cell {
  display: flex;
  align-items: center;
  padding: 0 8px;
  border-right: 1px solid var(--border);
  font-size: 12px;
  font-family: var(--font-mono);
  cursor: default;
  height: var(--row-h);
  flex-shrink: 0;
  position: relative;
  min-width: 0;
}
.grid-cell:hover { background: var(--bg-hover); }
.grid-cell.selected { background: var(--bg-selected); }
.grid-cell.active {
  background: var(--bg-selected-active);
  outline: 2px solid var(--accent);
  outline-offset: -2px;
  z-index: 1;
}
.grid-cell.find-match { background: #fef08a; color: #1a1a1a; }
.dark .grid-cell.find-match { background: #713f12; color: #fef08a; }
.cell-value { flex: 1; min-width: 0; }

.cell-editor {
  position: absolute;
  inset: 0;
  border: 2px solid var(--accent);
  background: var(--bg-surface);
  color: var(--text-primary);
  font-family: var(--font-mono);
  font-size: 12px;
  padding: 0 7px;
  outline: none;
  z-index: 10;
}

/* Selection overlay */
.selection-overlay {
  position: absolute;
  background: rgba(37, 99, 235, 0.08);
  border: 1.5px solid var(--accent);
  pointer-events: none;
  z-index: 3;
}

/* Loading */
.grid-loading {
  position: absolute;
  inset: 0;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  background: rgba(var(--bg-surface), 0.7);
  gap: 10px;
  font-size: 13px;
  color: var(--text-secondary);
  z-index: 20;
}
</style>
