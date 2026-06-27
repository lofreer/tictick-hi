package api

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

func (server *Server) serveFrontend(w http.ResponseWriter, r *http.Request) {
	if server.staticRoot == "" || (r.Method != http.MethodGet && r.Method != http.MethodHead) {
		writeError(w, http.StatusNotFound, "not found")
		return
	}

	requestPath := filepath.Clean(strings.TrimPrefix(r.URL.Path, "/"))
	if requestPath == "." {
		requestPath = "index.html"
	}
	if strings.HasPrefix(requestPath, "..") {
		writeError(w, http.StatusBadRequest, "invalid path")
		return
	}
	fullPath := filepath.Join(server.staticRoot, requestPath)
	if info, err := os.Stat(fullPath); err == nil && !info.IsDir() {
		http.ServeFile(w, r, fullPath)
		return
	}
	http.ServeFile(w, r, filepath.Join(server.staticRoot, "index.html"))
}
