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

	sort.SliceStable(sess.Rows, func(i, j int) bool {
		for _, key := range req.Keys {
			a := getCellValue(sess.Rows[i], key.ColIndex)
			b := getCellValue(sess.Rows[j], key.ColIndex)

			cmp := compareValues(a, b, key.Type)
			if cmp != 0 {
				if key.Order == "desc" {
					return cmp > 0
				}
				return cmp < 0
			}
		}
		return false
	})

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

func compareValues(a, b, typ string) int {
	switch typ {
	case "number":
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
	case "date":
		formats := []string{"2006-01-02", "01/02/2006", "02/01/2006", "2006/01/02"}
		for _, f := range formats {
			ta, errA := time.Parse(f, a)
			tb, errB := time.Parse(f, b)
			if errA == nil && errB == nil {
				if ta.Before(tb) {
					return -1
				}
				if ta.After(tb) {
					return 1
				}
				return 0
			}
		}
	case "length":
		la, lb := len([]rune(a)), len([]rune(b))
		if la < lb {
			return -1
		}
		if la > lb {
			return 1
		}
		return 0
	}
	return strings.Compare(a, b)
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
	case "endsWith":
		return strings.HasSuffix(strings.ToLower(val), strings.ToLower(cond.Value))
	case "empty":
		return strings.TrimSpace(val) == ""
	case "notEmpty":
		return strings.TrimSpace(val) != ""
	case "gt":
		fa, errA := strconv.ParseFloat(val, 64)
		fb, errB := strconv.ParseFloat(cond.Value, 64)
		if errA == nil && errB == nil {
			return fa > fb
		}
		return strings.Compare(val, cond.Value) > 0
	case "lt":
		fa, errA := strconv.ParseFloat(val, 64)
		fb, errB := strconv.ParseFloat(cond.Value, 64)
		if errA == nil && errB == nil {
			return fa < fb
		}
		return strings.Compare(val, cond.Value) < 0
	case "regex":
		re, err := regexp.Compile(cond.Value)
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
