package handlers

import (
	"fmt"
	"math"
	"net/http"
	"opencsv/models"
	"opencsv/services/session"
	"opencsv/services/transform"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// SortData handles POST /api/files/:id/sort
func SortData(c *gin.Context) {
	id := c.Param("id")
	var req models.SortRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	sess, err := session.Global.Get(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	n := len(sess.Rows)

	// Decorate-sort-undecorate: pre-parse each sort-key column ONCE into typed
	// comparable arrays, then sort an index slice. This avoids re-parsing
	// numbers/dates on every one of the O(n log n) comparisons (which made
	// date sorts on 1M rows take ~10s).
	type sortKeyData struct {
		key     models.SortKey
		numeric bool      // compare via nums[]
		nums    []float64 // number/date/length keys (NaN = unparseable)
	}
	keysData := make([]sortKeyData, len(req.Keys))
	for ki, key := range req.Keys {
		kd := sortKeyData{key: key}
		switch key.Type {
		case "number":
			kd.numeric = true
			kd.nums = make([]float64, n)
			for i := range sess.Rows {
				f, err := strconv.ParseFloat(getCellValue(sess.Rows[i], key.ColIndex), 64)
				if err != nil {
					f = math.NaN()
				}
				kd.nums[i] = f
			}
		case "date":
			kd.numeric = true
			kd.nums = make([]float64, n)
			for i := range sess.Rows {
				if ts, ok := parseDate(getCellValue(sess.Rows[i], key.ColIndex)); ok {
					kd.nums[i] = float64(ts)
				} else {
					kd.nums[i] = math.NaN()
				}
			}
		case "length":
			kd.numeric = true
			kd.nums = make([]float64, n)
			for i := range sess.Rows {
				kd.nums[i] = float64(len([]rune(getCellValue(sess.Rows[i], key.ColIndex))))
			}
		}
		keysData[ki] = kd
	}

	idx := make([]int, n)
	for i := range idx {
		idx[i] = i
	}

	sort.SliceStable(idx, func(a, b int) bool {
		ia, ib := idx[a], idx[b]
		for _, kd := range keysData {
			var cmp int
			if kd.numeric {
				na, nb := kd.nums[ia], kd.nums[ib]
				aNaN, bNaN := math.IsNaN(na), math.IsNaN(nb)
				switch {
				case aNaN && bNaN:
					// both unparseable: fall back to string compare
					cmp = strings.Compare(
						getCellValue(sess.Rows[ia], kd.key.ColIndex),
						getCellValue(sess.Rows[ib], kd.key.ColIndex))
				case aNaN:
					cmp = 1 // unparseable sorts last
				case bNaN:
					cmp = -1
				case na < nb:
					cmp = -1
				case na > nb:
					cmp = 1
				}
			} else {
				cmp = strings.Compare(
					getCellValue(sess.Rows[ia], kd.key.ColIndex),
					getCellValue(sess.Rows[ib], kd.key.ColIndex))
			}
			if cmp != 0 {
				if kd.key.Order == "desc" {
					return cmp > 0
				}
				return cmp < 0
			}
		}
		return false
	})

	// Reorder rows according to the sorted index slice.
	newRows := make([][]string, n)
	for newPos, oldPos := range idx {
		newRows[newPos] = sess.Rows[oldPos]
	}
	sess.Rows = newRows

	sess.Modified = true
	c.JSON(http.StatusOK, gin.H{"ok": true, "totalRows": sess.TotalRows})
}

// FilterData handles POST /api/files/:id/filter
// Returns filtered row indices rather than modifying in place
func FilterData(c *gin.Context) {
	id := c.Param("id")
	var req models.FilterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	sess, err := session.Global.Get(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	var matchIndices []int
	for i, row := range sess.Rows {
		if matchesGroup(row, req.Group) {
			matchIndices = append(matchIndices, i)
		}
	}

	// Return matching row indices and the first 200 matching rows
	var previewRows [][]string
	for _, idx := range matchIndices {
		if len(previewRows) >= 200 {
			break
		}
		previewRows = append(previewRows, sess.Rows[idx])
	}

	c.JSON(http.StatusOK, gin.H{
		"matchCount":  len(matchIndices),
		"matchIndices": matchIndices,
		"rows":        previewRows,
	})
}

// FindInData handles POST /api/files/:id/find
func FindInData(c *gin.Context) {
	id := c.Param("id")
	var req models.FindRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	sess, err := session.Global.Get(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	var matches []models.FindMatch

	var matchFunc func(s string) bool
	if req.Regex {
		flags := ""
		if !req.CaseSensitive {
			flags = "(?i)"
		}
		re, err := regexp.Compile(flags + req.Query)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid regex: " + err.Error()})
			return
		}
		matchFunc = re.MatchString
	} else {
		query := req.Query
		if !req.CaseSensitive {
			query = strings.ToLower(query)
		}
		matchFunc = func(s string) bool {
			if !req.CaseSensitive {
				s = strings.ToLower(s)
			}
			return strings.Contains(s, query)
		}
	}

	for ri, row := range sess.Rows {
		for ci, cell := range row {
			if req.ColIndex != nil && ci != *req.ColIndex {
				continue
			}
			if matchFunc(cell) {
				matches = append(matches, models.FindMatch{Row: ri, Col: ci})
				if len(matches) >= 10000 { // cap results
					goto done
				}
			}
		}
	}
done:
	c.JSON(http.StatusOK, gin.H{"matches": matches, "count": len(matches)})
}

// ReplaceInData handles POST /api/files/:id/replace
func ReplaceInData(c *gin.Context) {
	id := c.Param("id")
	var req models.ReplaceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	sess, err := session.Global.Get(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	count := 0

	var replaceFunc func(s string) (string, bool)
	if req.Regex {
		flags := ""
		if !req.CaseSensitive {
			flags = "(?i)"
		}
		re, err := regexp.Compile(flags + req.Query)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid regex: " + err.Error()})
			return
		}
		replaceFunc = func(s string) (string, bool) {
			if !re.MatchString(s) {
				return s, false
			}
			result := re.ReplaceAllString(s, req.Replacement)
			return result, true
		}
	} else {
		query := req.Query
		replaceFunc = func(s string) (string, bool) {
			compare := s
			q := query
			if !req.CaseSensitive {
				compare = strings.ToLower(s)
				q = strings.ToLower(query)
			}
			if !strings.Contains(compare, q) {
				return s, false
			}
			return strings.ReplaceAll(s, query, req.Replacement), true
		}
	}

	for ri, row := range sess.Rows {
		for ci, cell := range row {
			if req.ColIndex != nil && ci != *req.ColIndex {
				continue
			}
			newVal, replaced := replaceFunc(cell)
			if replaced {
				row[ci] = newVal
				count++
				if !req.All {
					sess.Rows[ri] = row
					sess.Modified = true
					c.JSON(http.StatusOK, gin.H{"count": 1})
					return
				}
			}
		}
		sess.Rows[ri] = row
	}

	if count > 0 {
		sess.Modified = true
	}

	c.JSON(http.StatusOK, gin.H{"count": count})
}

// InsertRows handles POST /api/files/:id/rows/insert
func InsertRows(c *gin.Context) {
	id := c.Param("id")
	var req models.InsertRowsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Count <= 0 {
		req.Count = 1
	}

	sess, err := session.Global.Get(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	colCount := len(sess.Columns)
	newRows := make([][]string, req.Count)
	for i := range newRows {
		newRows[i] = make([]string, colCount)
	}

	insertAt := req.AfterRow + 1
	if insertAt < 0 {
		insertAt = 0
	}
	if insertAt > len(sess.Rows) {
		insertAt = len(sess.Rows)
	}

	sess.Rows = append(sess.Rows[:insertAt], append(newRows, sess.Rows[insertAt:]...)...)
	sess.TotalRows = len(sess.Rows)
	sess.Modified = true

	c.JSON(http.StatusOK, gin.H{"ok": true, "totalRows": sess.TotalRows})
}

// DeleteRows handles DELETE /api/files/:id/rows
func DeleteRows(c *gin.Context) {
	id := c.Param("id")
	var req models.DeleteRowsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	sess, err := session.Global.Get(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	toDelete := make(map[int]bool)
	for _, r := range req.Rows {
		toDelete[r] = true
	}

	newRows := sess.Rows[:0]
	for i, row := range sess.Rows {
		if !toDelete[i] {
			newRows = append(newRows, row)
		}
	}

	sess.Rows = newRows
	sess.TotalRows = len(sess.Rows)
	sess.Modified = true

	c.JSON(http.StatusOK, gin.H{"ok": true, "totalRows": sess.TotalRows})
}

// InsertCols handles POST /api/files/:id/columns/insert
func InsertCols(c *gin.Context) {
	id := c.Param("id")
	var req models.InsertColsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Count <= 0 {
		req.Count = 1
	}

	sess, err := session.Global.Get(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	insertAt := req.AfterCol + 1
	if insertAt < 0 {
		insertAt = 0
	}

	// Insert columns into header
	newCols := make([]models.Column, req.Count)
	for i := range newCols {
		newCols[i] = models.Column{
			Index: insertAt + i,
			Name:  fmt.Sprintf("Column %d", len(sess.Columns)+i+1),
		}
	}
	sess.Columns = append(sess.Columns[:insertAt], append(newCols, sess.Columns[insertAt:]...)...)
	for i := range sess.Columns {
		sess.Columns[i].Index = i
	}

	// Insert cells into each row
	for ri, row := range sess.Rows {
		newCells := make([]string, req.Count)
		sess.Rows[ri] = append(row[:insertAt], append(newCells, row[insertAt:]...)...)
	}

	sess.Modified = true
	c.JSON(http.StatusOK, gin.H{"ok": true, "columns": sess.Columns})
}

// DeleteCols handles DELETE /api/files/:id/columns
func DeleteCols(c *gin.Context) {
	id := c.Param("id")
	var req models.DeleteColsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	sess, err := session.Global.Get(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	toDelete := make(map[int]bool)
	for _, col := range req.Cols {
		toDelete[col] = true
	}

	newCols := sess.Columns[:0]
	for _, col := range sess.Columns {
		if !toDelete[col.Index] {
			newCols = append(newCols, col)
		}
	}
	sess.Columns = newCols
	for i := range sess.Columns {
		sess.Columns[i].Index = i
	}

	for ri, row := range sess.Rows {
		newRow := row[:0]
		for ci, val := range row {
			if !toDelete[ci] {
				newRow = append(newRow, val)
			}
		}
		sess.Rows[ri] = newRow
	}

	sess.Modified = true
	c.JSON(http.StatusOK, gin.H{"ok": true, "columns": sess.Columns})
}

// TransformData handles POST /api/files/:id/transform
func TransformData(c *gin.Context) {
	id := c.Param("id")
	var req models.TransformRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	sess, err := session.Global.Get(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	for _, cell := range req.Cells {
		if cell.Row >= 0 && cell.Row < len(sess.Rows) {
			row := sess.Rows[cell.Row]
			if cell.Col >= 0 && cell.Col < len(row) {
				row[cell.Col] = transform.Apply(row[cell.Col], req.Transform)
			}
		}
	}

	sess.Modified = true
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// Deduplicate handles POST /api/files/:id/deduplicate
func Deduplicate(c *gin.Context) {
	id := c.Param("id")
	var req models.DeduplicateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	sess, err := session.Global.Get(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	seen := make(map[string]bool)
	newRows := sess.Rows[:0]
	removed := 0

	for _, row := range sess.Rows {
		key := buildKey(row, req.ColIndexes)
		if seen[key] {
			removed++
			continue
		}
		seen[key] = true
		newRows = append(newRows, row)
	}

	sess.Rows = newRows
	sess.TotalRows = len(sess.Rows)
	sess.Modified = true

	c.JSON(http.StatusOK, gin.H{"ok": true, "removed": removed, "totalRows": sess.TotalRows})
}

// Aggregate handles POST /api/files/:id/aggregate
func Aggregate(c *gin.Context) {
	id := c.Param("id")
	var req models.AggregateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	sess, err := session.Global.Get(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	var values []string
	for _, cell := range req.Cells {
		if cell.Row >= 0 && cell.Row < len(sess.Rows) {
			row := sess.Rows[cell.Row]
			if cell.Col >= 0 && cell.Col < len(row) {
				values = append(values, row[cell.Col])
			}
		}
	}

	result := computeAggregate(values)
	c.JSON(http.StatusOK, result)
}

// Transpose handles POST /api/files/:id/transpose
func Transpose(c *gin.Context) {
	id := c.Param("id")

	sess, err := session.Global.Get(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	if len(sess.Rows) == 0 {
		c.JSON(http.StatusOK, gin.H{"ok": true})
		return
	}

	// Find max cols
	maxCols := len(sess.Columns)
	for _, row := range sess.Rows {
		if len(row) > maxCols {
			maxCols = len(row)
		}
	}

	allRows := append([][]string{make([]string, len(sess.Columns))}, sess.Rows...)
	for i, col := range sess.Columns {
		allRows[0][i] = col.Name
	}

	// Transpose
	newCols := len(allRows)
	newRows := maxCols

	transposed := make([][]string, newRows)
	for i := range transposed {
		transposed[i] = make([]string, newCols)
		for j, row := range allRows {
			if i < len(row) {
				transposed[i][j] = row[i]
			}
		}
	}

	// Update session: first row becomes headers
	newHeaders := transposed[0]
	sess.Columns = make([]models.Column, len(newHeaders))
	for i, h := range newHeaders {
		sess.Columns[i] = models.Column{Index: i, Name: h}
	}
	sess.Rows = transposed[1:]
	sess.TotalRows = len(sess.Rows)
	sess.Modified = true

	c.JSON(http.StatusOK, gin.H{"ok": true, "columns": sess.Columns, "totalRows": sess.TotalRows})
}

// --- helpers ---

func getCellValue(row []string, col int) string {
	if col < len(row) {
		return row[col]
	}
	return ""
}

// dateFormats covers common date and datetime layouts (date-only first so a
// bare "2009-12-01" still parses).
var dateFormats = []string{
	"2006-01-02",
	"2006-01-02 15:04:05",
	"2006-01-02T15:04:05",
	"2006-01-02 15:04",
	time.RFC3339,
	"01/02/2006",
	"02/01/2006",
	"2006/01/02",
	"01/02/2006 15:04:05",
	"02/01/2006 15:04:05",
}

// parseDate tries the known layouts and returns a Unix timestamp on success.
func parseDate(s string) (int64, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, false
	}
	for _, f := range dateFormats {
		if t, err := time.Parse(f, s); err == nil {
			return t.Unix(), true
		}
	}
	return 0, false
}

// reCache memoizes compiled regexes (including LIKE-derived patterns) so a
// filter over 1M rows compiles each pattern once instead of per-row.
var reCache sync.Map // pattern string -> reCacheEntry

type reCacheEntry struct {
	re  *regexp.Regexp
	err error
}

func cachedRegex(pattern string) (*regexp.Regexp, error) {
	if v, ok := reCache.Load(pattern); ok {
		e := v.(reCacheEntry)
		return e.re, e.err
	}
	re, err := regexp.Compile(pattern)
	reCache.Store(pattern, reCacheEntry{re, err})
	return re, err
}

func matchesGroup(row []string, group models.FilterGroup) bool {
	if len(group.Conditions) == 0 {
		return true
	}

	if group.Logic == "OR" {
		for _, cond := range group.Conditions {
			if matchesCondition(row, cond) {
				return true
			}
		}
		return false
	}

	// AND (default)
	for _, cond := range group.Conditions {
		if !matchesCondition(row, cond) {
			return false
		}
	}
	return true
}

// numOrStrCompare compares a and b numerically when both parse as numbers,
// otherwise lexicographically. Returns -1, 0, or 1.
func numOrStrCompare(a, b string) int {
	fa, errA := strconv.ParseFloat(a, 64)
	fb, errB := strconv.ParseFloat(b, 64)
	if errA == nil && errB == nil {
		if fa < fb {
			return -1
		}
		if fa > fb {
			return 1
		}
		return 0
	}
	return strings.Compare(a, b)
}

// likeMatch implements SQL LIKE semantics (case-insensitive): % matches any
// sequence, _ matches any single character. Other characters are literal.
func likeMatch(val, pattern string) bool {
	// Fast paths for the common wildcard-only patterns (no '_' and '%' only at
	// the ends): avoid regex entirely and use case-insensitive string ops.
	if !strings.Contains(pattern, "_") {
		inner := strings.Trim(pattern, "%")
		if !strings.Contains(inner, "%") {
			lead := strings.HasPrefix(pattern, "%")
			trail := strings.HasSuffix(pattern, "%")
			lv, li := strings.ToLower(val), strings.ToLower(inner)
			switch {
			case lead && trail:
				return strings.Contains(lv, li) // %x%
			case trail:
				return strings.HasPrefix(lv, li) // x%
			case lead:
				return strings.HasSuffix(lv, li) // %x
			default:
				return lv == li // x (exact, case-insensitive)
			}
		}
	}

	var sb strings.Builder
	sb.WriteString("(?is)^") // case-insensitive, dot-matches-newline, anchored
	for _, r := range pattern {
		switch r {
		case '%':
			sb.WriteString(".*")
		case '_':
			sb.WriteString(".")
		default:
			sb.WriteString(regexp.QuoteMeta(string(r)))
		}
	}
	sb.WriteString("$")
	re, err := cachedRegex(sb.String()) // memoized: compiled once per pattern
	if err != nil {
		return false
	}
	return re.MatchString(val)
}

// betweenMatch reports whether val is within [bounds[0], bounds[1]] inclusive.
// Numeric comparison when all three parse as numbers, otherwise lexicographic.
func betweenMatch(val string, bounds []string) bool {
	if len(bounds) < 2 {
		return false
	}
	lo, hi := bounds[0], bounds[1]
	// Normalize so lo <= hi regardless of input order
	if numOrStrCompare(lo, hi) > 0 {
		lo, hi = hi, lo
	}
	return numOrStrCompare(val, lo) >= 0 && numOrStrCompare(val, hi) <= 0
}

func matchesCondition(row []string, cond models.FilterCondition) bool {
	val := getCellValue(row, cond.ColIndex)

	switch cond.Operator {
	case "eq":
		return strings.EqualFold(val, cond.Value)
	case "ne":
		return !strings.EqualFold(val, cond.Value)
	case "contains":
		return strings.Contains(strings.ToLower(val), strings.ToLower(cond.Value))
	case "notContains":
		return !strings.Contains(strings.ToLower(val), strings.ToLower(cond.Value))
	case "startsWith":
		return strings.HasPrefix(strings.ToLower(val), strings.ToLower(cond.Value))
	case "notStartsWith":
		return !strings.HasPrefix(strings.ToLower(val), strings.ToLower(cond.Value))
	case "endsWith":
		return strings.HasSuffix(strings.ToLower(val), strings.ToLower(cond.Value))
	case "notEndsWith":
		return !strings.HasSuffix(strings.ToLower(val), strings.ToLower(cond.Value))
	case "empty":
		return strings.TrimSpace(val) == ""
	case "notEmpty":
		return strings.TrimSpace(val) != ""
	case "gt":
		return numOrStrCompare(val, cond.Value) > 0
	case "gte":
		return numOrStrCompare(val, cond.Value) >= 0
	case "lt":
		return numOrStrCompare(val, cond.Value) < 0
	case "lte":
		return numOrStrCompare(val, cond.Value) <= 0
	case "like":
		return likeMatch(val, cond.Value)
	case "notLike":
		return !likeMatch(val, cond.Value)
	case "between":
		return betweenMatch(val, cond.Values)
	case "notBetween":
		return !betweenMatch(val, cond.Values)
	case "regex":
		re, err := cachedRegex(cond.Value)
		if err != nil {
			return false
		}
		return re.MatchString(val)
	case "in":
		// Match if cell value (case-sensitive, trimmed) is in Values list.
		// Also supports a sentinel "" entry to match empty cells.
		for _, v := range cond.Values {
			if v == val {
				return true
			}
		}
		return false
	case "notIn":
		for _, v := range cond.Values {
			if v == val {
				return false
			}
		}
		return true
	}
	return false
}

// GetColumnValues handles GET /api/files/:id/columns/:colIndex/values
// Returns distinct values for a column with counts, optional search filter,
// sorted alphabetically. Used by the per-column filter dropdown.
func GetColumnValues(c *gin.Context) {
	id := c.Param("id")
	colStr := c.Param("colIndex")
	colIdx, err := strconv.Atoi(colStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid colIndex"})
		return
	}

	q := strings.ToLower(c.Query("q"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "500"))
	if limit <= 0 || limit > 50000 {
		limit = 500
	}

	sess, err := session.Global.Get(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	counts := make(map[string]int)
	for _, row := range sess.Rows {
		v := ""
		if colIdx < len(row) {
			v = row[colIdx]
		}
		if q != "" && !strings.Contains(strings.ToLower(v), q) {
			continue
		}
		counts[v]++
	}

	keys := make([]string, 0, len(counts))
	for k := range counts {
		keys = append(keys, k)
	}
	sort.SliceStable(keys, func(i, j int) bool {
		// empties last, then case-insensitive alpha
		if keys[i] == "" && keys[j] != "" {
			return false
		}
		if keys[j] == "" && keys[i] != "" {
			return true
		}
		return strings.ToLower(keys[i]) < strings.ToLower(keys[j])
	})

	total := len(keys)
	if len(keys) > limit {
		keys = keys[:limit]
	}

	type valueEntry struct {
		Value string `json:"value"`
		Count int    `json:"count"`
	}
	out := make([]valueEntry, len(keys))
	for i, k := range keys {
		out[i] = valueEntry{Value: k, Count: counts[k]}
	}
	c.JSON(http.StatusOK, gin.H{"values": out, "total": total, "truncated": total > limit})
}

func buildKey(row []string, colIndexes []int) string {
	parts := make([]string, len(colIndexes))
	for i, ci := range colIndexes {
		parts[i] = getCellValue(row, ci)
	}
	return strings.Join(parts, "\x00")
}

func computeAggregate(values []string) models.AggregateResult {
	result := models.AggregateResult{Count: len(values)}
	if len(values) == 0 {
		return result
	}

	seen := make(map[string]bool)
	var nums []float64
	minStr, maxStr := values[0], values[0]

	for _, v := range values {
		if strings.TrimSpace(v) == "" {
			result.Empty++
		}
		seen[v] = true

		if f, err := strconv.ParseFloat(v, 64); err == nil {
			nums = append(nums, f)
			result.Sum += f
		}

		if strings.Compare(v, minStr) < 0 {
			minStr = v
		}
		if strings.Compare(v, maxStr) > 0 {
			maxStr = v
		}
	}

	result.Min = minStr
	result.Max = maxStr
	result.Unique = len(seen)

	if len(nums) > 0 {
		result.Avg = math.Round(result.Sum/float64(len(nums))*100) / 100
	}

	return result
}
