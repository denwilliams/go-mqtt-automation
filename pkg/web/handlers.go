// Package web provides HTTP handlers and web interface for the home automation system.
package web

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// handleStaticFiles serves static files from the admin-ui directory
// Falls back to index.html for SPA routing (any non-API route)
func (s *Server) handleStaticFiles(w http.ResponseWriter, r *http.Request) {
	// Skip API routes - they should be handled by their specific handlers
	if strings.HasPrefix(r.URL.Path, "/api/") {
		http.NotFound(w, r)
		return
	}

	// Determine the file path to serve
	staticDir := "static"
	requestPath := r.URL.Path

	// Clean the path and remove leading slash
	if requestPath == "/" {
		requestPath = "/index.html"
	}

	filePath := filepath.Join(staticDir, requestPath)

	// Check if the file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		// If file doesn't exist, serve index.html for SPA routing
		indexPath := filepath.Join(staticDir, "index.html")
		if _, indexErr := os.Stat(indexPath); indexErr != nil {
			// If index.html also doesn't exist, return 404
			s.logger.Printf("Static files not found: %s (tried %s and %s)", requestPath, filePath, indexPath)
			http.NotFound(w, r)
			return
		}
		filePath = indexPath
	}

	// Serve the file
	http.ServeFile(w, r, filePath)
}
