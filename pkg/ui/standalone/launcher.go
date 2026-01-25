// Package standalone provides process launching and lifecycle management for Chronicle daemons.
package standalone

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"sync"
	"time"
)

// Launcher manages Chronicle daemon processes for local projects.
type Launcher struct {
	mu         sync.RWMutex
	processes  map[string]*exec.Cmd // project ID -> process
	ports      map[string]int       // project ID -> port
	checker    *HealthChecker
	binaryPath string // path to chronicle binary (defaults to "chronicle")
}

// NewLauncher creates a new process launcher with the given health checker.
// The health checker is used to verify daemon health after launching.
func NewLauncher(checker *HealthChecker) *Launcher {
	return &Launcher{
		processes:  make(map[string]*exec.Cmd),
		ports:      make(map[string]int),
		checker:    checker,
		binaryPath: "chronicle",
	}
}

// Launch starts a Chronicle daemon for the given project.
// If a daemon is already running for this project, it returns the existing port.
// Otherwise, it spawns a new daemon process and waits for it to become healthy.
// Returns the port the daemon is listening on, or an error if launch fails.
func (l *Launcher) Launch(ctx context.Context, project *Project) (int, error) {
	l.mu.Lock()

	// Check if already running
	if port, ok := l.ports[project.ID]; ok {
		l.mu.Unlock()
		return port, nil
	}

	// Find an available port
	port, err := l.findAvailablePort()
	if err != nil {
		l.mu.Unlock()
		return 0, fmt.Errorf("failed to find available port: %w", err)
	}

	// Create the command
	cmd := exec.CommandContext(ctx, l.binaryPath, "daemon", "--addr", fmt.Sprintf(":%d", port), "--no-auth")
	cmd.Dir = project.Path
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Start the process
	if err := cmd.Start(); err != nil {
		l.mu.Unlock()
		return 0, fmt.Errorf("failed to start daemon: %w", err)
	}

	// Store process and port
	l.processes[project.ID] = cmd
	l.ports[project.ID] = port
	l.mu.Unlock()

	// Wait for daemon to become healthy
	if err := l.waitForHealth(ctx, project.ID, port); err != nil {
		// Health check failed - stop the process
		_ = l.Stop(ctx, project.ID)
		return 0, fmt.Errorf("daemon failed to become healthy: %w", err)
	}

	return port, nil
}

// Stop gracefully stops the Chronicle daemon for the given project.
// It sends SIGINT for graceful shutdown, waits up to 10 seconds,
// and then sends SIGKILL if the process hasn't exited.
func (l *Launcher) Stop(ctx context.Context, projectID string) error {
	l.mu.Lock()
	cmd, ok := l.processes[projectID]
	if !ok {
		l.mu.Unlock()
		return fmt.Errorf("no daemon running for project %s", projectID)
	}

	// Remove from maps immediately
	delete(l.processes, projectID)
	delete(l.ports, projectID)
	l.mu.Unlock()

	// Clear health check cache
	l.checker.ClearStatus(projectID)

	// Process already exited
	if cmd.Process == nil {
		return nil
	}

	// Send SIGINT for graceful shutdown
	if err := cmd.Process.Signal(os.Interrupt); err != nil {
		// Process may have already exited
		return nil
	}

	// Wait for process to exit with timeout
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	// Create a timeout context for shutdown
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	select {
	case <-shutdownCtx.Done():
		// Timeout - force kill
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
		}
		return fmt.Errorf("daemon did not stop gracefully, killed")
	case err := <-done:
		// Process exited
		if err != nil && err.Error() != "signal: interrupt" {
			return fmt.Errorf("daemon stopped with error: %w", err)
		}
		return nil
	}
}

// IsRunning returns true if a daemon is currently running for the given project.
func (l *Launcher) IsRunning(projectID string) bool {
	l.mu.RLock()
	defer l.mu.RUnlock()

	_, ok := l.processes[projectID]
	return ok
}

// GetPort returns the port the daemon is listening on for the given project.
// Returns 0 if no daemon is running.
func (l *Launcher) GetPort(projectID string) int {
	l.mu.RLock()
	defer l.mu.RUnlock()

	return l.ports[projectID]
}

// SetBinaryPath sets the path to the chronicle binary.
// This is useful for testing or when the binary is not in PATH.
func (l *Launcher) SetBinaryPath(path string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.binaryPath = path
}

// findAvailablePort finds an available port by opening a listener on a random port,
// getting the assigned port, and immediately closing the listener.
func (l *Launcher) findAvailablePort() (int, error) {
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		return 0, err
	}
	defer func() {
		_ = listener.Close()
	}()

	addr := listener.Addr().(*net.TCPAddr)
	return addr.Port, nil
}

// waitForHealth polls the daemon's health endpoint until it becomes healthy or times out.
// It polls every 500ms for up to 30 seconds.
func (l *Launcher) waitForHealth(ctx context.Context, projectID string, port int) error {
	timeout := 30 * time.Second
	pollInterval := 500 * time.Millisecond
	deadline := time.Now().Add(timeout)

	// Create a temporary project for health checking with the port
	tempProject := &Project{
		ID:        projectID,
		RemoteURL: fmt.Sprintf("http://localhost:%d/api/v1", port),
	}

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if time.Now().After(deadline) {
				return fmt.Errorf("timeout waiting for daemon to become healthy")
			}

			// Check health
			status := l.checker.Check(ctx, tempProject)
			if status.State == StateRunning {
				return nil
			}
		}
	}
}
