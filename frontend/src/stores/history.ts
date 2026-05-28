import { defineStore } from 'pinia'
import { ref } from 'vue'
import type { HistoryEntry } from '@/types'

export const useHistoryStore = defineStore('history', () => {
  // Per-file undo stacks
  const stacks = ref<Map<string, HistoryEntry[]>>(new Map())
  const pointers = ref<Map<string, number>>(new Map())

  function push(fileId: string, entry: HistoryEntry) {
    if (!stacks.value.has(fileId)) {
      stacks.value.set(fileId, [])
      pointers.value.set(fileId, -1)
    }
    const stack = stacks.value.get(fileId)!
    const ptr = pointers.value.get(fileId)!
    // Truncate future history
    stack.splice(ptr + 1)
    stack.push(entry)
    if (stack.length > 100) stack.shift()
    pointers.value.set(fileId, stack.length - 1)
  }

  function undo(fileId: string): HistoryEntry | null {
    const stack = stacks.value.get(fileId)
    const ptr = pointers.value.get(fileId) ?? -1
    if (!stack || ptr < 0) return null
    const entry = stack[ptr]
    pointers.value.set(fileId, ptr - 1)
    return entry
  }

  function redo(fileId: string): HistoryEntry | null {
    const stack = stacks.value.get(fileId)
    const ptr = pointers.value.get(fileId) ?? -1
    if (!stack || ptr >= stack.length - 1) return null
    pointers.value.set(fileId, ptr + 1)
    return stack[ptr + 1]
  }

  function canUndo(fileId: string): boolean {
    return (pointers.value.get(fileId) ?? -1) >= 0
  }

  function canRedo(fileId: string): boolean {
    const stack = stacks.value.get(fileId)
    const ptr = pointers.value.get(fileId) ?? -1
    return !!stack && ptr < stack.length - 1
  }

  function clear(fileId: string) {
    stacks.value.delete(fileId)
    pointers.value.delete(fileId)
  }

  return { push, undo, redo, canUndo, canRedo, clear }
})
