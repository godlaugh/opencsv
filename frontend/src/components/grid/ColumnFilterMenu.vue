<template>
  <Teleport to="body">
    <div class="cfm-backdrop" @mousedown.self="emit('close')">
      <div
        class="cfm-popover"
        :style="popoverStyle"
        @mousedown.stop
        @click.stop
      >
        <!-- SORT -->
        <div class="cfm-section">
          <div class="cfm-label">SORT</div>
          <div class="cfm-sort-row">
            <button class="cfm-chip" :class="{ active: activeSort?.order === 'asc' }" @click="sort('asc')">
              <BarChart3 :size="13" />
              <span>Sort Ascending</span>
            </button>
            <button class="cfm-chip" :class="{ active: activeSort?.order === 'desc' }" @click="sort('desc')">
              <BarChartHorizontal :size="13" />
              <span>Sort Descending</span>
            </button>
          </div>
          <details class="cfm-more" :open="!!activeSort && activeSort.type !== 'text'">
            <summary>More options</summary>
            <div class="cfm-more-row">
              <label class="cfm-sub-label">Type</label>
              <select v-model="sortType" class="cfm-select">
                <option value="text">Text</option>
                <option value="number">Number</option>
                <option value="date">Date</option>
                <option value="length">Length</option>
              </select>
            </div>
          </details>
        </div>

        <div class="cfm-divider" />

        <!-- FILTER -->
        <div class="cfm-section">
          <div class="cfm-filter-header">
            <span class="cfm-label">FILTER</span>
            <div class="cfm-tabs">
              <button
                class="cfm-tab"
                :class="{ active: mode === 'values' }"
                @click="mode = 'values'"
              >Filter by values</button>
              <button
                class="cfm-tab"
                :class="{ active: mode === 'condition' }"
                @click="mode = 'condition'"
              >Filter by condition</button>
            </div>
          </div>

          <!-- Filter by values -->
          <div v-if="mode === 'values'" class="cfm-values">
            <div class="cfm-search-wrap">
              <Search :size="13" class="cfm-search-icon" />
              <input
                ref="searchRef"
                class="cfm-search"
                placeholder="Filter values..."
                v-model="search"
                @input="onSearchInput"
              />
            </div>
            <div class="cfm-actions">
              <button class="cfm-btn" @click="selectAllVisible">Select all</button>
              <button class="cfm-btn" @click="clearAll">Clear</button>
              <span class="cfm-count">{{ checkedCount }} selected</span>
            </div>

            <div class="cfm-list" @scroll="onListScroll">
              <div v-if="loading && values.length === 0" class="cfm-empty">Loading…</div>
              <div v-else-if="values.length === 0" class="cfm-empty">No values</div>
              <label
                v-for="v in values"
                :key="v.value || '__EMPTY__'"
                class="cfm-item"
                :class="{ checked: checked.has(v.value) }"
              >
                <input
                  type="checkbox"
                  :checked="checked.has(v.value)"
                  @change="toggle(v.value)"
                />
                <span class="cfm-item-text">{{ v.value === '' ? '(blank)' : v.value }}</span>
                <span class="cfm-item-count">{{ v.count }}</span>
              </label>
              <div v-if="truncated" class="cfm-more-hint">
                Showing first {{ values.length }} unique values. Use search to narrow.
              </div>
            </div>
          </div>

          <!-- Filter by condition -->
          <div v-else class="cfm-condition">
            <select v-model="condOp" class="cfm-select">
              <option v-for="op in operators" :key="op.value" :value="op.value">{{ op.label }}</option>
            </select>
            <input
              v-if="!noValueOps.includes(condOp)"
              v-model="condValue"
              class="cfm-input"
              placeholder="Value…"
              @keydown.enter.prevent="apply"
            />
          </div>
        </div>

        <div class="cfm-footer">
          <button class="cfm-btn" @click="emit('close')">Close</button>
          <button class="cfm-btn" @click="clearFilter">Clear Filter</button>
          <button class="cfm-btn primary" @click="apply">Apply Filter</button>
        </div>
      </div>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, nextTick, watch } from 'vue'
import { Search, BarChart3, BarChartHorizontal } from 'lucide-vue-next'
import { dataApi } from '@/api/data'
import type { ColumnQuickFilter, SortKey } from '@/types'

const props = defineProps<{
  sessionId: string
  colIndex: number
  colName: string
  anchor: { x: number; y: number; width: number }
  initial?: ColumnQuickFilter
  activeSort?: { order: 'asc' | 'desc'; type: 'text' | 'number' | 'date' | 'length' } | null
}>()

const emit = defineEmits<{
  close: []
  apply: [filter: ColumnQuickFilter | null]
  sort: [key: SortKey]
}>()

const mode = ref<'values' | 'condition'>(props.initial?.mode ?? 'values')
const search = ref('')
const values = ref<{ value: string; count: number }[]>([])
const truncated = ref(false)
const loading = ref(false)

// checked set of selected values; empty set = nothing selected
const checked = ref<Set<string>>(new Set(props.initial?.selectedValues ?? []))
const initialHadNoSelection = !props.initial?.selectedValues

const condOp = ref(props.initial?.operator ?? 'contains')
const condValue = ref(props.initial?.value ?? '')

const sortType = ref<'text' | 'number' | 'date' | 'length'>(props.activeSort?.type ?? 'text')

const operators = [
  { value: 'contains', label: 'contains' },
  { value: 'notContains', label: 'does not contain' },
  { value: 'eq', label: 'equals' },
  { value: 'ne', label: 'not equals' },
  { value: 'startsWith', label: 'starts with' },
  { value: 'endsWith', label: 'ends with' },
  { value: 'gt', label: 'greater than' },
  { value: 'lt', label: 'less than' },
  { value: 'empty', label: 'is empty' },
  { value: 'notEmpty', label: 'is not empty' },
  { value: 'regex', label: 'matches regex' },
]
const noValueOps = ['empty', 'notEmpty']

const checkedCount = computed(() => checked.value.size)

const popoverStyle = computed(() => {
  // anchor: bottom of header cell. Default popover width 360.
  const W = 380
  const margin = 8
  let left = props.anchor.x
  if (left + W > window.innerWidth - margin) {
    left = Math.max(margin, window.innerWidth - W - margin)
  }
  if (left < margin) left = margin
  return {
    left: left + 'px',
    top: props.anchor.y + 'px',
    width: W + 'px'
  }
})

let searchTimer: ReturnType<typeof setTimeout> | null = null
function onSearchInput() {
  if (searchTimer) clearTimeout(searchTimer)
  searchTimer = setTimeout(() => loadValues(), 200)
}

async function loadValues(limit = 500) {
  loading.value = true
  try {
    const res = await dataApi.getColumnValues(props.sessionId, props.colIndex, search.value, limit)
    values.value = res.values
    truncated.value = res.truncated
    if (initialHadNoSelection && checked.value.size === 0) {
      // Default: nothing pre-checked, user picks what they want to keep
    }
  } finally {
    loading.value = false
  }
}

function onListScroll(e: Event) {
  const el = e.target as HTMLElement
  if (truncated.value && el.scrollHeight - el.scrollTop - el.clientHeight < 80) {
    loadValues(Math.min(values.value.length + 500, 50000))
  }
}

function selectAllVisible() {
  for (const v of values.value) checked.value.add(v.value)
  checked.value = new Set(checked.value)
}
function clearAll() {
  checked.value = new Set()
}
function toggle(v: string) {
  if (checked.value.has(v)) checked.value.delete(v)
  else checked.value.add(v)
  checked.value = new Set(checked.value)
}

function apply() {
  if (mode.value === 'values') {
    if (checked.value.size === 0) {
      emit('apply', null)
    } else {
      emit('apply', { mode: 'values', selectedValues: Array.from(checked.value) })
    }
  } else {
    if (!noValueOps.includes(condOp.value) && condValue.value === '') {
      emit('apply', null)
    } else {
      emit('apply', { mode: 'condition', operator: condOp.value, value: condValue.value })
    }
  }
  emit('close')
}

function clearFilter() {
  checked.value = new Set()
  condValue.value = ''
  emit('apply', null)
  emit('close')
}

function sort(order: 'asc' | 'desc') {
  emit('sort', { colIndex: props.colIndex, order, type: sortType.value })
  emit('close')
}

const searchRef = ref<HTMLInputElement | null>(null)

onMounted(async () => {
  await loadValues()
  nextTick(() => searchRef.value?.focus())
})

watch(() => props.colIndex, () => loadValues())

// Close on Escape
function onKey(e: KeyboardEvent) {
  if (e.key === 'Escape') emit('close')
}
onMounted(() => window.addEventListener('keydown', onKey))
import { onUnmounted } from 'vue'
onUnmounted(() => window.removeEventListener('keydown', onKey))
</script>

<style scoped>
.cfm-backdrop {
  position: fixed; inset: 0;
  z-index: 1000;
  background: transparent;
}
.cfm-popover {
  position: absolute;
  background: var(--bg-surface);
  color: var(--text-primary);
  border: 1px solid var(--border);
  border-radius: var(--radius);
  box-shadow: var(--shadow-lg);
  display: flex;
  flex-direction: column;
  max-height: 70vh;
  overflow: hidden;
}

.cfm-section { padding: 12px 14px; }
.cfm-label {
  font-size: 10px;
  letter-spacing: 1.2px;
  font-weight: 700;
  color: var(--text-muted);
  margin-bottom: 8px;
}
.cfm-divider { height: 1px; background: var(--border); }

.cfm-sort-row { display: flex; gap: 8px; }
.cfm-chip {
  flex: 1;
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 6px 10px;
  font-size: 12px;
  background: var(--bg-surface-2);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  color: var(--text-primary);
  cursor: pointer;
  transition: background 0.1s;
}
.cfm-chip:hover { background: var(--bg-hover); }
.cfm-chip.active {
  background: color-mix(in srgb, var(--accent) 15%, transparent);
  border-color: var(--accent);
  color: var(--accent);
  font-weight: 600;
}

.cfm-more { margin-top: 8px; font-size: 11px; color: var(--text-secondary); }
.cfm-more summary { cursor: pointer; user-select: none; }
.cfm-more-row { margin-top: 8px; display: flex; align-items: center; gap: 8px; }
.cfm-sub-label { font-size: 11px; color: var(--text-muted); }

.cfm-filter-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 10px;
  gap: 10px;
}
.cfm-tabs {
  display: flex;
  background: var(--bg-surface-2);
  border-radius: 999px;
  padding: 2px;
}
.cfm-tab {
  padding: 4px 12px;
  font-size: 12px;
  background: transparent;
  border: none;
  border-radius: 999px;
  cursor: pointer;
  color: var(--text-secondary);
  transition: background 0.1s, color 0.1s;
}
.cfm-tab.active {
  background: var(--bg-surface);
  color: var(--text-primary);
  box-shadow: var(--shadow-sm);
}

.cfm-search-wrap {
  position: relative;
  margin-bottom: 8px;
}
.cfm-search-icon {
  position: absolute;
  left: 9px;
  top: 50%;
  transform: translateY(-50%);
  color: var(--text-muted);
}
.cfm-search {
  width: 100%;
  padding: 7px 10px 7px 28px;
  font-size: 12px;
  background: var(--bg-app);
  border: 1px solid var(--accent);
  border-radius: var(--radius-sm);
  color: var(--text-primary);
  outline: none;
}

.cfm-actions {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 8px;
}
.cfm-btn {
  padding: 4px 10px;
  font-size: 11px;
  background: var(--bg-surface-2);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  color: var(--text-primary);
  cursor: pointer;
}
.cfm-btn:hover { background: var(--bg-hover); }
.cfm-btn.primary {
  background: var(--accent);
  color: white;
  border-color: var(--accent);
}
.cfm-btn.primary:hover { filter: brightness(1.08); }
.cfm-count { font-size: 11px; color: var(--text-muted); margin-left: auto; }

.cfm-list {
  flex: 1;
  overflow-y: auto;
  max-height: 280px;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  background: var(--bg-app);
}
.cfm-empty { padding: 24px; text-align: center; color: var(--text-muted); font-size: 12px; }
.cfm-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 5px 10px;
  font-size: 12px;
  cursor: pointer;
  border-bottom: 1px solid var(--border);
}
.cfm-item:last-child { border-bottom: none; }
.cfm-item:nth-child(even) { background: var(--bg-surface-2); }
.cfm-item:hover { background: var(--bg-hover); }
.cfm-item.checked { font-weight: 600; }
.cfm-item input[type="checkbox"] { accent-color: var(--accent); }
.cfm-item-text { flex: 1; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.cfm-item-count { color: var(--text-muted); font-size: 11px; font-variant-numeric: tabular-nums; }
.cfm-more-hint { padding: 8px 10px; font-size: 11px; color: var(--text-muted); text-align: center; }

.cfm-condition { display: flex; flex-direction: column; gap: 8px; }
.cfm-select, .cfm-input {
  padding: 6px 10px;
  font-size: 12px;
  background: var(--bg-app);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  color: var(--text-primary);
  outline: none;
}
.cfm-select:focus, .cfm-input:focus { border-color: var(--accent); }

.cfm-footer {
  display: flex;
  gap: 8px;
  padding: 10px 14px;
  border-top: 1px solid var(--border);
  background: var(--bg-surface-2);
  justify-content: flex-end;
}
</style>
