package daemon

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// ConfigWatcher watches for configuration file changes.
type ConfigWatcher struct {
	mu sync.RWMutex

	// Paths to watch
	paths []string

	// Poll interval
	interval time.Duration

	// File hashes for change detection
	hashes map[string]string

	// Callback on change
	onChange func() error

	// Stop channel
	stopCh chan struct{}

	// Running state
	running bool
}

// WatcherOption configures a ConfigWatcher.
type WatcherOption func(*ConfigWatcher)

// NewConfigWatcher creates a new configuration watcher.
func NewConfigWatcher(paths []string, onChange func() error, opts ...WatcherOption) *ConfigWatcher {
	w := &ConfigWatcher{
		paths:    paths,
		interval: 5 * time.Second,
		hashes:   make(map[string]string),
		onChange: onChange,
		stopCh:   make(chan struct{}),
	}

	for _, opt := range opts {
		opt(w)
	}

	return w
}

// WithInterval sets the poll interval.
func WithInterval(d time.Duration) WatcherOption {
	return func(w *ConfigWatcher) {
		w.interval = d
	}
}

// Start begins watching for changes.
func (w *ConfigWatcher) Start(ctx context.Context) error {
	w.mu.Lock()
	if w.running {
		w.mu.Unlock()
		return fmt.Errorf("watcher already running")
	}
	w.running = true
	w.mu.Unlock()

	// Calculate initial hashes
	if err := w.updateHashes(); err != nil {
		return fmt.Errorf("initial hash calculation: %w", err)
	}

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-w.stopCh:
			return nil
		case <-ticker.C:
			changed, err := w.checkForChanges()
			if err != nil {
				// Log error but continue watching
				fmt.Printf("Config watcher error: %v\n", err)
				continue
			}

			if changed {
				if err := w.onChange(); err != nil {
					fmt.Printf("Config reload error: %v\n", err)
				}
			}
		}
	}
}

// Stop stops the watcher.
func (w *ConfigWatcher) Stop() {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.running {
		close(w.stopCh)
		w.running = false
	}
}

// updateHashes calculates hashes for all watched files.
func (w *ConfigWatcher) updateHashes() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	for _, path := range w.paths {
		// Expand glob patterns
		matches, err := filepath.Glob(path)
		if err != nil {
			return fmt.Errorf("glob %s: %w", path, err)
		}

		for _, match := range matches {
			hash, err := fileHash(match)
			if err != nil {
				if os.IsNotExist(err) {
					// File was deleted
					delete(w.hashes, match)
					continue
				}
				return fmt.Errorf("hash %s: %w", match, err)
			}
			w.hashes[match] = hash
		}
	}

	return nil
}

// checkForChanges checks if any watched files have changed.
func (w *ConfigWatcher) checkForChanges() (bool, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	changed := false

	for _, path := range w.paths {
		matches, err := filepath.Glob(path)
		if err != nil {
			return false, fmt.Errorf("glob %s: %w", path, err)
		}

		for _, match := range matches {
			hash, err := fileHash(match)
			if err != nil {
				if os.IsNotExist(err) {
					// File was deleted
					if _, ok := w.hashes[match]; ok {
						delete(w.hashes, match)
						changed = true
					}
					continue
				}
				return false, fmt.Errorf("hash %s: %w", match, err)
			}

			oldHash, exists := w.hashes[match]
			if !exists || oldHash != hash {
				w.hashes[match] = hash
				changed = true
			}
		}
	}

	return changed, nil
}

// fileHash calculates the SHA256 hash of a file.
func fileHash(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

// DirectoryWatcher watches a directory for file changes.
type DirectoryWatcher struct {
	*ConfigWatcher
	extensions []string
}

// NewDirectoryWatcher creates a watcher for a directory.
func NewDirectoryWatcher(dir string, extensions []string, onChange func() error, opts ...WatcherOption) *DirectoryWatcher {
	// Build glob patterns for extensions
	var paths []string
	for _, ext := range extensions {
		paths = append(paths, filepath.Join(dir, "**", "*"+ext))
		paths = append(paths, filepath.Join(dir, "*"+ext))
	}

	return &DirectoryWatcher{
		ConfigWatcher: NewConfigWatcher(paths, onChange, opts...),
		extensions:    extensions,
	}
}
