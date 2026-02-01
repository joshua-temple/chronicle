package client

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

// LockfileInfo contains daemon instance information.
type LockfileInfo struct {
	PID        int    `json:"pid"`
	Port       int    `json:"port"`
	ProjectDir string `json:"project_dir"`
	StartedAt  string `json:"started_at"`
}

// DaemonManager manages daemon lifecycle and connectivity.
type DaemonManager struct {
	client       *Client
	port         int
	process      *exec.Cmd
	binaryPath   string
	projectDir   string
	lockfilePath string
}

// ManagerOption configures a DaemonManager.
type ManagerOption func(*DaemonManager)

// WithBinaryPath sets the path to the chronicle binary.
func WithBinaryPath(path string) ManagerOption {
	return func(m *DaemonManager) {
		m.binaryPath = path
	}
}

// WithProjectDir sets the project directory for daemon execution.
func WithProjectDir(dir string) ManagerOption {
	return func(m *DaemonManager) {
		m.projectDir = dir
	}
}

// NewDaemonManager creates a new daemon manager.
func NewDaemonManager(opts ...ManagerOption) *DaemonManager {
	m := &DaemonManager{
		binaryPath: "chronicle",
		projectDir: ".",
	}

	for _, opt := range opts {
		opt(m)
	}

	return m
}

// EnsureDaemon ensures a daemon is running, prompting to start one if needed.
// If interactive is true and no daemon is running, it prompts the user.
// Returns a client connected to the daemon.
func (m *DaemonManager) EnsureDaemon(ctx context.Context, interactive bool) (*Client, error) {
	// First, check lockfile for existing daemon
	lockInfo, err := m.checkLockfile()
	if err == nil && lockInfo != nil {
		client := New(fmt.Sprintf("http://localhost:%d", lockInfo.Port))
		if client.IsHealthy(ctx) {
			m.client = client
			m.port = lockInfo.Port
			fmt.Printf("Found existing daemon on port %d\n", lockInfo.Port)
			return client, nil
		}
		// Lockfile exists but daemon not healthy, clean up
		_ = m.removeLockfile()
	}

	// Try common ports as fallback
	ports := []int{3000, 8080, 8081, 8082}
	for _, port := range ports {
		client := New(fmt.Sprintf("http://localhost:%d", port))
		if client.IsHealthy(ctx) {
			m.client = client
			m.port = port
			return client, nil
		}
	}

	// No daemon found
	if !interactive {
		return nil, fmt.Errorf("no daemon running (try: chronicle daemon)")
	}

	// Prompt user to start daemon
	fmt.Println("No Chronicle daemon is running.")
	fmt.Print("Would you like to start one? [Y/n] ")

	reader := bufio.NewReader(os.Stdin)
	answer, err := reader.ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("read input: %w", err)
	}

	answer = strings.TrimSpace(strings.ToLower(answer))
	if answer != "" && answer != "y" && answer != "yes" {
		return nil, fmt.Errorf("daemon required to run scenarios")
	}

	// Start daemon
	return m.StartDaemon(ctx)
}

// StartDaemon starts a new daemon process and waits for it to be healthy.
func (m *DaemonManager) StartDaemon(ctx context.Context) (*Client, error) {
	// Find available port
	port, err := findAvailablePort()
	if err != nil {
		return nil, fmt.Errorf("find port: %w", err)
	}

	// Start daemon process
	fmt.Printf("Starting daemon on port %d...\n", port)

	cmd := exec.CommandContext(ctx, m.binaryPath, "daemon", "--addr", fmt.Sprintf(":%d", port), "--no-auth")
	cmd.Dir = m.projectDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start daemon: %w", err)
	}

	m.process = cmd
	m.port = port

	// Wait for daemon to be healthy
	client := New(fmt.Sprintf("http://localhost:%d", port))
	if err := m.waitForHealth(ctx, client); err != nil {
		// Kill the process if health check fails
		if m.process != nil && m.process.Process != nil {
			_ = m.process.Process.Kill()
		}
		return nil, fmt.Errorf("daemon health check failed: %w", err)
	}

	m.client = client
	fmt.Println("Daemon started successfully!")
	return client, nil
}

// waitForHealth waits for the daemon to become healthy.
func (m *DaemonManager) waitForHealth(ctx context.Context, client *Client) error {
	timeout := 30 * time.Second
	pollInterval := 500 * time.Millisecond
	deadline := time.Now().Add(timeout)

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

			if client.IsHealthy(ctx) {
				return nil
			}
		}
	}
}

// StopDaemon stops the daemon process if we started it.
func (m *DaemonManager) StopDaemon() error {
	if m.process == nil || m.process.Process == nil {
		return nil
	}

	// Send interrupt signal
	if err := m.process.Process.Signal(os.Interrupt); err != nil {
		// Process may have already exited
		return nil
	}

	// Wait for process to exit
	done := make(chan error, 1)
	go func() {
		done <- m.process.Wait()
	}()

	select {
	case <-time.After(10 * time.Second):
		// Force kill
		_ = m.process.Process.Kill()
		return fmt.Errorf("daemon did not stop gracefully")
	case err := <-done:
		if err != nil && !strings.Contains(err.Error(), "interrupt") {
			return err
		}
		return nil
	}
}

// GetClient returns the daemon client.
func (m *DaemonManager) GetClient() *Client {
	return m.client
}

// GetPort returns the daemon port.
func (m *DaemonManager) GetPort() int {
	return m.port
}

// findAvailablePort finds an available port by opening a listener.
func findAvailablePort() (int, error) {
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		return 0, err
	}
	defer func() { _ = listener.Close() }()

	addr := listener.Addr().(*net.TCPAddr)
	return addr.Port, nil
}

// CheckDaemonHealth checks if a daemon is running and healthy.
// First checks lockfile, then falls back to common ports.
func CheckDaemonHealth(ctx context.Context) (bool, int) {
	// Check current directory lockfile first
	if client, port := CheckDaemonFromLockfile(ctx, "."); client != nil {
		return true, port
	}

	// Fall back to checking common ports
	ports := []int{3000, 8080, 8081, 8082}
	for _, port := range ports {
		client := New(fmt.Sprintf("http://localhost:%d", port))
		if client.IsHealthy(ctx) {
			return true, port
		}
	}
	return false, 0
}

// PromptAndStart prompts the user to start a daemon and starts it if they agree.
// Returns the client and port, or an error if the user declines or startup fails.
func PromptAndStart(ctx context.Context, projectDir string) (*Client, int, error) {
	manager := NewDaemonManager(WithProjectDir(projectDir))
	client, err := manager.EnsureDaemon(ctx, true)
	if err != nil {
		return nil, 0, err
	}
	return client, manager.GetPort(), nil
}

// Lockfile management

// lockfilePath returns the path to the lockfile for this project.
func (m *DaemonManager) getLockfilePath() string {
	if m.lockfilePath != "" {
		return m.lockfilePath
	}
	return filepath.Join(m.projectDir, ".chronicle", "daemon.lock")
}

// checkLockfile reads and validates the lockfile.
func (m *DaemonManager) checkLockfile() (*LockfileInfo, error) {
	path := m.getLockfilePath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var info LockfileInfo
	if err := json.Unmarshal(data, &info); err != nil {
		return nil, err
	}

	// Verify process is still running
	if !isProcessRunning(info.PID) {
		return nil, nil
	}

	return &info, nil
}

// removeLockfile removes the lockfile.
func (m *DaemonManager) removeLockfile() error {
	path := m.getLockfilePath()
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// isProcessRunning checks if a process with the given PID is running.
func isProcessRunning(pid int) bool {
	if pid <= 0 {
		return false
	}

	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}

	// On Unix, FindProcess always succeeds. We need to send signal 0 to check.
	err = process.Signal(syscall.Signal(0))
	return err == nil
}

// CheckDaemonFromLockfile checks if a daemon is running based on lockfile info.
// Returns the client and port if found, nil otherwise.
func CheckDaemonFromLockfile(ctx context.Context, projectDir string) (*Client, int) {
	lockPath := filepath.Join(projectDir, ".chronicle", "daemon.lock")
	data, err := os.ReadFile(lockPath)
	if err != nil {
		return nil, 0
	}

	var info LockfileInfo
	if err := json.Unmarshal(data, &info); err != nil {
		return nil, 0
	}

	if !isProcessRunning(info.PID) {
		// Clean up stale lockfile
		_ = os.Remove(lockPath)
		return nil, 0
	}

	client := New(fmt.Sprintf("http://localhost:%d", info.Port))
	if client.IsHealthy(ctx) {
		return client, info.Port
	}

	return nil, 0
}
