<template>
  <div class="sql-console">
    <div class="sql-header">
      <span class="sql-title">SQL Console</span>
      <span class="sql-hint">Table name: <code>data</code></span>
      <button class="btn-icon" @click="emit('close')"><X :size="13" /></button>
    </div>

    <div class="sql-editor-wrap">
      <textarea
        ref="editorRef"
        class="sql-editor"
        v-model="query"
        placeholder="SELECT * FROM data WHERE col1 = 'value' LIMIT 100"
        @keydown.ctrl.enter.prevent="run"
        @keydown.meta.enter.prevent="run"
        spellcheck="false"
      />
      <button class="btn btn-primary sql-run" @click="run" :disabled="running">
        <span v-if="running" class="spinner" style="width:12px;height:12px" />
        <span v-else>▶ Run</span>
        <span class="sql-shortcut">⌘↵</span>
      </button>
    </div>

    <div v-if="error" class="sql-error">{{ error }}</div>

    <div v-if="result" class="sql-result">
      <div class="sql-result-meta">
        {{ result.totalRows }} rows · {{ execTime }}ms
      </div>
      <div class="sql-result-grid">
        <table>
          <thead>
            <tr>
              <th v-for="col in result.columns" :key="col.index">{{ col.name }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="(row, ri) in result.rows" :key="ri">
              <td v-for="(cell, ci) in row" :key="ci">{{ cell }}</td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { X } from 'lucide-vue-next'
import type { Tab, SqlResult } from '@/types'
import { sqlApi } from '@/api/data'

const props = defineProps<{ tab: Tab }>()
const emit = defineEmits<{ close: [] }>()

const editorRef = ref<HTMLTextAreaElement | null>(null)
const query = ref('SELECT * FROM data LIMIT 100')
const running = ref(false)
const error = ref<string | null>(null)
const result = ref<SqlResult | null>(null)
const execTime = ref(0)

async function run() {
  if (!query.value.trim()) return
  running.value = true
  error.value = null
  const start = Date.now()
  try {
    const res = await sqlApi.query(props.tab.session.id, query.value)
    result.value = res
    execTime.value = Date.now() - start
  } catch (err: any) {
    error.value = err.message
    result.value = null
  } finally {
    running.value = false
  }
}
</script>

<style scoped>
.sql-console {
  position: absolute;
  bottom: 0;
  left: 0;
  right: 0;
  height: 320px;
  background: var(--bg-surface);
  border-top: 2px solid var(--border-strong);
  display: flex;
  flex-direction: column;
  z-index: 30;
  box-shadow: 0 -4px 20px rgba(0,0,0,0.1);
}
.sql-header {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 6px 12px;
  border-bottom: 1px solid var(--border);
  background: var(--bg-grid-header);
  flex-shrink: 0;
}
.sql-title { font-size: 12px; font-weight: 600; color: var(--text-primary); }
.sql-hint { font-size: 11px; color: var(--text-muted); }
.sql-hint code { background: var(--bg-surface-3); padding: 1px 4px; border-radius: 3px; font-family: var(--font-mono); }

.sql-editor-wrap { display: flex; gap: 8px; padding: 8px; flex-shrink: 0; }
.sql-editor {
  flex: 1;
  height: 80px;
  font-family: var(--font-mono);
  font-size: 12px;
  background: var(--bg-surface-2);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  color: var(--text-primary);
  padding: 8px;
  resize: none;
  outline: none;
  line-height: 1.6;
}
.sql-editor:focus { border-color: var(--accent); }
.sql-run {
  align-self: flex-end;
  gap: 6px;
  padding: 6px 16px;
}
.sql-shortcut { font-size: 10px; opacity: 0.7; font-family: var(--font-mono); }

.sql-error {
  margin: 0 8px 6px;
  padding: 6px 10px;
  background: var(--danger-light);
  color: var(--danger);
  border-radius: var(--radius-sm);
  font-size: 12px;
  font-family: var(--font-mono);
}

.sql-result { flex: 1; overflow: auto; }
.sql-result-meta {
  padding: 4px 12px;
  font-size: 11px;
  color: var(--text-muted);
  border-bottom: 1px solid var(--border);
  background: var(--bg-surface-2);
  flex-shrink: 0;
}
.sql-result-grid { overflow: auto; }
table {
  width: 100%;
  border-collapse: collapse;
  font-size: 12px;
  font-family: var(--font-mono);
}
th {
  position: sticky;
  top: 0;
  background: var(--bg-grid-header);
  padding: 5px 10px;
  text-align: left;
  font-size: 10px;
  font-weight: 600;
  text-transform: uppercase;
  color: var(--text-secondary);
  border-bottom: 1px solid var(--border-strong);
  letter-spacing: 0.3px;
  white-space: nowrap;
}
td {
  padding: 4px 10px;
  border-bottom: 1px solid var(--border);
  color: var(--text-primary);
  white-space: nowrap;
  max-width: 300px;
  overflow: hidden;
  text-overflow: ellipsis;
}
tr:hover td { background: var(--bg-hover); }
</style>
