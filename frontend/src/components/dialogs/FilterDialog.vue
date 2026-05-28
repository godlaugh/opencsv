<template>
  <Teleport to="body">
    <div class="dialog-overlay" @click.self="emit('close')">
      <div class="dialog" style="min-width: 500px">
        <div class="dialog-header">
          <span class="dialog-title">Filter Data</span>
          <button class="btn-icon" @click="emit('close')"><X :size="14" /></button>
        </div>
        <div class="dialog-body">
          <div class="filter-logic">
            <span>Match</span>
            <select class="select" v-model="group.logic">
              <option value="AND">ALL conditions (AND)</option>
              <option value="OR">ANY condition (OR)</option>
            </select>
          </div>

          <div v-for="(cond, i) in group.conditions" :key="i" class="filter-row">
            <select class="select" v-model="cond.colIndex" style="flex: 1.2">
              <option v-for="col in columns" :key="col.index" :value="col.index">{{ col.name }}</option>
            </select>
            <select class="select" v-model="cond.operator" style="flex: 1.3">
              <option v-for="op in operators" :key="op.value" :value="op.value">{{ op.label }}</option>
            </select>
            <input
              v-if="!noValueOps.includes(cond.operator)"
              class="input flex-1"
              v-model="cond.value"
              placeholder="Value..."
            />
            <button class="btn-icon" @click="removeCondition(i)"><Trash2 :size="13" /></button>
          </div>

          <button class="btn btn-ghost" style="margin-top: 8px" @click="addCondition">
            <Plus :size="13" /> Add Condition
          </button>
        </div>
        <div class="dialog-footer">
          <button class="btn" @click="emit('close')">Cancel</button>
          <button class="btn btn-primary" @click="apply">Apply Filter</button>
        </div>
      </div>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { reactive } from 'vue'
import { X, Trash2, Plus } from 'lucide-vue-next'
import type { Column, FilterGroup } from '@/types'

const props = defineProps<{ columns: Column[] }>()
const emit = defineEmits<{ close: []; filter: [group: FilterGroup] }>()

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

const group = reactive<FilterGroup>({
  logic: 'AND',
  conditions: [{ colIndex: props.columns[0]?.index ?? 0, operator: 'contains', value: '' }]
})

function addCondition() {
  group.conditions.push({ colIndex: props.columns[0]?.index ?? 0, operator: 'contains', value: '' })
}
function removeCondition(i: number) { group.conditions.splice(i, 1) }
function apply() { emit('filter', { ...group }) }
</script>

<style scoped>
.filter-logic { display: flex; align-items: center; gap: 8px; margin-bottom: 12px; font-size: 12px; }
.filter-row { display: flex; align-items: center; gap: 6px; margin-bottom: 8px; }
</style>
