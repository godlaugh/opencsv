package handlers

import (
	"net/http"
	"opencsv/models"
	"opencsv/services/session"
	sqlengine "opencsv/services/sql"

	"github.com/gin-gonic/gin"
)

// ExecuteSQL handles POST /api/files/:id/sql
func ExecuteSQL(c *gin.Context) {
	id := c.Param("id")
	var req models.SqlQueryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	sess, err := session.Global.Get(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	columns, rows, err := sqlengine.Execute(sess, req.Query)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Convert columns to Column slice
	cols := make([]models.Column, len(columns))
	for i, name := range columns {
		cols[i] = models.Column{Index: i, Name: name}
	}

	c.JSON(http.StatusOK, gin.H{
		"columns":   cols,
		"rows":      rows,
		"totalRows": len(rows),
	})
}
