import client from './client'
import type { SortKey, FilterGroup, Cell, AggregateResult, FindMatch } from '@/types'

export const dataApi = {
  sort(id: string, keys: SortKey[]) {
    return client.post(`/files/${id}/sort`, { keys }).then(r => r.data)
  },

  filter(id: string, group: FilterGroup) {
    return client.post(`/files/${id}/filter`, { group }).then(r => r.data as {
      matchCount: number
      matchIndices: number[]
      rows: string[][]
    })
  },

  find(id: string, query: string, options?: { regex?: boolean; caseSensitive?: boolean; colIndex?: number }) {
    return client.post(`/files/${id}/find`, {
      query,
      regex: options?.regex ?? false,
      caseSensitive: options?.caseSensitive ?? false,
      colIndex: options?.colIndex ?? null
    }).then(r => r.data as { matches: FindMatch[]; count: number })
  },

  replace(id: string, query: string, replacement: string, options?: {
    regex?: boolean; caseSensitive?: boolean; all?: boolean; colIndex?: number
  }) {
    return client.post(`/files/${id}/replace`, {
      query,
      replacement,
      regex: options?.regex ?? false,
      caseSensitive: options?.caseSensitive ?? false,
      all: options?.all ?? true,
      colIndex: options?.colIndex ?? null
    }).then(r => r.data as { count: number })
  },

  insertRows(id: string, afterRow: number, count = 1) {
    return client.post(`/files/${id}/rows/insert`, { afterRow, count }).then(r => r.data)
  },

  deleteRows(id: string, rows: number[]) {
    return client.delete(`/files/${id}/rows`, { data: { rows } }).then(r => r.data)
  },

  insertCols(id: string, afterCol: number, count = 1) {
    return client.post(`/files/${id}/columns/insert`, { afterCol, count }).then(r => r.data)
  },

  deleteCols(id: string, cols: number[]) {
    return client.delete(`/files/${id}/columns`, { data: { cols } }).then(r => r.data)
  },

  transform(id: string, cells: Cell[], transformType: string) {
    return client.post(`/files/${id}/transform`, { cells, transform: transformType }).then(r => r.data)
  },

  deduplicate(id: string, colIndexes: number[]) {
    return client.post(`/files/${id}/deduplicate`, { colIndexes }).then(r => r.data)
  },

  aggregate(id: string, cells: Cell[]) {
    return client.post(`/files/${id}/aggregate`, { cells }).then(r => r.data as AggregateResult)
  },

  transpose(id: string) {
    return client.post(`/files/${id}/transpose`).then(r => r.data)
  }
}

export const sqlApi = {
  query(id: string, query: string) {
    return client.post(`/files/${id}/sql`, { query }).then(r => r.data)
  }
}

export const exportApi = {
  toExcel(id: string, filePath: string) {
    return client.post(`/files/${id}/export/excel`, { filePath }).then(r => r.data)
  },

  toFormat(id: string, format: string, cells: Cell[]) {
    return client.post(`/files/${id}/export/format`, { format, cells }).then(r => r.data as { content: string; format: string })
  }
}
