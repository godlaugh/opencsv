<template>
  <div class="app-layout">
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
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import Toolbar from './Toolbar.vue'
import TabBar from './TabBar.vue'
import StatusBar from './StatusBar.vue'
import VirtualGrid from '@/components/grid/VirtualGrid.vue'
import WelcomeScreen from './WelcomeScreen.vue'
import { useTabsStore } from '@/stores/tabs'

const tabsStore = useTabsStore()
const activeTab = computed(() => tabsStore.activeTab)
const gridRef = ref<InstanceType<typeof VirtualGrid> | null>(null)
</script>

<style scoped>
.app-layout {
  height: 100%;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}
.app-body {
  flex: 1;
  overflow: hidden;
  display: flex;
}
</style>
