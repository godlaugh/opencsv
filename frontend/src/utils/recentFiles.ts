import type { FsaFileHandle } from './fileSystem'

// Recently-opened files. Metadata lives in localStorage; the File System Access
// handle (when available) lives in IndexedDB so the file can be reopened in
// place across reloads (Chromium). Browsers without FSA keep the metadata but
// can't auto-reopen — clicking falls back to the open picker.

export interface RecentFile {
  key: string
  name: string
  size: number
  lastOpened: number
  hasHandle: boolean
}

const LS_KEY = 'opencsv:recent'
const MAX = 12

export function listRecent(): RecentFile[] {
  try {
    const arr = JSON.parse(localStorage.getItem(LS_KEY) || '[]')
    return Array.isArray(arr) ? arr : []
  } catch {
    return []
  }
}

function saveList(list: RecentFile[]) {
  try {
    localStorage.setItem(LS_KEY, JSON.stringify(list.slice(0, MAX)))
  } catch {}
}

export async function addRecent(name: string, size: number, handle?: FsaFileHandle) {
  const key = `${name}::${size}`
  const list = listRecent().filter(r => r.key !== key)
  list.unshift({ key, name, size, lastOpened: Date.now(), hasHandle: !!handle })
  saveList(list)
  if (handle) await putHandle(key, handle)
}

export function removeRecent(key: string) {
  saveList(listRecent().filter(r => r.key !== key))
  delHandle(key)
}

export function clearRecent() {
  for (const r of listRecent()) delHandle(r.key)
  saveList([])
}

// --- IndexedDB handle store ---
const DB = 'opencsv'
const STORE = 'handles'
let dbp: Promise<IDBDatabase> | null = null

function idb(): Promise<IDBDatabase> {
  if (dbp) return dbp
  dbp = new Promise((resolve, reject) => {
    const req = indexedDB.open(DB, 1)
    req.onupgradeneeded = () => req.result.createObjectStore(STORE)
    req.onsuccess = () => resolve(req.result)
    req.onerror = () => reject(req.error)
  })
  return dbp
}

async function putHandle(key: string, handle: FsaFileHandle) {
  try {
    const db = await idb()
    await new Promise<void>((res, rej) => {
      const tx = db.transaction(STORE, 'readwrite')
      tx.objectStore(STORE).put(handle, key)
      tx.oncomplete = () => res()
      tx.onerror = () => rej(tx.error)
    })
  } catch {}
}

export async function getRecentHandle(key: string): Promise<FsaFileHandle | undefined> {
  try {
    const db = await idb()
    return await new Promise(res => {
      const r = db.transaction(STORE, 'readonly').objectStore(STORE).get(key)
      r.onsuccess = () => res(r.result as FsaFileHandle | undefined)
      r.onerror = () => res(undefined)
    })
  } catch {
    return undefined
  }
}

async function delHandle(key: string) {
  try {
    const db = await idb()
    db.transaction(STORE, 'readwrite').objectStore(STORE).delete(key)
  } catch {}
}
