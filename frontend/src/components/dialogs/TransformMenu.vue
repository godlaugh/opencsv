<template>
  <Teleport to="body">
    <div class="dialog-overlay" @click.self="emit('close')">
      <div class="dialog" style="min-width: 300px">
        <div class="dialog-header">
          <span class="dialog-title">Transform Text</span>
          <button class="btn-icon" @click="emit('close')"><X :size="14" /></button>
        </div>
        <div class="dialog-body" style="padding: 8px 12px;">
          <div class="transform-grid">
            <button v-for="t in transforms" :key="t.value" class="transform-btn" @click="apply(t.value)">
              <span class="transform-label">{{ t.label }}</span>
              <span class="transform-example">{{ t.example }}</span>
            </button>
          </div>
        </div>
      </div>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { X } from 'lucide-vue-next'
const emit = defineEmits<{ close: []; apply: [type: string] }>()

const transforms = [
  { value: 'upper', label: 'UPPERCASE', example: 'HELLO' },
  { value: 'lower', label: 'lowercase', example: 'hello' },
  { value: 'title', label: 'Title Case', example: 'Hello World' },
  { value: 'camel', label: 'camelCase', example: 'helloWorld' },
  { value: 'pascal', label: 'PascalCase', example: 'HelloWorld' },
  { value: 'snake', label: 'snake_case', example: 'hello_world' },
  { value: 'trim', label: 'Trim', example: 'trim spaces' },
  { value: 'ltrim', label: 'Left Trim', example: 'ltrim' },
  { value: 'rtrim', label: 'Right Trim', example: 'rtrim' },
]

function apply(type: string) {
  emit('apply', type)
  emit('close')
}
</script>

<style scoped>
.transform-grid { display: grid; grid-template-columns: 1fr 1fr 1fr; gap: 6px; }
.transform-btn {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 10px 8px;
  border: 1px solid var(--border);
  border-radius: var(--radius);
  background: var(--bg-surface);
  cursor: pointer;
  transition: background 0.1s, border-color 0.1s;
  gap: 3px;
}
.transform-btn:hover { background: var(--accent-light); border-color: var(--accent); }
.transform-label { font-size: 11px; font-weight: 600; color: var(--text-primary); }
.transform-example { font-size: 10px; color: var(--text-muted); font-family: var(--font-mono); }
</style>
