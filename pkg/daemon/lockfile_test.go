package daemon

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteAndReadLockfile(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()
	lockPath := filepath.Join(tmpDir, "daemon.lock")

	info := &LockfileInfo{
		PID:        12345,
		Port:       3000,
		ProjectDir: "/path/to/project",
		StartedAt:  "2024-01-01T00:00:00Z",
	}

	// Write lockfile
	if err := WriteLockfile(lockPath, info); err != nil {
		t.Fatalf("WriteLockfile failed: %v", err)
	}

	// Read lockfile
	readInfo, err := ReadLockfile(lockPath)
	if err != nil {
		t.Fatalf("ReadLockfile failed: %v", err)
	}

	if readInfo == nil {
		t.Fatal("ReadLockfile returned nil")
	}

	if readInfo.PID != info.PID {
		t.Errorf("PID mismatch: got %d, want %d", readInfo.PID, info.PID)
	}

	if readInfo.Port != info.Port {
		t.Errorf("Port mismatch: got %d, want %d", readInfo.Port, info.Port)
	}

	if readInfo.ProjectDir != info.ProjectDir {
		t.Errorf("ProjectDir mismatch: got %s, want %s", readInfo.ProjectDir, info.ProjectDir)
	}

	if readInfo.StartedAt != info.StartedAt {
		t.Errorf("StartedAt mismatch: got %s, want %s", readInfo.StartedAt, info.StartedAt)
	}
}

func TestReadLockfile_NotExist(t *testing.T) {
	info, err := ReadLockfile("/nonexistent/path/daemon.lock")
	if err != nil {
		t.Fatalf("ReadLockfile should not error for non-existent file: %v", err)
	}
	if info != nil {
		t.Error("ReadLockfile should return nil for non-existent file")
	}
}

func TestRemoveLockfile(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()
	lockPath := filepath.Join(tmpDir, "daemon.lock")

	// Write lockfile
	info := &LockfileInfo{PID: 12345, Port: 3000}
	if err := WriteLockfile(lockPath, info); err != nil {
		t.Fatalf("WriteLockfile failed: %v", err)
	}

	// Verify it exists
	if _, err := os.Stat(lockPath); os.IsNotExist(err) {
		t.Fatal("Lockfile should exist after write")
	}

	// Remove lockfile
	if err := RemoveLockfile(lockPath); err != nil {
		t.Fatalf("RemoveLockfile failed: %v", err)
	}

	// Verify it's gone
	if _, err := os.Stat(lockPath); !os.IsNotExist(err) {
		t.Error("Lockfile should not exist after remove")
	}
}

func TestRemoveLockfile_NotExist(t *testing.T) {
	// Should not error when file doesn't exist
	err := RemoveLockfile("/nonexistent/path/daemon.lock")
	if err != nil {
		t.Errorf("RemoveLockfile should not error for non-existent file: %v", err)
	}
}

func TestIsProcessRunning(t *testing.T) {
	t.Run("current process is running", func(t *testing.T) {
		pid := os.Getpid()
		if !IsProcessRunning(pid) {
			t.Error("Current process should be running")
		}
	})

	t.Run("invalid PID", func(t *testing.T) {
		if IsProcessRunning(0) {
			t.Error("PID 0 should not be running")
		}
		if IsProcessRunning(-1) {
			t.Error("Negative PID should not be running")
		}
	})

	t.Run("non-existent PID", func(t *testing.T) {
		// Use a very high PID that's unlikely to exist
		if IsProcessRunning(999999999) {
			t.Error("Non-existent PID should not be running")
		}
	})
}

func TestDefaultLockfilePath(t *testing.T) {
	path := DefaultLockfilePath("/path/to/project")
	expected := filepath.Join("/path/to/project", ".chronicle", "daemon.lock")
	if path != expected {
		t.Errorf("DefaultLockfilePath: got %s, want %s", path, expected)
	}
}

func TestValidateLockfile(t *testing.T) {
	t.Run("valid lockfile with running process", func(t *testing.T) {
		tmpDir := t.TempDir()
		lockPath := filepath.Join(tmpDir, "daemon.lock")

		// Write lockfile with current PID
		info := &LockfileInfo{
			PID:  os.Getpid(),
			Port: 3000,
		}
		if err := WriteLockfile(lockPath, info); err != nil {
			t.Fatalf("WriteLockfile failed: %v", err)
		}

		// Validate
		result, err := ValidateLockfile(lockPath)
		if err != nil {
			t.Fatalf("ValidateLockfile failed: %v", err)
		}
		if result == nil {
			t.Error("ValidateLockfile should return info for running process")
		}
	})

	t.Run("stale lockfile gets cleaned up", func(t *testing.T) {
		tmpDir := t.TempDir()
		lockPath := filepath.Join(tmpDir, "daemon.lock")

		// Write lockfile with non-existent PID
		info := &LockfileInfo{
			PID:  999999999,
			Port: 3000,
		}
		if err := WriteLockfile(lockPath, info); err != nil {
			t.Fatalf("WriteLockfile failed: %v", err)
		}

		// Validate - should return nil and clean up
		result, err := ValidateLockfile(lockPath)
		if err != nil {
			t.Fatalf("ValidateLockfile failed: %v", err)
		}
		if result != nil {
			t.Error("ValidateLockfile should return nil for non-running process")
		}

		// Verify lockfile was removed
		if _, err := os.Stat(lockPath); !os.IsNotExist(err) {
			t.Error("Stale lockfile should be removed")
		}
	})

	t.Run("non-existent lockfile", func(t *testing.T) {
		result, err := ValidateLockfile("/nonexistent/path/daemon.lock")
		if err != nil {
			t.Fatalf("ValidateLockfile failed: %v", err)
		}
		if result != nil {
			t.Error("ValidateLockfile should return nil for non-existent file")
		}
	})
}

func TestWriteLockfile_CreatesDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	lockPath := filepath.Join(tmpDir, "subdir", "daemon.lock")

	info := &LockfileInfo{PID: 12345, Port: 3000}
	if err := WriteLockfile(lockPath, info); err != nil {
		t.Fatalf("WriteLockfile failed to create directory: %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(lockPath); os.IsNotExist(err) {
		t.Error("Lockfile should be created in new directory")
	}
}
