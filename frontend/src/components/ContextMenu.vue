<template>
  <Teleport to="body">
    <div class="ctx-backdrop" @mousedown.self="emit('close')" @contextmenu.prevent />
    <div class="context-menu" :style="{ left: safeX + 'px', top: safeY + 'px' }" ref="menuRef">
      <template v-for="item in items" :key="item.id">
        <div v-if="item.id.startsWith('sep')" class="context-menu-sep" />
        <div
          v-else
          class="context-menu-item"
          :class="{ danger: item.danger }"
          @click="onSelect(item.id)"
        >
          <span>{{ item.label }}</span>
          <span v-if="item.shortcut" class="context-menu-shortcut">{{ item.shortcut }}</span>
        </div>
      </template>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'

const props = defineProps<{
  items: { id: string; label: string; danger?: boolean; shortcut?: string }[]
  x: number
  y: number
}>()
const emit = defineEmits<{ close: []; select: [id: string] }>()

const menuRef = ref<HTMLElement | null>(null)
const safeX = computed(() => Math.min(props.x, window.innerWidth - 200))
const safeY = computed(() => Math.min(props.y, window.innerHeight - 200))

function onSelect(id: string) {
  emit('select', id)
  emit('close')
}

onMounted(() => {
  menuRef.value?.focus()
})
</script>

<style scoped>
.ctx-backdrop {
  position: fixed;
  inset: 0;
  z-index: 1999;
}
</style>
