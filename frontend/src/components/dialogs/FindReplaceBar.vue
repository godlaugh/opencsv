<template>
  <div class="find-bar">
    <div class="find-section">
      <input
        ref="findInput"
        class="input"
        placeholder="Find..."
        v-model="query"
        @keydown.enter="findNext"
        @keydown.escape="emit('close')"
        @input="onQueryChange"
      />
      <button class="btn btn-ghost" :class="{ active: useRegex }" @click="useRegex = !useRegex" title="Use Regex">.*</button>
      <button class="btn btn-ghost" :class="{ active: caseSensitive }" @click="caseSensitive = !caseSensitive" title="Case Sensitive">Aa</button>
    </div>

    <div class="find-section">
      <input
        class="input"
        placeholder="Replace with..."
        v-model="replacement"
        @keydown.enter="replaceNext"
        @keydown.escape="emit('close')"
      />
      <button class="btn btn-ghost" :disabled="!query" @click="replaceNext">Replace</button>
      <button class="btn btn-ghost" :disabled="!query" @click="replaceAll">All</button>
    </div>

    <div class="find-info" v-if="matchCount !== null">
      <span v-if="matchCount === 0" class="text-danger">No matches</span>
      <span v-else class="text-success">{{ currentMatch + 1 }} / {{ matchCount }}</span>
    </div>

    <div class="find-nav">
      <button class="btn-icon" :disabled="matchCount === 0" @click="findPrev" title="Previous (⇧Enter)">↑</button>
      <button class="btn-icon" :disabled="matchCount === 0" @click="findNext" title="Next (Enter)">↓</button>
    </div>

    <button class="btn-icon" @click="emit('close')" title="Close (Esc)">
      <X :size="13" />
    </button>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, watch } from 'vue'
import { X } from 'lucide-vue-next'
import type { Tab, FindMatch } from '@/types'
import { dataApi } from '@/api/data'

const props = defineProps<{ tab: Tab }>()
const emit = defineEmits<{ close: []; matches: [matches: FindMatch[], current: number] }>()

const findInput = ref<HTMLInputElement | null>(null)
const query = ref('')
const replacement = ref('')
const useRegex = ref(false)
const caseSensitive = ref(false)
const matches = ref<FindMatch[]>([])
const matchCount = ref<number | null>(null)
const currentMatch = ref(0)

onMounted(() => findInput.value?.focus())

let searchTimer: ReturnType<typeof setTimeout> | null = null
function onQueryChange() {
  if (searchTimer) clearTimeout(searchTimer)
  if (!query.value) { matchCount.value = null; emit('matches', [], 0); return }
  searchTimer = setTimeout(doFind, 300)
}

async function doFind() {
  try {
    const result = await dataApi.find(props.tab.session.id, query.value, {
      regex: useRegex.value,
      caseSensitive: caseSensitive.value
    })
    matches.value = result.matches
    matchCount.value = result.count
    currentMatch.value = 0
    emit('matches', result.matches, 0)
  } catch {}
}

function findNext() {
  if (matchCount.value === 0 || !matchCount.value) { doFind(); return }
  currentMatch.value = (currentMatch.value + 1) % matchCount.value
  emit('matches', matches.value, currentMatch.value)
}

function findPrev() {
  if (!matchCount.value) return
  currentMatch.value = (currentMatch.value - 1 + matchCount.value) % matchCount.value
  emit('matches', matches.value, currentMatch.value)
}

async function replaceNext() {
  if (!query.value) return
  try {
    const res = await dataApi.replace(props.tab.session.id, query.value, replacement.value, {
      regex: useRegex.value, caseSensitive: caseSensitive.value, all: false
    })
    await doFind()
  } catch {}
}

async function replaceAll() {
  if (!query.value) return
  try {
    const res = await dataApi.replace(props.tab.session.id, query.value, replacement.value, {
      regex: useRegex.value, caseSensitive: caseSensitive.value, all: true
    })
    matchCount.value = 0
    emit('matches', [], 0)
  } catch {}
}

watch([useRegex, caseSensitive], doFind)
</script>

<style scoped>
.find-bar {
  position: absolute;
  top: 0;
  right: 0;
  background: var(--bg-surface);
  border: 1px solid var(--border);
  border-top: none;
  border-right: none;
  border-radius: 0 0 0 var(--radius);
  padding: 6px 8px;
  display: flex;
  align-items: center;
  gap: 6px;
  box-shadow: var(--shadow);
  z-index: 50;
}
.find-section { display: flex; align-items: center; gap: 3px; }
.find-info { font-size: 11px; min-width: 60px; text-align: center; }
.find-nav { display: flex; gap: 2px; }
.btn.active { background: var(--accent-light); color: var(--accent); border-color: var(--accent); }
.text-danger { color: var(--danger); }
.text-success { color: var(--success); }
</style>
