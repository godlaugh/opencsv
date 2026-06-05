import client from './client'
import type { CsvConfig, Column, Cell } from '@/types'

export interface OpenResponse {
  id: string
  filePath: string
  fileName: string
  config: CsvConfig
  columns: Column[]
  totalRows: number
  rows: string[][]
  modified: boolean
}

export const fileApi = {
  open(filePath: string, config?: Partial<CsvConfig>): Promise<OpenResponse> {
    return client.post('/files/open', {
      filePath,
      config: {
        delimiter: '',
        quote: '"',
        encoding: 'utf-8',
        hasHeader: true,
        lineEnding: 'LF',
        ...config
      }
    }).then(r => r.data)
  },

  getRows(id: string, offset: number, limit: number) {
    return client.get(`/files/${id}/rows`, { params: { offset, limit } }).then(r => r.data as {
      rows: string[][]
      offset: number
      total: number
    })
  },

  // Fetch rows for an explicit list of full-dataset indices (windowed grid).
  getRowsByIndices(id: string, indices: number[]) {
    return client.post(`/files/${id}/rows/by-indices`, { indices }).then(r => r.data as {
      rows: string[][]
    })
  },

  updateCells(id: string, cells: Cell[]) {
    return client.put(`/files/${id}/cells`, { cells }).then(r => r.data)
  },

  updateColumns(id: string, columns: Column[]) {
    return client.put(`/files/${id}/columns`, { columns }).then(r => r.data)
  },

  save(id: string, filePath?: string) {
    return client.post(`/files/${id}/save`, { filePath: filePath || '' }).then(r => r.data)
  },

  getContent(id: string): Promise<Blob> {
    return client.get(`/files/${id}/content`, { responseType: 'blob' }).then(r => r.data)
  },

  close(id: string) {
    return client.delete(`/files/${id}`).then(r => r.data)
  },

  getInfo(id: string) {
    return client.get(`/files/${id}`).then(r => r.data)
  },

  importExcel(filePath: string) {
    return client.post('/files/import/excel', { filePath, hasHeader: true }).then(r => r.data as OpenResponse)
  }
}
