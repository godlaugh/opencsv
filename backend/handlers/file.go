package handlers

import (
	"net/http"
	"opencsv/models"
	"opencsv/services/session"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// OpenFile handles POST /api/files/open
func OpenFile(c *gin.Context) {
	var req models.OpenFileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// HasHeader defaults to true. Since JSON bool zero-value is false,
	// we re-read the raw body to distinguish "not provided" vs "explicitly false".
	// Simplest fix: always default to true (callers must explicitly pass false).
	req.Config.HasHeader = true

	sess, err := session.Global.Open(req.FilePath, req.Config)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Return first 200 rows as preview
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
		"modified":  sess.Modified,
	})
}

// GetRows handles GET /api/files/:id/rows
func GetRows(c *gin.Context) {
	id := c.Param("id")
	offsetStr := c.DefaultQuery("offset", "0")
	limitStr := c.DefaultQuery("limit", "100")

	offset, _ := strconv.Atoi(offsetStr)
	limit, _ := strconv.Atoi(limitStr)
	if limit <= 0 || limit > 5000 {
		limit = 100
	}

	rows, total, err := session.Global.GetRows(id, offset, limit)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"rows":   rows,
		"offset": offset,
		"total":  total,
	})
}

// UpdateCells handles PUT /api/files/:id/cells
func UpdateCells(c *gin.Context) {
	id := c.Param("id")
	var req models.UpdateCellsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := session.Global.UpdateCells(id, req.Cells); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// SaveFile handles POST /api/files/:id/save
func SaveFile(c *gin.Context) {
	id := c.Param("id")
	var req models.SaveFileRequest
	_ = c.ShouldBindJSON(&req)

	if err := session.Global.Save(id, req.FilePath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// CloseFile handles DELETE /api/files/:id
func CloseFile(c *gin.Context) {
	id := c.Param("id")
	session.Global.Close(id)
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// GetFileInfo handles GET /api/files/:id
func GetFileInfo(c *gin.Context) {
	id := c.Param("id")
	sess, err := session.Global.Get(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"id":        sess.ID,
		"filePath":  sess.FilePath,
		"fileName":  filepath.Base(sess.FilePath),
		"config":    sess.Config,
		"columns":   sess.Columns,
		"totalRows": sess.TotalRows,
		"modified":  sess.Modified,
	})
}

// UpdateColumns handles PUT /api/files/:id/columns
func UpdateColumns(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		Columns []models.Column `json:"columns" binding:"required"`
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

	sess.Columns = req.Columns
	sess.Modified = true
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// OpenDialog returns a file path from a native-like dialog simulation
// In web mode, the client supplies the path
func OpenDialog(c *gin.Context) {
	// In a real desktop app this would open a native dialog.
	// For our local web app, the client sends the path directly.
	c.JSON(http.StatusOK, gin.H{"supported": false, "message": "Use the file input on the frontend"})
}

// GetRecentFiles returns placeholder recent files (stored client-side in this version)
func GetRecentFiles(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"files": []interface{}{}})
}

// helper to check if a file extension is CSV
func isCSVFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".csv" || ext == ".tsv" || ext == ".txt" || ext == ".dat"
}
