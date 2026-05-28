<template>
  <div class="tabbar" v-if="tabs.length > 0">
    <div
      v-for="tab in tabs"
      :key="tab.id"
      class="tab"
      :class="{ active: tab.id === activeTabId, modified: tab.session.modified }"
      @click="tabsStore.setActiveTab(tab.id)"
      @mousedown.middle.prevent="tabsStore.removeTab(tab.id)"
    >
      <span class="tab-icon">
        <FileText :size="12" />
      </span>
      <span class="tab-name">{{ tab.session.fileName }}</span>
      <span v-if="tab.session.modified" class="tab-dot" />
      <button class="tab-close" @click.stop="tabsStore.removeTab(tab.id)">
        <X :size="11" />
      </button>
    </div>

    <!-- Add tab button -->
    <label class="tab-add tooltip" data-tip="Open File">
      <Plus :size="13" />
      <input type="file" accept=".csv,.tsv,.txt,.xlsx" style="display:none" @change="onFileInput" multiple />
    </label>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { FileText, X, Plus } from 'lucide-vue-next'
import { useTabsStore } from '@/stores/tabs'
import { fileApi } from '@/api/file'

const tabsStore = useTabsStore()
const tabs = computed(() => tabsStore.tabs)
const activeTabId = computed(() => tabsStore.activeTabId)

async function onFileInput(e: Event) {
  const files = (e.target as HTMLInputElement).files
  if (!files) return
  for (const file of Array.from(files)) {
    const arrayBuffer = await file.arrayBuffer()
    const bytes = new Uint8Array(arrayBuffer)
    try {
      const response = await fetch('/api/files/upload', {
        method: 'POST',
        headers: { 'Content-Type': 'application/octet-stream', 'X-Filename': encodeURIComponent(file.name) },
        body: bytes
      })
      const data = await response.json()
      const session = await fileApi.open(data.filePath)
      tabsStore.addTab(session, session.rows)
    } catch {}
  }
  ;(e.target as HTMLInputElement).value = ''
}
</script>

<style scoped>
.tabbar {
  height: var(--tabbar-h);
  background: var(--bg-app);
  border-bottom: 1px solid var(--border);
  display: flex;
  align-items: end;
  padding: 0 8px;
  overflow-x: auto;
  flex-shrink: 0;
  scrollbar-width: none;
}
.tabbar::-webkit-scrollbar { display: none; }

.tab {
  display: flex;
  align-items: center;
  gap: 5px;
  padding: 5px 10px 5px 8px;
  border: 1px solid transparent;
  border-bottom: none;
  border-radius: var(--radius-sm) var(--radius-sm) 0 0;
  cursor: pointer;
  font-size: 12px;
  color: var(--text-secondary);
  white-space: nowrap;
  position: relative;
  transition: background 0.1s, color 0.1s;
  max-width: 180px;
}
.tab:hover { background: var(--bg-surface-2); color: var(--text-primary); }
.tab.active {
  background: var(--bg-surface);
  border-color: var(--border);
  color: var(--text-primary);
  margin-bottom: -1px;
  z-index: 1;
}
.tab-name { overflow: hidden; text-overflow: ellipsis; flex: 1; min-width: 0; }
.tab-dot {
  width: 6px; height: 6px;
  border-radius: 50%;
  background: var(--warning);
  flex-shrink: 0;
}
.tab-close {
  background: none;
  border: none;
  cursor: pointer;
  color: var(--text-muted);
  display: flex;
  align-items: center;
  padding: 1px;
  border-radius: 2px;
  transition: background 0.1s, color 0.1s;
}
.tab-close:hover { background: var(--bg-hover); color: var(--danger); }
.tab-icon { color: var(--accent); flex-shrink: 0; }

.tab-add {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  margin-bottom: 1px;
  cursor: pointer;
  color: var(--text-muted);
  border-radius: var(--radius-sm);
  transition: background 0.1s, color 0.1s;
}
.tab-add:hover { background: var(--bg-hover); color: var(--text-primary); }
</style>
