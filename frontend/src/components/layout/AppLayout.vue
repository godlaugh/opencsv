<template>
  <div
    class="app-layout"
    @dragenter.prevent="onDragEnter"
    @dragover.prevent
    @dragleave.prevent="onDragLeave"
    @drop.prevent="onDrop"
  >
    <Toolbar @find-replace="gridRef?.openFindReplace()" @sort="gridRef?.openSort()" @filter="gridRef?.openFilter()" @sql="gridRef?.openSql()" @insert-row="gridRef?.insertRowAtActive()" @delete-rows="gridRef?.deleteSelectedRows()" />
    <TabBar />
    <div class="app-body">
      <template v-if="activeTab">
        <VirtualGrid :key="activeTab.id" :tab="activeTab" ref="gridRef" />
      </template>
      <template v-else>
        <WelcomeScreen />
      </template>
    </div>
    <StatusBar />

    <div v-if="dragDepth > 0" class="drop-overlay">
      <div class="drop-hint">
        <Upload :size="32" />
        <span>Drop CSV / Excel files to open</span>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { Upload } from 'lucide-vue-next'
import Toolbar from './Toolbar.vue'
import TabBar from './TabBar.vue'
import StatusBar from './StatusBar.vue'
import VirtualGrid from '@/components/grid/VirtualGrid.vue'
import WelcomeScreen from './WelcomeScreen.vue'
import { useTabsStore } from '@/stores/tabs'
import { useFileOpener } from '@/composables/useFileOpener'

const tabsStore = useTabsStore()
const activeTab = computed(() => tabsStore.activeTab)
const gridRef = ref<InstanceType<typeof VirtualGrid> | null>(null)

const { openFromFile } = useFileOpener()
const OPENABLE = /\.(csv|tsv|txt|dat|xlsx|xls)$/i

// depth counter so nested dragenter/leave on children don't flicker the overlay
const dragDepth = ref(0)
function dragHasFiles(e: DragEvent) {
  return Array.from(e.dataTransfer?.types || []).includes('Files')
}
function onDragEnter(e: DragEvent) {
  if (dragHasFiles(e)) dragDepth.value++
}
function onDragLeave() {
  dragDepth.value = Math.max(0, dragDepth.value - 1)
}
async function onDrop(e: DragEvent) {
  dragDepth.value = 0
  const files = e.dataTransfer?.files
  if (!files) return
  for (const f of Array.from(files)) {
    if (OPENABLE.test(f.name)) await openFromFile(f)
  }
}
</script>

<style scoped>
.app-layout {
  height: 100%;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  position: relative;
}
.app-body {
  flex: 1;
  overflow: hidden;
  display: flex;
}
.drop-overlay {
  position: absolute;
  inset: 0;
  z-index: 2000;
  display: flex;
  align-items: center;
  justify-content: center;
  background: color-mix(in srgb, var(--accent) 14%, rgba(0, 0, 0, 0.35));
  backdrop-filter: blur(2px);
  pointer-events: none;
}
.drop-hint {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 12px;
  padding: 32px 48px;
  border: 2px dashed white;
  border-radius: var(--radius);
  color: white;
  font-size: 15px;
  font-weight: 600;
  background: color-mix(in srgb, var(--accent) 50%, transparent);
}
</style>
