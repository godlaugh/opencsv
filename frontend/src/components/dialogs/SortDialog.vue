<template>
  <Teleport to="body">
    <div class="dialog-overlay" @click.self="emit('close')">
      <div class="dialog" style="min-width: 440px">
        <div class="dialog-header">
          <span class="dialog-title">Sort Data</span>
          <button class="btn-icon" @click="emit('close')"><X :size="14" /></button>
        </div>
        <div class="dialog-body">
          <div v-for="(key, i) in keys" :key="i" class="sort-row">
            <span class="sort-label">{{ i === 0 ? 'Sort by' : 'Then by' }}</span>
            <select class="select flex-1" v-model="key.colIndex">
              <option v-for="col in columns" :key="col.index" :value="col.index">{{ col.name }}</option>
            </select>
            <select class="select" v-model="key.order">
              <option value="asc">Ascending ↑</option>
              <option value="desc">Descending ↓</option>
            </select>
            <select class="select" v-model="key.type">
              <option value="text">Text</option>
              <option value="number">Number</option>
              <option value="date">Date</option>
              <option value="length">Length</option>
            </select>
            <button class="btn-icon" @click="removeKey(i)" :disabled="keys.length === 1"><Trash2 :size="13" /></button>
          </div>
          <button class="btn btn-ghost" style="margin-top: 8px" @click="addKey">
            <Plus :size="13" /> Add Level
          </button>
        </div>
        <div class="dialog-footer">
          <button class="btn" @click="emit('close')">Cancel</button>
          <button class="btn btn-primary" @click="apply">Sort</button>
        </div>
      </div>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { X, Trash2, Plus } from 'lucide-vue-next'
import type { Column, SortKey } from '@/types'

const props = defineProps<{ columns: Column[]; initial?: SortKey[] | null }>()
const emit = defineEmits<{ close: []; sort: [keys: SortKey[]] }>()

// Initialize from the existing sort so the global dialog reflects whatever
// the per-column sort buttons produced (and vice-versa).
const keys = ref<SortKey[]>(
  props.initial && props.initial.length > 0
    ? props.initial.map(k => ({ ...k }))
    : [{ colIndex: props.columns[0]?.index ?? 0, order: 'asc', type: 'text' }]
)

function addKey() {
  keys.value.push({ colIndex: props.columns[0]?.index ?? 0, order: 'asc', type: 'text' })
}
function removeKey(i: number) { keys.value.splice(i, 1) }
function apply() { emit('sort', keys.value) }
</script>

<style scoped>
.sort-row { display: flex; align-items: center; gap: 6px; margin-bottom: 8px; }
.sort-label { font-size: 11px; color: var(--text-muted); width: 50px; flex-shrink: 0; }
</style>
