<template>
  <div class="app-layout">
    <!-- Title Bar / Toolbar -->
    <Toolbar />

    <!-- Tab Bar -->
    <TabBar />

    <!-- Main Content -->
    <div class="app-body">
      <template v-if="activeTab">
        <VirtualGrid :key="activeTab.id" :tab="activeTab" />
      </template>
      <template v-else>
        <WelcomeScreen />
      </template>
    </div>

    <!-- Status Bar -->
    <StatusBar />
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import Toolbar from './Toolbar.vue'
import TabBar from './TabBar.vue'
import StatusBar from './StatusBar.vue'
import VirtualGrid from '@/components/grid/VirtualGrid.vue'
import WelcomeScreen from './WelcomeScreen.vue'
import { useTabsStore } from '@/stores/tabs'

const tabsStore = useTabsStore()
const activeTab = computed(() => tabsStore.activeTab)
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
