package daemon

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

// LockfileInfo contains daemon instance information.
type LockfileInfo struct {
	PID        int    `json:"pid"`
	Port       int    `json:"port"`
	ProjectDir string `json:"project_dir"`
	StartedAt  string `json:"started_at"`
}

// DefaultLockfilePath returns the default lockfile path for a project.
func DefaultLockfilePath(projectDir string) string {
	return filepath.Join(projectDir, ".chronicle", "daemon.lock")
}

// GlobalLockfilePath returns the user's global lockfile path.
func GlobalLockfilePath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".chronicle/daemon.lock"
	}
	return filepath.Join(home, ".chronicle", "daemon.lock")
}

// WriteLockfile writes daemon info to the lockfile.
func WriteLockfile(path string, info *LockfileInfo) error {
	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create lockfile directory: %w", err)
	}

	data, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal lockfile: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("write lockfile: %w", err)
	}

	return nil
}

// ReadLockfile reads daemon info from the lockfile.
func ReadLockfile(path string) (*LockfileInfo, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read lockfile: %w", err)
	}

	var info LockfileInfo
	if err := json.Unmarshal(data, &info); err != nil {
		return nil, fmt.Errorf("parse lockfile: %w", err)
	}

	return &info, nil
}

// RemoveLockfile removes the lockfile.
func RemoveLockfile(path string) error {
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove lockfile: %w", err)
	}
	return nil
}

// IsProcessRunning checks if a process with the given PID is running.
func IsProcessRunning(pid int) bool {
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

// CheckExistingDaemon checks if there's already a daemon running based on lockfile.
// Returns the lockfile info if a daemon is running, nil otherwise.
func CheckExistingDaemon(projectDir string) (*LockfileInfo, error) {
	// Check project-specific lockfile first
	projectLock := DefaultLockfilePath(projectDir)
	info, err := ReadLockfile(projectLock)
	if err != nil {
		return nil, err
	}

	if info != nil && IsProcessRunning(info.PID) {
		return info, nil
	}

	// Check global lockfile
	globalLock := GlobalLockfilePath()
	info, err = ReadLockfile(globalLock)
	if err != nil {
		return nil, err
	}

	if info != nil && IsProcessRunning(info.PID) {
		return info, nil
	}

	// Clean up stale lockfiles
	_ = RemoveLockfile(projectLock)
	_ = RemoveLockfile(globalLock)

	return nil, nil
}

// ValidateLockfile checks if the lockfile points to a running daemon and cleans up if not.
func ValidateLockfile(path string) (*LockfileInfo, error) {
	info, err := ReadLockfile(path)
	if err != nil {
		return nil, err
	}

	if info == nil {
		return nil, nil
	}

	if !IsProcessRunning(info.PID) {
		// Stale lockfile, remove it
		_ = RemoveLockfile(path)
		return nil, nil
	}

	return info, nil
}

// ErrPortInUse indicates the port is in use by another application.
var ErrPortInUse = errors.New("port in use by another application")

// ErrDaemonAlreadyRunning indicates a daemon is already running.
var ErrDaemonAlreadyRunning = errors.New("daemon already running")
