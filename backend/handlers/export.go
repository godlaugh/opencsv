package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"opencsv/models"
	"opencsv/services/excel"
	"opencsv/services/session"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
)

// ImportExcel handles POST /api/import/excel
func ImportExcel(c *gin.Context) {
	var req struct {
		FilePath  string `json:"filePath" binding:"required"`
		HasHeader bool   `json:"hasHeader"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if !req.HasHeader {
		req.HasHeader = true // default
	}

	headers, rows, err := excel.Import(req.FilePath, req.HasHeader)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Create a session for the imported data
	config := models.CsvConfig{
		Delimiter: ",",
		HasHeader: req.HasHeader,
		Encoding:  "utf-8",
	}

	sess, err := session.Global.Open(req.FilePath, config)
	if err != nil {
		// If the file isn't parseable as CSV, create a synthetic session
		_ = headers
		_ = rows
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create session: " + err.Error()})
		return
	}

	firstRows := sess.Rows
	if len(firstRows) > 200 {
		firstRows = firstRows[:200]
	}

	c.JSON(http.StatusOK, gin.H{
		"id":        sess.ID,
		"filePath":  sess.FilePath,
		"fileName":  filepath.Base(sess.FilePath),
		"config":    sess.Config,
		"columns":   sess.Columns,
		"totalRows": sess.TotalRows,
		"rows":      firstRows,
	})
}

// ExportExcel handles POST /api/files/:id/export/excel
func ExportExcel(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		FilePath string `json:"filePath" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	sess, err := session.Global.Get(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	headers := make([]string, len(sess.Columns))
	for i, col := range sess.Columns {
		headers[i] = col.Name
	}

	if err := excel.Export(req.FilePath, headers, sess.Rows); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// ExportFormat handles POST /api/files/:id/export/format
// Converts selected cells to the requested text format
func ExportFormat(c *gin.Context) {
	id := c.Param("id")
	var req models.ExportFormatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	sess, err := session.Global.Get(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	var rows [][]string
	var headers []string

	if len(req.Cells) == 0 {
		// Empty selection = export entire file
		rows = sess.Rows
		headers = make([]string, len(sess.Columns))
		for i, col := range sess.Columns {
			headers[i] = col.Name
		}
	} else {
		// Build a 2D selection map
		type cellKey struct{ r, c int }
		selection := make(map[cellKey]string)
		minRow, maxRow, minCol, maxCol := 1<<30, -1, 1<<30, -1
		for _, cell := range req.Cells {
			if cell.Row < len(sess.Rows) {
				selection[cellKey{cell.Row, cell.Col}] = getCellValue(sess.Rows[cell.Row], cell.Col)
			}
			if cell.Row < minRow {
				minRow = cell.Row
			}
			if cell.Row > maxRow {
				maxRow = cell.Row
			}
			if cell.Col < minCol {
				minCol = cell.Col
			}
			if cell.Col > maxCol {
				maxCol = cell.Col
			}
		}

		if maxRow == -1 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "no cells selected"})
			return
		}

		rows = make([][]string, maxRow-minRow+1)
		for ri := range rows {
			row := make([]string, maxCol-minCol+1)
			for ci := range row {
				row[ci] = selection[cellKey{minRow + ri, minCol + ci}]
			}
			rows[ri] = row
		}

		headers = make([]string, maxCol-minCol+1)
		for ci := range headers {
			colIdx := minCol + ci
			if colIdx < len(sess.Columns) {
				headers[ci] = sess.Columns[colIdx].Name
			} else {
				headers[ci] = fmt.Sprintf("Col %d", colIdx+1)
			}
		}
	}

	var output string
	switch req.Format {
	case "markdown":
		output = toMarkdown(headers, rows)
	case "html":
		output = toHTML(headers, rows)
	case "json":
		output = toJSON(headers, rows)
	case "sql":
		output = toSQL("table", headers, rows)
	case "latex":
		output = toLaTeX(headers, rows)
	case "csv":
		output = toCSV(headers, rows)
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "unknown format: " + req.Format})
		return
	}

	c.JSON(http.StatusOK, gin.H{"content": output, "format": req.Format})
}

// --- format helpers ---

func toMarkdown(headers []string, rows [][]string) string {
	var sb strings.Builder
	sep := make([]string, len(headers))
	for i := range sep {
		sep[i] = "---"
	}
	sb.WriteString("| " + strings.Join(headers, " | ") + " |\n")
	sb.WriteString("| " + strings.Join(sep, " | ") + " |\n")
	for _, row := range rows {
		sb.WriteString("| " + strings.Join(row, " | ") + " |\n")
	}
	return sb.String()
}

func toHTML(headers []string, rows [][]string) string {
	var sb strings.Builder
	sb.WriteString("<table>\n<thead><tr>")
	for _, h := range headers {
		sb.WriteString("<th>" + escapeHTML(h) + "</th>")
	}
	sb.WriteString("</tr></thead>\n<tbody>\n")
	for _, row := range rows {
		sb.WriteString("<tr>")
		for _, cell := range row {
			sb.WriteString("<td>" + escapeHTML(cell) + "</td>")
		}
		sb.WriteString("</tr>\n")
	}
	sb.WriteString("</tbody>\n</table>")
	return sb.String()
}

func toJSON(headers []string, rows [][]string) string {
	result := make([]map[string]string, len(rows))
	for i, row := range rows {
		obj := make(map[string]string)
		for j, h := range headers {
			if j < len(row) {
				obj[h] = row[j]
			}
		}
		result[i] = obj
	}
	b, _ := json.MarshalIndent(result, "", "  ")
	return string(b)
}

func toSQL(table string, headers []string, rows [][]string) string {
	var sb strings.Builder
	quotedCols := make([]string, len(headers))
	for i, h := range headers {
		quotedCols[i] = `"` + h + `"`
	}
	for _, row := range rows {
		vals := make([]string, len(headers))
		for i := range vals {
			if i < len(row) {
				vals[i] = "'" + strings.ReplaceAll(row[i], "'", "''") + "'"
			} else {
				vals[i] = "NULL"
			}
		}
		sb.WriteString(fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s);\n",
			table,
			strings.Join(quotedCols, ", "),
			strings.Join(vals, ", "),
		))
	}
	return sb.String()
}

func toLaTeX(headers []string, rows [][]string) string {
	var sb strings.Builder
	cols := strings.Repeat("l", len(headers))
	sb.WriteString(fmt.Sprintf("\\begin{tabular}{%s}\n\\hline\n", cols))
	sb.WriteString(strings.Join(headers, " & ") + " \\\\\n\\hline\n")
	for _, row := range rows {
		sb.WriteString(strings.Join(row, " & ") + " \\\\\n")
	}
	sb.WriteString("\\hline\n\\end{tabular}")
	return sb.String()
}

func toCSV(headers []string, rows [][]string) string {
	var sb strings.Builder
	sb.WriteString(strings.Join(headers, ",") + "\n")
	for _, row := range rows {
		sb.WriteString(strings.Join(row, ",") + "\n")
	}
	return sb.String()
}

func escapeHTML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}
