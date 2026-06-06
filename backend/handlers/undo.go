package handlers

import (
	"net/http"
	"opencsv/services/session"

	"github.com/gin-gonic/gin"
)

// UndoOp handles POST /api/files/:id/undo
func UndoOp(c *gin.Context) { applyUndoRedo(c, true) }

// RedoOp handles POST /api/files/:id/redo
func RedoOp(c *gin.Context) { applyUndoRedo(c, false) }

func applyUndoRedo(c *gin.Context, isUndo bool) {
	id := c.Param("id")

	var ok bool
	var err error
	if isUndo {
		ok, err = session.Global.Undo(id)
	} else {
		ok, err = session.Global.Redo(id)
	}
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	sess, err := session.Global.Get(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"ok":        ok,
		"columns":   sess.Columns,
		"totalRows": sess.TotalRows,
	})
}
