// Package standalone provides daemon status tracking and health checking for Chronicle's standalone UI.
package standalone

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// DaemonState represents the state of a Chronicle daemon.
type DaemonState string

const (
	// StateUnknown indicates the daemon state cannot be determined.
	StateUnknown DaemonState = "unknown"
	// StateStopped indicates the daemon is not running.
	StateStopped DaemonState = "stopped"
	// StateStarting indicates the daemon is starting up.
	StateStarting DaemonState = "starting"
	// StateRunning indicates the daemon is running and healthy.
	StateRunning DaemonState = "running"
	// StateUnhealthy indicates the daemon is running but not healthy.
	StateUnhealthy DaemonState = "unhealthy"
)

// DaemonStatus contains the status information for a Chronicle daemon.
type DaemonStatus struct {
	// State is the current state of the daemon.
	State DaemonState `json:"state"`
	// Port is the port the daemon is listening on (if running).
	Port int `json:"port,omitempty"`
	// Version is the daemon version (if available).
	Version string `json:"version,omitempty"`
	// Error contains error details if the daemon is unhealthy or unreachable.
	Error string `json:"error,omitempty"`
	// LastChecked is when this status was last updated.
	LastChecked time.Time `json:"last_checked"`
	// Uptime is the daemon uptime in seconds (if available).
	Uptime float64 `json:"uptime,omitempty"`
	// Scenarios is the count of scenarios (if available).
	Scenarios int `json:"scenarios,omitempty"`
}

// HealthResponse represents the response from a daemon's /health endpoint.
type HealthResponse struct {
	// Status is the health status string (e.g., "healthy").
	Status string `json:"status"`
	// Version is the daemon version.
	Version string `json:"version"`
	// Uptime is the daemon uptime (may be missing in older versions).
	Uptime float64 `json:"uptime,omitempty"`
	// Scenarios is the count of scenarios (may be missing).
	Scenarios int `json:"scenarios,omitempty"`
}

// HealthChecker checks the health of Chronicle daemons.
type HealthChecker struct {
	client *http.Client
	mu     sync.RWMutex
	cache  map[string]*DaemonStatus // keyed by project ID
}

// NewHealthChecker creates a new health checker with a 5-second timeout.
func NewHealthChecker() *HealthChecker {
	return &HealthChecker{
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
		cache: make(map[string]*DaemonStatus),
	}
}

// Check checks the health of a Chronicle daemon for the given project.
// For remote projects (with RemoteURL), it checks the RemoteURL/health endpoint.
// For local projects (without RemoteURL), it tries to find a running daemon on common ports.
func (h *HealthChecker) Check(ctx context.Context, project *Project) *DaemonStatus {
	now := time.Now()

	// If remote URL is set, only check that URL
	if project.RemoteURL != "" {
		if status := h.checkURL(ctx, project.RemoteURL+"/health", 0); status != nil {
			status.LastChecked = now
			h.SetStatus(project.ID, status)
			return status
		}
		// Remote URL failed - return stopped
		status := &DaemonStatus{
			State:       StateStopped,
			LastChecked: now,
		}
		h.SetStatus(project.ID, status)
		return status
	}

	// No remote URL - try common ports for local daemons
	commonPorts := []int{8080, 3000, 8081, 8082}
	for _, port := range commonPorts {
		url := fmt.Sprintf("http://localhost:%d/api/v1/health", port)
		if status := h.checkURL(ctx, url, port); status != nil {
			status.LastChecked = now
			h.SetStatus(project.ID, status)
			return status
		}
	}

	// No daemon found
	status := &DaemonStatus{
		State:       StateStopped,
		LastChecked: now,
	}
	h.SetStatus(project.ID, status)
	return status
}

// checkURL checks a specific health endpoint URL.
// Returns nil if the URL is not reachable.
func (h *HealthChecker) checkURL(ctx context.Context, url string, port int) *DaemonStatus {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil
	}

	resp, err := h.client.Do(req)
	if err != nil {
		return nil
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	// If we get a response, try to parse it
	if resp.StatusCode == http.StatusOK {
		var health HealthResponse
		if err := json.NewDecoder(resp.Body).Decode(&health); err != nil {
			// Could not parse response, but server is up
			return &DaemonStatus{
				State: StateUnhealthy,
				Port:  port,
				Error: fmt.Sprintf("invalid health response: %v", err),
			}
		}

		// Determine state based on status field
		state := StateRunning
		if health.Status != "healthy" {
			state = StateUnhealthy
		}

		return &DaemonStatus{
			State:     state,
			Port:      port,
			Version:   health.Version,
			Uptime:    health.Uptime,
			Scenarios: health.Scenarios,
		}
	}

	// Server responded but not healthy
	return &DaemonStatus{
		State: StateUnhealthy,
		Port:  port,
		Error: fmt.Sprintf("HTTP %d", resp.StatusCode),
	}
}

// GetStatus retrieves the cached status for a project.
// Returns nil if no status is cached.
func (h *HealthChecker) GetStatus(projectID string) *DaemonStatus {
	h.mu.RLock()
	defer h.mu.RUnlock()

	status, ok := h.cache[projectID]
	if !ok {
		return nil
	}

	// Return a copy to prevent external modification
	statusCopy := *status
	return &statusCopy
}

// SetStatus updates the cached status for a project.
func (h *HealthChecker) SetStatus(projectID string, status *DaemonStatus) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Store a copy to prevent external modification
	statusCopy := *status
	h.cache[projectID] = &statusCopy
}

// ClearStatus removes the cached status for a project.
func (h *HealthChecker) ClearStatus(projectID string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	delete(h.cache, projectID)
}
