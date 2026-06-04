import { fileApi } from '@/api/file'

// Minimal File System Access API surface — declared locally so the build
// doesn't depend on the API being present in the TS DOM lib.
export interface FsaFileHandle {
  name: string
  getFile(): Promise<File>
  createWritable(): Promise<FsaWritable>
  queryPermission(opts: { mode: 'read' | 'readwrite' }): Promise<PermissionState>
  requestPermission(opts: { mode: 'read' | 'readwrite' }): Promise<PermissionState>
}

interface FsaWritable {
  write(data: Blob | BufferSource | string): Promise<void>
  close(): Promise<void>
}

// Handles are live browser objects that can't round-trip through the backend,
// so we keep them in a module-level registry keyed by tab/session id.
const handles = new Map<string, FsaFileHandle>()

export function supportsFileSystemAccess(): boolean {
  return typeof window !== 'undefined' && 'showOpenFilePicker' in window
}

export function registerHandle(id: string, handle: FsaFileHandle) {
  handles.set(id, handle)
}

export function getHandle(id: string): FsaFileHandle | undefined {
  return handles.get(id)
}

export function clearHandle(id: string) {
  handles.delete(id)
}

const ACCEPT_TYPES = [
  {
    description: 'Spreadsheets',
    accept: {
      'text/csv': ['.csv'],
      'text/tab-separated-values': ['.tsv'],
      'text/plain': ['.txt', '.dat'],
      'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet': ['.xlsx']
    }
  }
]

// Opens the native file picker and returns each picked file along with its
// writable handle. Throws AbortError if the user cancels.
export async function pickFiles(): Promise<{ handle: FsaFileHandle; file: File }[]> {
  const picker = (window as any).showOpenFilePicker as (
    opts: unknown
  ) => Promise<FsaFileHandle[]>
  const fileHandles = await picker({ multiple: true, types: ACCEPT_TYPES })
  const result: { handle: FsaFileHandle; file: File }[] = []
  for (const handle of fileHandles) {
    result.push({ handle, file: await handle.getFile() })
  }
  return result
}

async function ensureWritePermission(handle: FsaFileHandle): Promise<boolean> {
  const opts = { mode: 'readwrite' as const }
  if ((await handle.queryPermission(opts)) === 'granted') return true
  return (await handle.requestPermission(opts)) === 'granted'
}

// Saves the session. If a File System Access handle is registered, the current
// data is written back to the original file in place ('disk'). Otherwise it
// falls back to the server-side save against the session's path ('server').
export async function saveSession(id: string): Promise<'disk' | 'server'> {
  const handle = handles.get(id)
  if (handle && supportsFileSystemAccess()) {
    // Request permission first, while still inside the user gesture.
    if (!(await ensureWritePermission(handle))) {
      throw new Error('Write permission denied')
    }
    const blob = await fileApi.getContent(id)
    const writable = await handle.createWritable()
    await writable.write(blob)
    await writable.close()
    return 'disk'
  }
  await fileApi.save(id)
  return 'server'
}
