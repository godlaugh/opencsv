export interface CsvConfig {
  delimiter: string
  quote: string
  encoding: string
  hasHeader: boolean
  lineEnding: string
}

export interface Column {
  index: number
  name: string
}

export interface Cell {
  row: number
  col: number
  value: string
}

export interface FileSession {
  id: string
  filePath: string
  fileName: string
  config: CsvConfig
  columns: Column[]
  totalRows: number
  modified: boolean
}

export interface Tab {
  id: string
  session: FileSession
  rows: string[][]
  cachedPages: Map<number, string[][]>
  loading: boolean
  filterActive: boolean
  filteredIndices: number[] | null
  /**
   * Single source of truth for all filtering. Both the per-column header
   * dropdown and the global Filter dialog read from and write to this group,
   * so they stay in sync (SmoothCSV-style).
   */
  filterGroup?: FilterGroup | null
}

export interface SortKey {
  colIndex: number
  order: 'asc' | 'desc'
  type: 'text' | 'number' | 'date' | 'length'
}

export interface FilterCondition {
  colIndex: number
  operator: string
  value: string
  values?: string[]
}

// Per-column quick-filter state (Excel-style header dropdown)
export interface ColumnQuickFilter {
  mode: 'values' | 'condition'
  // 'values' mode: only rows where the column value is in this set are kept.
  // When undefined or covers all distinct values, the filter is inactive.
  selectedValues?: string[]
  // 'condition' mode
  operator?: string
  value?: string
}

export interface FilterGroup {
  logic: 'AND' | 'OR'
  conditions: FilterCondition[]
}

export interface FindMatch {
  row: number
  col: number
}

export interface Selection {
  startRow: number
  startCol: number
  endRow: number
  endCol: number
}

export interface AggregateResult {
  count: number
  sum: number
  avg: number
  min: string
  max: string
  empty: number
  unique: number
}

export interface SqlResult {
  columns: Column[]
  rows: string[][]
  totalRows: number
}

export interface HistoryEntry {
  type: string
  before: { cells: Cell[] }
  after: { cells: Cell[] }
}
