import { ref, inject } from 'vue'
import { useTabsStore } from '@/stores/tabs'
import { fileApi } from '@/api/file'
import { supportsFileSystemAccess, pickFiles, registerHandle, type FsaFileHandle } from '@/utils/fileSystem'

// Shared open logic for the welcome screen and toolbar. When the File System
// Access API is available, files are opened through the native picker so the
// writable handle can be retained for in-place Cmd+S saves; otherwise it falls
// back to a hidden <input type="file"> upload.
export function useFileOpener() {
  const tabsStore = useTabsStore()
  const notify = inject<(type: string, msg: string) => void>('notify')
  const fileInput = ref<HTMLInputElement | null>(null)

  async function openFromFile(file: File, handle?: FsaFileHandle) {
    try {
      const arrayBuffer = await file.arrayBuffer()
      const bytes = new Uint8Array(arrayBuffer)
      const response = await fetch('/api/files/upload', {
        method: 'POST',
        headers: { 'Content-Type': 'application/octet-stream', 'X-Filename': encodeURIComponent(file.name) },
        body: bytes
      })
      const data = await response.json()
      if (data.error) throw new Error(data.error)

      const isExcel = file.name.endsWith('.xlsx') || file.name.endsWith('.xls')
      const session = isExcel ? await fileApi.importExcel(data.filePath) : await fileApi.open(data.filePath)

      tabsStore.addTab(session, session.rows)
      if (handle) registerHandle(session.id, handle)
      notify?.('success', `Opened ${file.name}`)
    } catch (err: any) {
      notify?.('error', err.message)
    }
  }

  async function openFile() {
    if (supportsFileSystemAccess()) {
      try {
        const picked = await pickFiles()
        for (const { handle, file } of picked) await openFromFile(file, handle)
      } catch (err: any) {
        if (err?.name !== 'AbortError') notify?.('error', err.message || 'Could not open file')
      }
    } else {
      fileInput.value?.click()
    }
  }

  async function onFileInput(e: Event) {
    const files = (e.target as HTMLInputElement).files
    if (!files) return
    for (const file of Array.from(files)) await openFromFile(file)
    ;(e.target as HTMLInputElement).value = ''
  }

  return { fileInput, openFile, onFileInput, openFromFile }
}
