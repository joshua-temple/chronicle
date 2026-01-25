package ui

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/joshua-temple/chronicle/pkg/config"
	"github.com/joshua-temple/chronicle/pkg/discovery"
	"gopkg.in/yaml.v3"
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

func (s *Server) handleGetConfig(w http.ResponseWriter, _ *http.Request) {
	configPath := filepath.Join(s.dir, "chronicle.yaml")

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			writeError(w, http.StatusNotFound, "config file not found", nil)
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to read config", err)
		return
	}

	var cfg config.Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to parse config", err)
		return
	}

	writeJSON(w, http.StatusOK, cfg)
}

func (s *Server) handlePutConfig(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, "failed to read request body", err)
		return
	}

	var cfg config.Config
	if err := json.Unmarshal(body, &cfg); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON", err)
		return
	}

	// Validate before saving
	if err := cfg.Validate(); err != nil {
		writeError(w, http.StatusBadRequest, "validation failed", err)
		return
	}

	yamlData, err := yaml.Marshal(&cfg)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to marshal YAML", err)
		return
	}

	configPath := filepath.Join(s.dir, "chronicle.yaml")
	if err := os.WriteFile(configPath, yamlData, 0644); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to write config", err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "saved"})
}

// ValidationResult holds config validation results.
type ValidationResult struct {
	Valid    bool     `json:"valid"`
	Errors   []string `json:"errors"`
	Warnings []string `json:"warnings"`
}

func (s *Server) handleValidateConfig(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, "failed to read request body", err)
		return
	}

	var cfg config.Config
	if err := json.Unmarshal(body, &cfg); err != nil {
		writeJSON(w, http.StatusOK, ValidationResult{
			Valid:    false,
			Errors:   []string{"Invalid JSON: " + err.Error()},
			Warnings: []string{},
		})
		return
	}

	result := ValidationResult{
		Valid:    true,
		Errors:   []string{},
		Warnings: []string{},
	}

	if err := cfg.Validate(); err != nil {
		result.Valid = false
		result.Errors = append(result.Errors, err.Error())
	}

	writeJSON(w, http.StatusOK, result)
}

// DiscoveredComponent represents a discovered component.
type DiscoveredComponent struct {
	Name        string   `json:"name"`
	Type        string   `json:"type"`
	Description string   `json:"description"`
	Tags        []string `json:"tags"`
	Produces    []string `json:"produces"`
	Requires    []string `json:"requires"`
	SourceFile  string   `json:"source_file"`
}

// DiscoveryResult contains discovery results.
type DiscoveryResult struct {
	Components   []DiscoveredComponent `json:"components"`
	DiscoveredAt time.Time             `json:"discovered_at"`
}

func (s *Server) handleDiscover(w http.ResponseWriter, _ *http.Request) {
	parser := discovery.NewParser(s.dir)
	registry, err := parser.Discover()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "discovery failed", err)
		return
	}

	components := make([]DiscoveredComponent, 0, len(registry.Components))
	for _, c := range registry.Components {
		tags := c.Tags
		if tags == nil {
			tags = []string{}
		}

		produces := make([]string, 0, len(c.Produces))
		for _, p := range c.Produces {
			if p.Type != "" {
				produces = append(produces, p.Key+":"+p.Type)
			} else {
				produces = append(produces, p.Key)
			}
		}

		requires := make([]string, 0, len(c.Requires))
		for _, r := range c.Requires {
			if r.Type != "" {
				requires = append(requires, r.Key+":"+r.Type)
			} else {
				requires = append(requires, r.Key)
			}
		}

		components = append(components, DiscoveredComponent{
			Name:        c.Name,
			Type:        string(c.Type),
			Description: c.Description,
			Tags:        tags,
			Produces:    produces,
			Requires:    requires,
			SourceFile:  c.SourceFile,
		})
	}

	now := time.Now()
	s.componentsMu.Lock()
	s.components = components
	s.discoveredAt = now
	s.componentsMu.Unlock()

	writeJSON(w, http.StatusOK, DiscoveryResult{
		Components:   components,
		DiscoveredAt: now,
	})
}

func (s *Server) handleGetComponents(w http.ResponseWriter, _ *http.Request) {
	s.componentsMu.RLock()
	components := s.components
	discoveredAt := s.discoveredAt
	s.componentsMu.RUnlock()

	if components == nil {
		writeJSON(w, http.StatusOK, DiscoveryResult{
			Components:   []DiscoveredComponent{},
			DiscoveredAt: time.Time{},
		})
		return
	}

	writeJSON(w, http.StatusOK, DiscoveryResult{
		Components:   components,
		DiscoveredAt: discoveredAt,
	})
}
