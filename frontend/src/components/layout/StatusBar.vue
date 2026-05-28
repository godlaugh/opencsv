<template>
  <div class="statusbar">
    <template v-if="activeTab">
      <span class="stat">
        <span class="stat-label">Rows:</span>
        <span class="stat-value">{{ activeTab.session.totalRows.toLocaleString() }}</span>
      </span>
      <span class="stat">
        <span class="stat-label">Cols:</span>
        <span class="stat-value">{{ activeTab.session.columns.length }}</span>
      </span>

      <template v-if="aggregate && selCount > 1">
        <span class="divider-v" />
        <span class="stat">Count: <b>{{ aggregate.count }}</b></span>
        <span v-if="aggregate.sum !== 0 || selCount > 1" class="stat">
          Sum: <b>{{ aggregate.sum.toLocaleString(undefined, { maximumFractionDigits: 4 }) }}</b>
        </span>
        <span class="stat">Avg: <b>{{ aggregate.avg }}</b></span>
        <span class="stat">Min: <b>{{ aggregate.min }}</b></span>
        <span class="stat">Max: <b>{{ aggregate.max }}</b></span>
        <span class="stat">Unique: <b>{{ aggregate.unique }}</b></span>
      </template>

      <template v-if="activeTab.filterActive && activeTab.filteredIndices">
        <span class="divider-v" />
        <span class="badge">
          Filtered: {{ activeTab.filteredIndices.length.toLocaleString() }} / {{ activeTab.session.totalRows.toLocaleString() }}
        </span>
        <button class="btn-icon" @click="clearFilter" title="Clear Filter">
          <X :size="11" />
        </button>
      </template>
    </template>

    <div class="flex-1" />

    <template v-if="activeTab">
      <span class="stat">
        <span class="stat-label">Enc:</span>
        <span class="stat-value">{{ activeTab.session.config.encoding || 'UTF-8' }}</span>
      </span>
      <span class="stat">
        <span class="stat-label">Sep:</span>
        <span class="stat-value mono">{{ delimLabel }}</span>
      </span>
      <span v-if="activeTab.session.modified" class="badge badge-danger">Modified</span>
    </template>

    <span v-else class="stat-label">Ready</span>
  </div>
</template>

<script setup lang="ts">
import { computed, inject } from 'vue'
import { X } from 'lucide-vue-next'
import { useTabsStore } from '@/stores/tabs'
import type { AggregateResult } from '@/types'

const tabsStore = useTabsStore()
const activeTab = computed(() => tabsStore.activeTab)
const aggregate = inject<AggregateResult | null>('aggregateResult', null)
const selCount = inject<number>('selectionCount', 0)

const delimLabel = computed(() => {
  const d = activeTab.value?.session.config.delimiter
  if (!d || d === ',') return ','
  if (d === '\t') return 'TAB'
  if (d === ';') return ';'
  if (d === '|') return '|'
  return d
})

function clearFilter() {
  const tab = tabsStore.activeTab
  if (!tab) return
  tab.filterActive = false
  tab.filteredIndices = null
}
</script>

<style scoped>
.statusbar {
  height: var(--statusbar-h);
  background: var(--bg-surface-2);
  border-top: 1px solid var(--border);
  display: flex;
  align-items: center;
  padding: 0 10px;
  gap: 10px;
  font-size: 11px;
  flex-shrink: 0;
  color: var(--text-secondary);
}
.stat { display: flex; align-items: center; gap: 3px; }
.stat-label { color: var(--text-muted); }
.stat-value { color: var(--text-primary); font-family: var(--font-mono); }
.divider-v { width: 1px; height: 12px; background: var(--border); }
</style>
