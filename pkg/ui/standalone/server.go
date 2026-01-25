// Package standalone provides the standalone HTTP server for Chronicle's multi-project control center.
package standalone

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/joshua-temple/chronicle/web"
)

// Server is the standalone HTTP server that manages multiple Chronicle projects.
type Server struct {
	port     int
	registry *Registry
	checker  *HealthChecker
	launcher *Launcher
	mux      *http.ServeMux
	server   *http.Server
	webFS    fs.FS
}

// ServerOption is a functional option for configuring the server.
type ServerOption func(*Server)

// WithPort sets the port the server listens on.
func WithPort(port int) ServerOption {
	return func(s *Server) {
		s.port = port
	}
}

// WithRegistry sets the project registry.
func WithRegistry(registry *Registry) ServerOption {
	return func(s *Server) {
		s.registry = registry
	}
}

// WithWebFS sets the embedded web filesystem.
func WithWebFS(webFS fs.FS) ServerOption {
	return func(s *Server) {
		s.webFS = webFS
	}
}

// NewServer creates a new standalone server with the given options.
// If no registry is provided, it creates one at ~/.chronicle/projects.json.
// If no web FS is provided, it uses the embedded web.WebFS.
func NewServer(opts ...ServerOption) *Server {
	s := &Server{
		port:  8080, // default port
		webFS: web.WebFS,
		mux:   http.NewServeMux(),
	}

	// Apply options
	for _, opt := range opts {
		opt(s)
	}

	// Create default registry if not provided
	if s.registry == nil {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			panic(fmt.Errorf("failed to get home directory: %w", err))
		}
		registryPath := filepath.Join(homeDir, ".chronicle", "projects.json")
		registry, err := NewRegistry(registryPath)
		if err != nil {
			panic(fmt.Errorf("failed to create registry: %w", err))
		}
		s.registry = registry
	}

	// Create health checker and launcher
	s.checker = NewHealthChecker()
	s.launcher = NewLauncher(s.checker)

	// Set up routes
	s.setupRoutes()

	return s
}

// Start starts the HTTP server and blocks until the context is canceled or an error occurs.
func (s *Server) Start(ctx context.Context) error {
	s.server = &http.Server{
		Addr:              fmt.Sprintf(":%d", s.port),
		Handler:           s.mux,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	// Start server in background
	errChan := make(chan error, 1)
	go func() {
		if err := s.server.ListenAndServe(); err != nil {
			// Send all errors, including ErrServerClosed
			errChan <- err
		}
	}()

	// Wait for context cancellation or error
	select {
	case <-ctx.Done():
		// Context canceled - shut down gracefully
		return s.Stop(context.Background())
	case err := <-errChan:
		// Server stopped - return error unless it was a graceful shutdown
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}

// Stop gracefully stops the HTTP server with a 5-second timeout.
func (s *Server) Stop(ctx context.Context) error {
	if s.server == nil {
		return nil
	}

	shutdownCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	return s.server.Shutdown(shutdownCtx)
}

// setupRoutes configures all HTTP routes for the server.
func (s *Server) setupRoutes() {
	// API routes
	s.mux.HandleFunc("GET /api/standalone/mode", s.handleMode)
	s.mux.HandleFunc("GET /api/standalone/projects", s.handleListProjects)
	s.mux.HandleFunc("POST /api/standalone/projects", s.handleAddProject)
	s.mux.HandleFunc("DELETE /api/standalone/projects/{id}", s.handleRemoveProject)
	s.mux.HandleFunc("PUT /api/standalone/projects/{id}", s.handleUpdateProject)
	s.mux.HandleFunc("POST /api/standalone/projects/{id}/launch", s.handleLaunchDaemon)
	s.mux.HandleFunc("POST /api/standalone/projects/{id}/stop", s.handleStopDaemon)
	s.mux.HandleFunc("GET /api/standalone/projects/{id}/health", s.handleHealthCheck)
	s.mux.HandleFunc("POST /api/standalone/discover", s.handleDiscover)

	// SPA handler for all other routes
	s.mux.HandleFunc("/", s.handleSPA)
}

// handleMode returns the server mode (always "standalone").
func (s *Server) handleMode(w http.ResponseWriter, r *http.Request) {
	s.setCORSHeaders(w, r)
	s.jsonResponse(w, http.StatusOK, map[string]string{"mode": "standalone"})
}

// handleListProjects returns all projects with their current status.
func (s *Server) handleListProjects(w http.ResponseWriter, r *http.Request) {
	s.setCORSHeaders(w, r)

	projects := s.registry.List()

	// Build response with status for each project
	type projectWithStatus struct {
		Project
		Status *DaemonStatus `json:"status"`
	}

	response := make([]projectWithStatus, len(projects))
	for i, p := range projects {
		// Check health asynchronously
		status := s.checker.Check(r.Context(), &p)
		response[i] = projectWithStatus{
			Project: p,
			Status:  status,
		}
	}

	s.jsonResponse(w, http.StatusOK, map[string]interface{}{"projects": response})
}

// handleAddProject adds a new project to the registry.
func (s *Server) handleAddProject(w http.ResponseWriter, r *http.Request) {
	s.setCORSHeaders(w, r)

	var req struct {
		Name      string `json:"name"`
		Path      string `json:"path"`
		RemoteURL string `json:"remote_url,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.jsonError(w, http.StatusBadRequest, fmt.Sprintf("invalid request body: %v", err))
		return
	}

	if req.Name == "" || req.Path == "" {
		s.jsonError(w, http.StatusBadRequest, "name and path are required")
		return
	}

	project := Project{
		Name:      req.Name,
		Path:      req.Path,
		RemoteURL: req.RemoteURL,
	}

	id, err := s.registry.Add(project)
	if err != nil {
		if strings.Contains(err.Error(), "already exists") {
			s.jsonError(w, http.StatusConflict, err.Error())
		} else {
			s.jsonError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}

	s.jsonResponse(w, http.StatusCreated, map[string]string{
		"id":      id,
		"message": "project added successfully",
	})
}

// handleRemoveProject removes a project from the registry.
func (s *Server) handleRemoveProject(w http.ResponseWriter, r *http.Request) {
	s.setCORSHeaders(w, r)

	id := r.PathValue("id")
	if id == "" {
		s.jsonError(w, http.StatusBadRequest, "project id is required")
		return
	}

	// Check if project exists
	if s.registry.Get(id) == nil {
		s.jsonError(w, http.StatusNotFound, "project not found")
		return
	}

	// Stop daemon if running
	if s.launcher.IsRunning(id) {
		_ = s.launcher.Stop(r.Context(), id)
	}

	if err := s.registry.Remove(id); err != nil {
		s.jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.jsonResponse(w, http.StatusOK, map[string]string{"message": "project removed successfully"})
}

// handleUpdateProject updates project preferences.
func (s *Server) handleUpdateProject(w http.ResponseWriter, r *http.Request) {
	s.setCORSHeaders(w, r)

	id := r.PathValue("id")
	if id == "" {
		s.jsonError(w, http.StatusBadRequest, "project id is required")
		return
	}

	project := s.registry.Get(id)
	if project == nil {
		s.jsonError(w, http.StatusNotFound, "project not found")
		return
	}

	var req struct {
		Preferences map[string]interface{} `json:"preferences"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.jsonError(w, http.StatusBadRequest, fmt.Sprintf("invalid request body: %v", err))
		return
	}

	// Update only preferences
	project.Preferences = req.Preferences
	if err := s.registry.Update(*project); err != nil {
		s.jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.jsonResponse(w, http.StatusOK, map[string]string{"message": "project updated successfully"})
}

// handleLaunchDaemon starts a Chronicle daemon for a local project.
func (s *Server) handleLaunchDaemon(w http.ResponseWriter, r *http.Request) {
	s.setCORSHeaders(w, r)

	id := r.PathValue("id")
	if id == "" {
		s.jsonError(w, http.StatusBadRequest, "project id is required")
		return
	}

	project := s.registry.Get(id)
	if project == nil {
		s.jsonError(w, http.StatusNotFound, "project not found")
		return
	}

	// Cannot launch daemon for remote projects
	if project.RemoteURL != "" {
		s.jsonError(w, http.StatusBadRequest, "cannot launch daemon for remote project")
		return
	}

	port, err := s.launcher.Launch(r.Context(), project)
	if err != nil {
		s.jsonError(w, http.StatusInternalServerError, fmt.Sprintf("failed to launch daemon: %v", err))
		return
	}

	s.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"message": "daemon launched successfully",
		"port":    port,
	})
}

// handleStopDaemon stops a running Chronicle daemon.
func (s *Server) handleStopDaemon(w http.ResponseWriter, r *http.Request) {
	s.setCORSHeaders(w, r)

	id := r.PathValue("id")
	if id == "" {
		s.jsonError(w, http.StatusBadRequest, "project id is required")
		return
	}

	if !s.launcher.IsRunning(id) {
		s.jsonError(w, http.StatusBadRequest, "no daemon running for this project")
		return
	}

	if err := s.launcher.Stop(r.Context(), id); err != nil {
		s.jsonError(w, http.StatusInternalServerError, fmt.Sprintf("failed to stop daemon: %v", err))
		return
	}

	s.jsonResponse(w, http.StatusOK, map[string]string{"message": "daemon stopped successfully"})
}

// handleHealthCheck checks the health of a project's daemon.
func (s *Server) handleHealthCheck(w http.ResponseWriter, r *http.Request) {
	s.setCORSHeaders(w, r)

	id := r.PathValue("id")
	if id == "" {
		s.jsonError(w, http.StatusBadRequest, "project id is required")
		return
	}

	project := s.registry.Get(id)
	if project == nil {
		s.jsonError(w, http.StatusNotFound, "project not found")
		return
	}

	status := s.checker.Check(r.Context(), project)
	s.jsonResponse(w, http.StatusOK, status)
}

// handleDiscover scans for Chronicle projects in a directory.
func (s *Server) handleDiscover(w http.ResponseWriter, r *http.Request) {
	s.setCORSHeaders(w, r)

	var req struct {
		Path string `json:"path"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.jsonError(w, http.StatusBadRequest, fmt.Sprintf("invalid request body: %v", err))
		return
	}

	if req.Path == "" {
		s.jsonError(w, http.StatusBadRequest, "path is required")
		return
	}

	found, err := s.discoverProjects(req.Path)
	if err != nil {
		s.jsonError(w, http.StatusInternalServerError, fmt.Sprintf("discovery failed: %v", err))
		return
	}

	s.jsonResponse(w, http.StatusOK, map[string]interface{}{"found": found})
}

// discoverProjects recursively scans a directory for Chronicle projects.
// A directory is considered a Chronicle project if it contains a chronicle.yaml file.
func (s *Server) discoverProjects(rootPath string) ([]string, error) {
	var found []string

	err := filepath.WalkDir(rootPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			// Skip directories we can't read
			return nil
		}

		// Skip hidden directories and common ignore patterns
		if d.IsDir() && strings.HasPrefix(d.Name(), ".") {
			return filepath.SkipDir
		}
		if d.IsDir() && (d.Name() == "node_modules" || d.Name() == "vendor") {
			return filepath.SkipDir
		}

		// Check for chronicle.yaml
		if !d.IsDir() && d.Name() == "chronicle.yaml" {
			// Found a Chronicle project
			projectPath := filepath.Dir(path)
			found = append(found, projectPath)
			// Don't descend into Chronicle projects
			return filepath.SkipDir
		}

		return nil
	})

	return found, err
}

// handleSPA serves the React SPA, handling client-side routing.
// For API routes, it does nothing (they're handled by other handlers).
// For asset files, it serves them directly from the embedded FS.
// For all other routes, it serves index.html to support client-side routing.
func (s *Server) handleSPA(w http.ResponseWriter, r *http.Request) {
	// API routes are handled elsewhere
	if strings.HasPrefix(r.URL.Path, "/api/") {
		http.NotFound(w, r)
		return
	}

	s.setCORSHeaders(w, r)

	path := r.URL.Path
	if path == "/" {
		path = "dist/index.html"
	} else {
		// Remove leading slash and prepend dist/
		path = "dist" + path
	}

	// Try to open the requested file
	file, err := s.webFS.Open(path)
	if err != nil {
		// File not found - serve index.html for client-side routing
		indexFile, err := s.webFS.Open("dist/index.html")
		if err != nil {
			http.Error(w, "index.html not found", http.StatusInternalServerError)
			return
		}
		defer func() {
			_ = indexFile.Close()
		}()

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = io.Copy(w, indexFile)
		return
	}
	defer func() {
		_ = file.Close()
	}()

	// Serve the file with appropriate content type
	stat, err := file.Stat()
	if err != nil || stat.IsDir() {
		// Serve index.html for directories
		indexFile, err := s.webFS.Open("dist/index.html")
		if err != nil {
			http.Error(w, "index.html not found", http.StatusInternalServerError)
			return
		}
		defer func() {
			_ = indexFile.Close()
		}()

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = io.Copy(w, indexFile)
		return
	}

	// Set content type based on file extension
	contentType := getContentType(path)
	w.Header().Set("Content-Type", contentType)

	_, _ = io.Copy(w, file)
}

// getContentType returns the MIME type based on file extension.
func getContentType(path string) string {
	ext := filepath.Ext(path)
	switch ext {
	case ".html":
		return "text/html; charset=utf-8"
	case ".js":
		return "application/javascript"
	case ".css":
		return "text/css"
	case ".json":
		return "application/json"
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".svg":
		return "image/svg+xml"
	case ".ico":
		return "image/x-icon"
	default:
		return "application/octet-stream"
	}
}

// setCORSHeaders sets CORS headers for development.
func (s *Server) setCORSHeaders(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	// Handle preflight requests
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
	}
}

// jsonResponse writes a JSON response with the given status code and data.
func (s *Server) jsonResponse(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

// jsonError writes a JSON error response with the given status code and message.
func (s *Server) jsonError(w http.ResponseWriter, status int, message string) {
	s.jsonResponse(w, status, map[string]string{"error": message})
}
