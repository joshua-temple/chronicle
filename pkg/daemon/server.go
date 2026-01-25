package daemon

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/joshua-temple/chronicle/pkg/config"
	"github.com/joshua-temple/chronicle/pkg/discovery"
	"github.com/joshua-temple/chronicle/pkg/execution"
	"github.com/joshua-temple/chronicle/pkg/results"
	"github.com/joshua-temple/chronicle/pkg/scenario"
)

// Server provides a REST API for Chronicle daemon mode.
type Server struct {
	mu sync.RWMutex

	// Configuration
	config     *config.Config
	configPath string

	// Discovery
	registry *discovery.Registry

	// Execution
	executor *execution.Executor
	resolver *scenario.Resolver

	// Results storage
	storage results.Storage

	// HTTP server
	httpServer *http.Server
	router     *http.ServeMux

	// Authentication
	auth *Auth

	// Event bus
	eventBus EventBus

	// Config watcher
	watcher *ConfigWatcher

	// Active runs
	activeRuns map[string]*RunInfo
}

// RunInfo tracks information about an active run.
type RunInfo struct {
	ID         string
	Status     string
	ScenarioID string
	StartTime  time.Time
	Cancel     context.CancelFunc
}

// ServerOption configures a Server.
type ServerOption func(*Server)

// NewServer creates a new daemon server.
func NewServer(configPath string, opts ...ServerOption) (*Server, error) {
	s := &Server{
		configPath: configPath,
		activeRuns: make(map[string]*RunInfo),
		router:     http.NewServeMux(),
	}

	for _, opt := range opts {
		opt(s)
	}

	// Load configuration
	cfg, err := config.Load(configPath)
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}
	s.config = cfg

	// Discover components
	parser := discovery.NewParser(cfg.Discovery.Paths...)
	registry, err := parser.Discover()
	if err != nil {
		return nil, fmt.Errorf("discover: %w", err)
	}
	s.registry = registry

	// Create resolver
	s.resolver = scenario.NewResolver(cfg, registry)

	// Create executor
	s.executor = execution.NewExecutor(
		execution.WithDefaultTimeout(cfg.Execution.DefaultTimeout),
		execution.WithParallelism(cfg.Execution.Parallelism),
		execution.WithFailFast(cfg.Execution.FailFast),
	)

	// Register components
	for _, comp := range registry.Components {
		s.executor.RegisterComponent(comp)
	}

	// Setup results storage
	if s.storage == nil {
		storagePath := cfg.Results.Storage.Path
		if storagePath == "" {
			storagePath = ".chronicle/results"
		}
		storage, err := results.NewFileStorage(storagePath)
		if err != nil {
			return nil, fmt.Errorf("create storage: %w", err)
		}
		s.storage = storage
	}

	// Setup authentication
	if s.auth == nil {
		s.auth = NewAuth(AuthConfig{Method: AuthMethodAPIKey})
	}

	// Setup event bus
	if s.eventBus == nil {
		s.eventBus = NewEmbeddedEventBus()
	}

	// Setup routes
	s.setupRoutes()

	return s, nil
}

// WithAuth sets the authentication provider.
func WithAuth(auth *Auth) ServerOption {
	return func(s *Server) {
		s.auth = auth
	}
}

// WithStorage sets the results storage.
func WithStorage(storage results.Storage) ServerOption {
	return func(s *Server) {
		s.storage = storage
	}
}

// WithEventBus sets the event bus.
func WithEventBus(bus EventBus) ServerOption {
	return func(s *Server) {
		s.eventBus = bus
	}
}

// setupRoutes configures the HTTP routes.
func (s *Server) setupRoutes() {
	// Health check (no auth required)
	s.router.HandleFunc("GET /api/v1/health", s.handleHealth)

	// Runs API
	s.router.HandleFunc("POST /api/v1/runs", s.auth.Middleware(s.handleCreateRun))
	s.router.HandleFunc("GET /api/v1/runs", s.auth.Middleware(s.handleListRuns))
	s.router.HandleFunc("GET /api/v1/runs/{id}", s.auth.Middleware(s.handleGetRun))
	s.router.HandleFunc("DELETE /api/v1/runs/{id}", s.auth.Middleware(s.handleDeleteRun))

	// Scenarios API
	s.router.HandleFunc("GET /api/v1/scenarios", s.auth.Middleware(s.handleListScenarios))
	s.router.HandleFunc("GET /api/v1/scenarios/{name}", s.auth.Middleware(s.handleGetScenario))

	// Components API
	s.router.HandleFunc("GET /api/v1/components", s.auth.Middleware(s.handleListComponents))
	s.router.HandleFunc("GET /api/v1/components/{name}", s.auth.Middleware(s.handleGetComponent))

	// Results API
	s.router.HandleFunc("GET /api/v1/results", s.auth.Middleware(s.handleListResults))
	s.router.HandleFunc("GET /api/v1/results/{id}", s.auth.Middleware(s.handleGetResult))
	s.router.HandleFunc("DELETE /api/v1/results/{id}", s.auth.Middleware(s.handleDeleteResult))

	// Config API
	s.router.HandleFunc("GET /api/v1/config", s.auth.Middleware(s.handleGetConfig))
	s.router.HandleFunc("POST /api/v1/config/reload", s.auth.Middleware(s.handleReloadConfig))
}

// Start starts the HTTP server.
func (s *Server) Start(addr string) error {
	s.httpServer = &http.Server{
		Addr:         addr,
		Handler:      s.router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start config watcher if configured
	if s.watcher != nil {
		go func() {
			if err := s.watcher.Start(context.Background()); err != nil {
				fmt.Printf("Config watcher error: %v\n", err)
			}
		}()
	}

	return s.httpServer.ListenAndServe()
}

// Shutdown gracefully shuts down the server.
func (s *Server) Shutdown(ctx context.Context) error {
	// Cancel all active runs
	s.mu.Lock()
	for _, run := range s.activeRuns {
		if run.Cancel != nil {
			run.Cancel()
		}
	}
	s.mu.Unlock()

	// Stop config watcher
	if s.watcher != nil {
		s.watcher.Stop()
	}

	// Shutdown HTTP server
	return s.httpServer.Shutdown(ctx)
}

// ReloadConfig reloads the configuration.
func (s *Server) ReloadConfig() error {
	cfg, err := config.Load(s.configPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	// Re-discover components
	parser := discovery.NewParser(cfg.Discovery.Paths...)
	registry, err := parser.Discover()
	if err != nil {
		return fmt.Errorf("discover: %w", err)
	}

	// Update server state
	s.mu.Lock()
	defer s.mu.Unlock()

	s.config = cfg
	s.registry = registry
	s.resolver = scenario.NewResolver(cfg, registry)

	// Re-register components
	s.executor = execution.NewExecutor(
		execution.WithDefaultTimeout(cfg.Execution.DefaultTimeout),
		execution.WithParallelism(cfg.Execution.Parallelism),
		execution.WithFailFast(cfg.Execution.FailFast),
	)
	for _, comp := range registry.Components {
		s.executor.RegisterComponent(comp)
	}

	// Publish reload event
	s.eventBus.Publish(Event{
		Type:      EventConfigReload,
		Timestamp: time.Now(),
		Data:      map[string]any{"config": s.configPath},
	})

	return nil
}

// Helper functions for JSON responses
func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
