package main

import (
	"embed"
	"fmt"
	"io/fs"
	"log"
	"net"
	"net/http"
	"opencsv/handlers"
	"opencsv/middleware"
	"os"
	"os/exec"
	"runtime"

	"github.com/gin-gonic/gin"
)

//go:embed dist
var distFS embed.FS

func main() {
	port := "7070"
	if p := os.Getenv("PORT"); p != "" {
		port = p
	}

	// Check if in dev mode (no dist folder or DEV=1)
	devMode := os.Getenv("DEV") == "1"

	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()
	r.Use(middleware.CORS())

	// API routes
	api := r.Group("/api")
	{
		files := api.Group("/files")
		{
			files.POST("/open", handlers.OpenFile)
			files.GET("/recent", handlers.GetRecentFiles)
			files.POST("/upload", handlers.UploadFile)
			files.POST("/import/excel", handlers.ImportExcel)

			file := files.Group("/:id")
			{
				file.GET("", handlers.GetFileInfo)
				file.GET("/rows", handlers.GetRows)
				file.POST("/rows/by-indices", handlers.GetRowsByIndices)
				file.GET("/content", handlers.GetContent)
				file.PUT("/cells", handlers.UpdateCells)
				file.PUT("/columns", handlers.UpdateColumns)
				file.POST("/save", handlers.SaveFile)
				file.DELETE("", handlers.CloseFile)

				// Data operations
				file.POST("/sort", handlers.SortData)
				file.POST("/filter", handlers.FilterData)
				file.POST("/find", handlers.FindInData)
				file.POST("/replace", handlers.ReplaceInData)
				file.POST("/rows/insert", handlers.InsertRows)
				file.DELETE("/rows", handlers.DeleteRows)
				file.POST("/columns/insert", handlers.InsertCols)
				file.DELETE("/columns", handlers.DeleteCols)
				file.GET("/columns/:colIndex/values", handlers.GetColumnValues)
				file.POST("/transform", handlers.TransformData)
				file.POST("/deduplicate", handlers.Deduplicate)
				file.POST("/aggregate", handlers.Aggregate)
				file.POST("/transpose", handlers.Transpose)

				// SQL
				file.POST("/sql", handlers.ExecuteSQL)

				// Export
				file.POST("/export/excel", handlers.ExportExcel)
				file.POST("/export/format", handlers.ExportFormat)
			}
		}
	}

	// Serve frontend
	if devMode {
		// In dev mode, proxy to Vite dev server
		log.Println("Dev mode: frontend served by Vite at :5173")
		r.NoRoute(func(c *gin.Context) {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		})
	} else {
		// Serve embedded Vue build
		subFS, err := fs.Sub(distFS, "dist")
		if err != nil {
			log.Printf("Warning: cannot serve dist: %v", err)
		} else {
			r.NoRoute(func(c *gin.Context) {
				path := c.Request.URL.Path
				// Try to serve the file, fallback to index.html for SPA routing
				if _, err := subFS.Open(path[1:]); err != nil {
					c.FileFromFS("index.html", http.FS(subFS))
				} else {
					c.FileFromFS(path[1:], http.FS(subFS))
				}
			})
		}
	}

	addr := fmt.Sprintf("localhost:%s", port)
	url := fmt.Sprintf("http://%s", addr)

	// Try to open browser
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("Cannot bind to %s: %v", addr, err)
	}

	log.Printf("OpenCSV server running at %s", url)
	if !devMode {
		go openBrowser(url)
	}

	if err := http.Serve(ln, r); err != nil {
		log.Fatal(err)
	}
}

func openBrowser(url string) {
	var cmd string
	var args []string
	switch runtime.GOOS {
	case "darwin":
		cmd = "open"
		args = []string{url}
	case "linux":
		cmd = "xdg-open"
		args = []string{url}
	case "windows":
		cmd = "cmd"
		args = []string{"/c", "start", url}
	default:
		return
	}
	_ = exec.Command(cmd, args...).Start()
}
