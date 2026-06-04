package models

// CsvConfig holds the parsing configuration for a CSV file
type CsvConfig struct {
	Delimiter  string `json:"delimiter"`  // default ","
	Quote      string `json:"quote"`      // default "\""
	Encoding   string `json:"encoding"`   // default "utf-8"
	HasHeader  bool   `json:"hasHeader"`  // default true
	LineEnding string `json:"lineEnding"` // "LF", "CRLF", "CR"
}

// Column represents a CSV column definition
type Column struct {
	Index int    `json:"index"`
	Name  string `json:"name"`
}

// Cell represents a single cell value update
type Cell struct {
	Row    int    `json:"row"`
	Col    int    `json:"col"`
	Value  string `json:"value"`
}

// FileSession is the in-memory state of an opened file
type FileSession struct {
	ID       string    `json:"id"`
	FilePath string    `json:"filePath"`
	Config   CsvConfig `json:"config"`
	Columns  []Column  `json:"columns"`
	Rows     [][]string `json:"-"` // full data in memory
	TotalRows int       `json:"totalRows"`
	Modified  bool      `json:"modified"`
}

// SortKey defines one sort key
type SortKey struct {
	ColIndex int    `json:"colIndex"`
	Order    string `json:"order"` // "asc" | "desc"
	Type     string `json:"type"`  // "text" | "number" | "date" | "length"
}

// FilterCondition defines one filter condition
type FilterCondition struct {
	ColIndex int      `json:"colIndex"`
	Operator string   `json:"operator"` // "eq","ne","contains","startsWith","endsWith","gt","lt","regex","empty","notEmpty","in","notIn"
	Value    string   `json:"value"`
	Values   []string `json:"values,omitempty"` // used by "in" / "notIn"
}

// FilterGroup holds a logical group of conditions
type FilterGroup struct {
	Logic      string            `json:"logic"` // "AND" | "OR"
	Conditions []FilterCondition `json:"conditions"`
}

// AggregateResult holds statistics for a selection
type AggregateResult struct {
	Count   int     `json:"count"`
	Sum     float64 `json:"sum"`
	Avg     float64 `json:"avg"`
	Min     string  `json:"min"`
	Max     string  `json:"max"`
	Empty   int     `json:"empty"`
	Unique  int     `json:"unique"`
}
