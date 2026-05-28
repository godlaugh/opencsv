<template>
  <div class="app-root" :class="{ dark: isDark }">
    <AppLayout />
    <CommandPalette v-if="showCommandPalette" @close="showCommandPalette = false" />
    <Transition name="fade">
      <div v-if="notification" class="notification" :class="notification.type">
        {{ notification.message }}
      </div>
    </Transition>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, provide, onMounted, onUnmounted } from 'vue'
import { createPinia } from 'pinia'
import AppLayout from '@/components/layout/AppLayout.vue'
import CommandPalette from '@/components/CommandPalette.vue'
import { useSettingsStore } from '@/stores/settings'
import { useTabsStore } from '@/stores/tabs'
import { fileApi } from '@/api/file'

const settings = useSettingsStore()
const tabsStore = useTabsStore()

const isDark = computed(() => {
  if (settings.theme === 'dark') return true
  if (settings.theme === 'light') return false
  return window.matchMedia('(prefers-color-scheme: dark)').matches
})

const showCommandPalette = ref(false)

interface Notification {
  type: 'success' | 'error' | 'info'
  message: string
}
const notification = ref<Notification | null>(null)
let notifTimer: ReturnType<typeof setTimeout> | null = null

function showNotification(type: Notification['type'], message: string) {
  notification.value = { type, message }
  if (notifTimer) clearTimeout(notifTimer)
  notifTimer = setTimeout(() => { notification.value = null }, 3000)
}

provide('notify', showNotification)
provide('openCommandPalette', () => { showCommandPalette.value = true })

// Global keyboard shortcuts
function onKeydown(e: KeyboardEvent) {
  const meta = e.metaKey || e.ctrlKey

  if (meta && e.key === 'p') {
    e.preventDefault()
    showCommandPalette.value = !showCommandPalette.value
    return
  }

  if (showCommandPalette.value && e.key === 'Escape') {
    showCommandPalette.value = false
    return
  }

  if (meta && e.key === 's') {
    e.preventDefault()
    const tab = tabsStore.activeTab
    if (tab) {
      fileApi.save(tab.session.id).then(() => {
        tabsStore.markModified(tab.session.id, false)
        showNotification('success', 'File saved')
      }).catch(err => showNotification('error', err.message))
    }
    return
  }

  if (meta && e.key === 'w') {
    e.preventDefault()
    const tab = tabsStore.activeTab
    if (tab) tabsStore.removeTab(tab.session.id)
    return
  }
}

onMounted(() => window.addEventListener('keydown', onKeydown))
onUnmounted(() => window.removeEventListener('keydown', onKeydown))
</script>

<style scoped>
.app-root {
  height: 100%;
  display: flex;
  flex-direction: column;
  background: var(--bg-app);
  color: var(--text-primary);
}

.notification {
  position: fixed;
  bottom: 40px;
  left: 50%;
  transform: translateX(-50%);
  padding: 8px 18px;
  border-radius: var(--radius);
  font-size: 12px;
  font-weight: 500;
  z-index: 9999;
  box-shadow: var(--shadow-lg);
  pointer-events: none;
}
.notification.success { background: var(--success); color: white; }
.notification.error { background: var(--danger); color: white; }
.notification.info { background: var(--accent); color: white; }
</style>
