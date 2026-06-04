import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type { Tab, FileSession } from '@/types'
import { fileApi } from '@/api/file'
import { clearHandle } from '@/utils/fileSystem'

export const useTabsStore = defineStore('tabs', () => {
  const tabs = ref<Tab[]>([])
  const activeTabId = ref<string | null>(null)

  const activeTab = computed(() => tabs.value.find(t => t.id === activeTabId.value) ?? null)

  function addTab(session: FileSession, rows: string[][]): Tab {
    const tab: Tab = {
      id: session.id,
      session,
      rows,
      cachedPages: new Map(),
      loading: false,
      filterActive: false,
      filteredIndices: null,
      filterGroup: null
    }
    tabs.value.push(tab)
    activeTabId.value = tab.id
    return tab
  }

  function removeTab(id: string) {
    const idx = tabs.value.findIndex(t => t.id === id)
    if (idx === -1) return
    fileApi.close(id).catch(() => {})
    clearHandle(id)
    tabs.value.splice(idx, 1)
    if (activeTabId.value === id) {
      activeTabId.value = tabs.value[Math.max(0, idx - 1)]?.id ?? null
    }
  }

  function setActiveTab(id: string) {
    activeTabId.value = id
  }

  function updateTabRows(id: string, rows: string[][]) {
    const tab = tabs.value.find(t => t.id === id)
    if (tab) {
      tab.rows = rows
      tab.session.totalRows = rows.length
    }
  }

  function markModified(id: string, modified = true) {
    const tab = tabs.value.find(t => t.id === id)
    if (tab) tab.session.modified = modified
  }

  return { tabs, activeTabId, activeTab, addTab, removeTab, setActiveTab, updateTabRows, markModified }
})
