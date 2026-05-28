package handlers

import (
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
)

// UploadFile handles POST /api/files/upload
// Receives raw file bytes, writes to a temp file, returns the temp path
func UploadFile(c *gin.Context) {
	filename := c.GetHeader("X-Filename")
	if filename == "" {
		filename = "upload.csv"
	}
	// URL-decode the filename
	decoded, err := url.QueryUnescape(filename)
	if err == nil {
		filename = decoded
	}

	// Sanitize filename
	filename = filepath.Base(filename)

	// Create temp file
	tmpDir := os.TempDir()
	tmpPath := filepath.Join(tmpDir, "opencsv_"+filename)

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot read body: " + err.Error()})
		return
	}

	if err := os.WriteFile(tmpPath, body, 0644); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot write temp file: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"filePath": tmpPath,
		"fileName": filename,
	})
}
