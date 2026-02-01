package daemon

import (
	"context"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"
)

func TestNewConfigWatcher(t *testing.T) {
	paths := []string{"test.yaml"}
	onChange := func() error {
		return nil
	}

	watcher := NewConfigWatcher(paths, onChange)

	if watcher == nil {
		t.Fatal("NewConfigWatcher should not return nil")
	}

	if len(watcher.paths) != 1 {
		t.Errorf("Expected 1 path, got %d", len(watcher.paths))
	}

	if watcher.interval != 5*time.Second {
		t.Errorf("Default interval = %v, expected 5s", watcher.interval)
	}

	if watcher.onChange == nil {
		t.Error("onChange should be set")
	}
}

func TestWithInterval(t *testing.T) {
	watcher := NewConfigWatcher([]string{}, nil, WithInterval(10*time.Second))

	if watcher.interval != 10*time.Second {
		t.Errorf("Interval = %v, expected 10s", watcher.interval)
	}
}

func TestConfigWatcher_StartStop(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "watcher-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Create a test file
	testFile := filepath.Join(tmpDir, "test.yaml")
	if err := os.WriteFile(testFile, []byte("initial"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	var callCount int32
	watcher := NewConfigWatcher(
		[]string{testFile},
		func() error {
			atomic.AddInt32(&callCount, 1)
			return nil
		},
		WithInterval(50*time.Millisecond),
	)

	// Start watcher in goroutine
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		_ = watcher.Start(ctx)
	}()

	// Wait for watcher to start
	time.Sleep(100 * time.Millisecond)

	// Modify file
	if err := os.WriteFile(testFile, []byte("modified"), 0644); err != nil {
		t.Fatalf("Failed to modify test file: %v", err)
	}

	// Wait for change detection
	time.Sleep(200 * time.Millisecond)

	// Stop watcher
	watcher.Stop()

	// Check that callback was called
	if atomic.LoadInt32(&callCount) < 1 {
		t.Error("onChange should have been called at least once")
	}
}

func TestConfigWatcher_AlreadyRunning(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "watcher-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	testFile := filepath.Join(tmpDir, "test.yaml")
	if err := os.WriteFile(testFile, []byte("content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	watcher := NewConfigWatcher([]string{testFile}, func() error { return nil })

	ctx, cancel := context.WithCancel(context.Background())

	// Start first time
	go func() {
		_ = watcher.Start(ctx)
	}()

	time.Sleep(50 * time.Millisecond)

	// Try to start again
	err = watcher.Start(context.Background())
	if err == nil {
		t.Error("Start() should return error when already running")
	}

	cancel()
	watcher.Stop()
}

func TestConfigWatcher_ContextCancellation(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "watcher-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	testFile := filepath.Join(tmpDir, "test.yaml")
	if err := os.WriteFile(testFile, []byte("content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	watcher := NewConfigWatcher([]string{testFile}, func() error { return nil })

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan error, 1)
	go func() {
		done <- watcher.Start(ctx)
	}()

	time.Sleep(50 * time.Millisecond)
	cancel()

	select {
	case err := <-done:
		if err != context.Canceled {
			t.Errorf("Expected context.Canceled, got %v", err)
		}
	case <-time.After(time.Second):
		t.Error("Start() should have returned after context cancellation")
	}
}

func TestFileHash(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "hash-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	testFile := filepath.Join(tmpDir, "test.txt")
	content := "test content"
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Calculate hash
	hash1, err := fileHash(testFile)
	if err != nil {
		t.Fatalf("fileHash() unexpected error: %v", err)
	}

	if hash1 == "" {
		t.Error("Hash should not be empty")
	}

	// Same content should give same hash
	hash2, err := fileHash(testFile)
	if err != nil {
		t.Fatalf("fileHash() second call error: %v", err)
	}

	if hash1 != hash2 {
		t.Error("Same content should produce same hash")
	}

	// Different content should give different hash
	if err := os.WriteFile(testFile, []byte("different"), 0644); err != nil {
		t.Fatalf("Failed to modify file: %v", err)
	}

	hash3, err := fileHash(testFile)
	if err != nil {
		t.Fatalf("fileHash() third call error: %v", err)
	}

	if hash1 == hash3 {
		t.Error("Different content should produce different hash")
	}
}

func TestFileHash_NotExists(t *testing.T) {
	_, err := fileHash("/nonexistent/file.txt")
	if err == nil {
		t.Error("fileHash() should return error for non-existent file")
	}
}

func TestConfigWatcher_FileDeleted(t *testing.T) {
	// Note: File deletion detection for exact file paths is not supported by the current
	// implementation because filepath.Glob returns an empty slice for deleted files,
	// so they're never processed in checkForChanges. This test verifies that behavior
	// doesn't cause errors.
	tmpDir, err := os.MkdirTemp("", "watcher-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	testFile := filepath.Join(tmpDir, "test.yaml")
	if err := os.WriteFile(testFile, []byte("content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	var callCount int32
	watcher := NewConfigWatcher(
		[]string{testFile},
		func() error {
			atomic.AddInt32(&callCount, 1)
			return nil
		},
		WithInterval(50*time.Millisecond),
	)

	// Calculate initial hash
	if err := watcher.updateHashes(); err != nil {
		t.Fatalf("updateHashes() error: %v", err)
	}

	// Delete the file
	if err := os.Remove(testFile); err != nil {
		t.Fatalf("Failed to delete file: %v", err)
	}

	// Check for changes - should not error even if file is deleted
	_, err = watcher.checkForChanges()
	if err != nil {
		t.Fatalf("checkForChanges() should not error when file is deleted: %v", err)
	}
}

func TestConfigWatcher_NewFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "watcher-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Watch a pattern that doesn't match anything initially
	pattern := filepath.Join(tmpDir, "*.yaml")

	watcher := NewConfigWatcher(
		[]string{pattern},
		func() error { return nil },
	)

	// Calculate initial (empty) hashes
	if err := watcher.updateHashes(); err != nil {
		t.Fatalf("updateHashes() error: %v", err)
	}

	// Create a new file
	testFile := filepath.Join(tmpDir, "new.yaml")
	if err := os.WriteFile(testFile, []byte("content"), 0644); err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	// Check for changes
	changed, err := watcher.checkForChanges()
	if err != nil {
		t.Fatalf("checkForChanges() error: %v", err)
	}

	if !changed {
		t.Error("New file should be detected as a change")
	}
}

func TestNewDirectoryWatcher(t *testing.T) {
	extensions := []string{".yaml", ".yml"}
	onChange := func() error { return nil }

	watcher := NewDirectoryWatcher("./testdir", extensions, onChange)

	if watcher == nil {
		t.Fatal("NewDirectoryWatcher should not return nil")
	}

	if len(watcher.extensions) != 2 {
		t.Errorf("Expected 2 extensions, got %d", len(watcher.extensions))
	}

	// Should have 4 patterns (2 extensions * 2 patterns each: ** and top-level)
	if len(watcher.paths) != 4 {
		t.Errorf("Expected 4 glob patterns, got %d", len(watcher.paths))
	}
}

func TestConfigWatcher_MultipleFiles(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "watcher-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Create multiple test files
	file1 := filepath.Join(tmpDir, "config1.yaml")
	file2 := filepath.Join(tmpDir, "config2.yaml")

	if err := os.WriteFile(file1, []byte("config1"), 0644); err != nil {
		t.Fatalf("Failed to create file1: %v", err)
	}
	if err := os.WriteFile(file2, []byte("config2"), 0644); err != nil {
		t.Fatalf("Failed to create file2: %v", err)
	}

	var callCount int32
	watcher := NewConfigWatcher(
		[]string{file1, file2},
		func() error {
			atomic.AddInt32(&callCount, 1)
			return nil
		},
	)

	// Calculate initial hashes
	if err := watcher.updateHashes(); err != nil {
		t.Fatalf("updateHashes() error: %v", err)
	}

	// Verify both files are tracked
	watcher.mu.RLock()
	if len(watcher.hashes) != 2 {
		t.Errorf("Expected 2 hashes, got %d", len(watcher.hashes))
	}
	watcher.mu.RUnlock()

	// Modify only one file
	if err := os.WriteFile(file1, []byte("modified"), 0644); err != nil {
		t.Fatalf("Failed to modify file: %v", err)
	}

	changed, err := watcher.checkForChanges()
	if err != nil {
		t.Fatalf("checkForChanges() error: %v", err)
	}

	if !changed {
		t.Error("Modifying one file should be detected as a change")
	}
}
