package models

// OpenFileRequest is the payload for opening a file
type OpenFileRequest struct {
	FilePath string    `json:"filePath" binding:"required"`
	Config   CsvConfig `json:"config"`
}

// SaveFileRequest is the payload for saving
type SaveFileRequest struct {
	FilePath string    `json:"filePath"` // empty = save to original path
	Config   CsvConfig `json:"config"`
}

// UpdateCellsRequest is the payload for bulk cell updates
type UpdateCellsRequest struct {
	Cells []Cell `json:"cells" binding:"required"`
}

// SortRequest defines a sort operation
type SortRequest struct {
	Keys []SortKey `json:"keys" binding:"required"`
}

// FilterRequest defines a filter operation
type FilterRequest struct {
	Group FilterGroup `json:"group" binding:"required"`
}

// FindRequest defines a find operation
type FindRequest struct {
	Query       string `json:"query" binding:"required"`
	Regex       bool   `json:"regex"`
	CaseSensitive bool `json:"caseSensitive"`
	ColIndex    *int   `json:"colIndex"` // nil = search all
}

// ReplaceRequest defines a find-and-replace operation
type ReplaceRequest struct {
	Query       string `json:"query" binding:"required"`
	Replacement string `json:"replacement"`
	Regex       bool   `json:"regex"`
	CaseSensitive bool `json:"caseSensitive"`
	All         bool   `json:"all"`
	ColIndex    *int   `json:"colIndex"`
}

// InsertRowsRequest defines row insertion
type InsertRowsRequest struct {
	AfterRow int `json:"afterRow"` // -1 = insert at top
	Count    int `json:"count"`
}

// DeleteRowsRequest defines row deletion
type DeleteRowsRequest struct {
	Rows []int `json:"rows" binding:"required"`
}

// InsertColsRequest defines column insertion
type InsertColsRequest struct {
	AfterCol int `json:"afterCol"` // -1 = insert at left
	Count    int `json:"count"`
}

// DeleteColsRequest defines column deletion
type DeleteColsRequest struct {
	Cols []int `json:"cols" binding:"required"`
}

// TransformRequest defines a text transform operation
type TransformRequest struct {
	Cells     []Cell `json:"cells" binding:"required"` // target cells
	Transform string `json:"transform" binding:"required"` // "upper","lower","title","snake","camel","trim","ltrim","rtrim"
}

// DeduplicateRequest defines a dedup operation
type DeduplicateRequest struct {
	ColIndexes []int `json:"colIndexes" binding:"required"`
}

// AggregateRequest defines an aggregate operation
type AggregateRequest struct {
	Cells []Cell `json:"cells" binding:"required"`
}

// SqlQueryRequest defines a SQL query
type SqlQueryRequest struct {
	Query string `json:"query" binding:"required"`
}

// ExportFormatRequest defines a copy-as-format operation
type ExportFormatRequest struct {
	Format string `json:"format" binding:"required"` // "markdown","html","json","sql","latex","csv"
	Cells  []Cell `json:"cells" binding:"required"`
}

// FindMatch is one find result
type FindMatch struct {
	Row int `json:"row"`
	Col int `json:"col"`
}

// PagedRowsResponse is the paginated row data response
type PagedRowsResponse struct {
	Rows   [][]string `json:"rows"`
	Offset int        `json:"offset"`
	Total  int        `json:"total"`
}
