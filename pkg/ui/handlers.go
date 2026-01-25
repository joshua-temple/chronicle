package ui

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// ProjectInfo contains information about the Chronicle project.
type ProjectInfo struct {
	Directory    string     `json:"directory"`
	ConfigFile   string     `json:"config_file"`
	ConfigExists bool       `json:"config_exists"`
	LastModified *time.Time `json:"last_modified,omitempty"`
}

func (s *Server) handleProject(w http.ResponseWriter, _ *http.Request) {
	absDir, err := filepath.Abs(s.dir)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to resolve directory", err)
		return
	}

	configPath := filepath.Join(absDir, "chronicle.yaml")
	info := ProjectInfo{
		Directory:    absDir,
		ConfigFile:   "chronicle.yaml",
		ConfigExists: false,
	}

	if stat, err := os.Stat(configPath); err == nil {
		info.ConfigExists = true
		modTime := stat.ModTime()
		info.LastModified = &modTime
	}

	writeJSON(w, http.StatusOK, info)
}

func writeError(w http.ResponseWriter, status int, message string, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	resp := map[string]string{
		"error":   message,
		"details": "",
	}
	if err != nil {
		resp["details"] = err.Error()
	}
	_ = json.NewEncoder(w).Encode(resp)
}
