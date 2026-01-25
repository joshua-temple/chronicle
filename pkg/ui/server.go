// Package ui provides a standalone HTTP server for the Chronicle UI.
// It serves a local web interface for editing configuration and scenarios.
package ui

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"strings"
	"time"

	"github.com/joshua-temple/chronicle/web"
)

// Default configuration values.
const (
	DefaultPort = 3000
	DefaultDir  = "."
)

// Server provides a local HTTP server for the Chronicle UI.
type Server struct {
	port int
	dir  string
	mux  *http.ServeMux

	httpServer *http.Server
	webFS      fs.FS
}

// Option configures a Server.
type Option func(*Server)

// WithPort sets the server port.
func WithPort(port int) Option {
	return func(s *Server) {
		s.port = port
	}
}

// WithDir sets the working directory for Chronicle operations.
func WithDir(dir string) Option {
	return func(s *Server) {
		s.dir = dir
	}
}

// New creates a new UI server with the given options.
func New(opts ...Option) *Server {
	s := &Server{
		port: DefaultPort,
		dir:  DefaultDir,
		mux:  http.NewServeMux(),
	}

	for _, opt := range opts {
		opt(s)
	}

	// Initialize the embedded web filesystem
	s.initWebFS()

	// Setup routes
	s.setupRoutes()

	return s
}

// Port returns the configured port.
func (s *Server) Port() int {
	return s.port
}

// Dir returns the configured working directory.
func (s *Server) Dir() string {
	return s.dir
}

// initWebFS initializes the embedded web filesystem.
func (s *Server) initWebFS() {
	// Extract the dist subdirectory from the embedded FS.
	// The web.WebFS embeds files as "dist/*", so we need fs.Sub to serve them at root.
	subFS, err := fs.Sub(web.WebFS, "dist")
	if err != nil {
		// If dist doesn't exist (e.g., not built yet), leave webFS nil.
		return
	}
	s.webFS = subFS
}

// setupRoutes configures the HTTP routes.
func (s *Server) setupRoutes() {
	// Local API routes
	s.mux.HandleFunc("GET /api/local/project", s.handleProject)
	s.mux.HandleFunc("GET /api/local/config", s.handleGetConfig)
	s.mux.HandleFunc("PUT /api/local/config", s.handlePutConfig)
	s.mux.HandleFunc("POST /api/local/config/validate", s.handleValidateConfig)
	s.mux.HandleFunc("POST /api/local/discover", s.handleDiscover)
	s.mux.HandleFunc("GET /api/local/components", s.handleGetComponents)

	// Static file serving (SPA) - must be last to not interfere with API routes
	if s.webFS != nil {
		s.mux.Handle("GET /", s.spaHandler())
	} else {
		s.mux.Handle("GET /", s.devModeHandler())
	}
}

// Start starts the HTTP server and blocks until the context is cancelled.
// It performs graceful shutdown when the context is done.
func (s *Server) Start(ctx context.Context) error {
	addr := fmt.Sprintf(":%d", s.port)

	s.httpServer = &http.Server{
		Addr:         addr,
		Handler:      s.mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Channel to capture server errors
	errCh := make(chan error, 1)

	// Start server in goroutine
	go func() {
		if err := s.httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
		close(errCh)
	}()

	// Wait for context cancellation or server error
	select {
	case <-ctx.Done():
		// Graceful shutdown with timeout
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return s.httpServer.Shutdown(shutdownCtx)
	case err := <-errCh:
		return err
	}
}

// spaHandler returns an HTTP handler that serves the SPA.
// For any path that doesn't match a real file, it serves index.html.
func (s *Server) spaHandler() http.Handler {
	fileServer := http.FileServer(http.FS(s.webFS))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/")
		if path == "" {
			path = "index.html"
		}

		// Check if file exists using Stat (avoids opening file descriptor)
		_, err := fs.Stat(s.webFS, path)
		if err != nil {
			// File not found - serve index.html for SPA routing
			r.URL.Path = "/"
			fileServer.ServeHTTP(w, r)
			return
		}

		// File exists, serve it
		fileServer.ServeHTTP(w, r)
	})
}

// devModeHandler returns a handler for when no embedded files are available.
func (s *Server) devModeHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`<!DOCTYPE html>
<html>
<head><title>Chronicle UI</title></head>
<body style="font-family: system-ui; max-width: 600px; margin: 50px auto; padding: 20px;">
<h1>Chronicle UI Server</h1>
<p>Web UI not embedded. For development:</p>
<pre>cd web && npm run dev</pre>
<p>API available at <a href="/api/local/project">/api/local/project</a></p>
</body>
</html>`))
	})
}

// API Handler placeholders

// handleDiscover runs component discovery.
func (s *Server) handleDiscover(w http.ResponseWriter, _ *http.Request) {
	// Placeholder - will be implemented in a later task
	writeJSON(w, http.StatusOK, map[string]any{})
}

// handleGetComponents returns discovered components.
func (s *Server) handleGetComponents(w http.ResponseWriter, _ *http.Request) {
	// Placeholder - will be implemented in a later task
	writeJSON(w, http.StatusOK, map[string]any{"components": []any{}})
}

// Helper functions for JSON responses.

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}
