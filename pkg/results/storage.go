package results

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Storage defines the interface for persisting run results.
type Storage interface {
	// Save persists a run result.
	Save(ctx context.Context, result *RunResult) error

	// Load retrieves a run result by ID.
	Load(ctx context.Context, id string) (*RunResult, error)

	// List returns all run IDs, optionally filtered.
	List(ctx context.Context, opts ...ListOption) ([]string, error)

	// Delete removes a run result by ID.
	Delete(ctx context.Context, id string) error
}

// ListOptions configures the List operation.
type ListOptions struct {
	Limit     int
	After     time.Time
	Before    time.Time
	NameMatch string
	Tags      []string
}

// ListOption configures list options.
type ListOption func(*ListOptions)

// WithListLimit sets the maximum number of results to return.
func WithListLimit(limit int) ListOption {
	return func(o *ListOptions) {
		o.Limit = limit
	}
}

// WithListAfter filters results after the given time.
func WithListAfter(t time.Time) ListOption {
	return func(o *ListOptions) {
		o.After = t
	}
}

// WithListBefore filters results before the given time.
func WithListBefore(t time.Time) ListOption {
	return func(o *ListOptions) {
		o.Before = t
	}
}

// WithListNameMatch filters results by name pattern.
func WithListNameMatch(pattern string) ListOption {
	return func(o *ListOptions) {
		o.NameMatch = pattern
	}
}

// WithListTags filters results that have all the given tags.
func WithListTags(tags []string) ListOption {
	return func(o *ListOptions) {
		o.Tags = tags
	}
}

// FileStorage stores results as JSON files in a directory.
type FileStorage struct {
	dir string
}

// NewFileStorage creates a new file-based storage.
func NewFileStorage(dir string) (*FileStorage, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("create storage directory: %w", err)
	}
	return &FileStorage{dir: dir}, nil
}

// Save writes a result to a JSON file.
func (s *FileStorage) Save(ctx context.Context, result *RunResult) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	filename := filepath.Join(s.dir, result.ID+".json")
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal result: %w", err)
	}

	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("write result file: %w", err)
	}

	return nil
}

// Load reads a result from a JSON file.
func (s *FileStorage) Load(ctx context.Context, id string) (*RunResult, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	filename := filepath.Join(s.dir, id+".json")
	data, err := os.ReadFile(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("result not found: %s", id)
		}
		return nil, fmt.Errorf("read result file: %w", err)
	}

	var result RunResult
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("unmarshal result: %w", err)
	}

	return &result, nil
}

// List returns all run IDs from the storage directory.
func (s *FileStorage) List(ctx context.Context, opts ...ListOption) ([]string, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	options := &ListOptions{}
	for _, opt := range opts {
		opt(options)
	}

	entries, err := os.ReadDir(s.dir)
	if err != nil {
		return nil, fmt.Errorf("read storage directory: %w", err)
	}

	var ids []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if filepath.Ext(name) != ".json" {
			continue
		}

		id := name[:len(name)-5] // Remove .json extension

		// Apply filtering if options are set
		if !options.After.IsZero() || !options.Before.IsZero() || options.NameMatch != "" || len(options.Tags) > 0 {
			result, err := s.Load(ctx, id)
			if err != nil {
				continue // Skip files we can't load
			}

			if !options.After.IsZero() && result.EndTime.Before(options.After) {
				continue
			}
			if !options.Before.IsZero() && result.EndTime.After(options.Before) {
				continue
			}
			if options.NameMatch != "" && !matchesPattern(result.Name, options.NameMatch) {
				continue
			}
			if len(options.Tags) > 0 && !hasAllTags(result.Config.Tags, options.Tags) {
				continue
			}
		}

		ids = append(ids, id)

		if options.Limit > 0 && len(ids) >= options.Limit {
			break
		}
	}

	return ids, nil
}

// Delete removes a result file.
func (s *FileStorage) Delete(ctx context.Context, id string) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	filename := filepath.Join(s.dir, id+".json")
	if err := os.Remove(filename); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("result not found: %s", id)
		}
		return fmt.Errorf("delete result file: %w", err)
	}

	return nil
}

// MemoryStorage stores results in memory (useful for testing).
type MemoryStorage struct {
	mu      sync.RWMutex
	results map[string]*RunResult
}

// NewMemoryStorage creates a new in-memory storage.
func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{
		results: make(map[string]*RunResult),
	}
}

// Save stores a result in memory.
func (s *MemoryStorage) Save(_ context.Context, result *RunResult) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Make a copy to prevent external modification
	copy := *result
	s.results[result.ID] = &copy
	return nil
}

// Load retrieves a result from memory.
func (s *MemoryStorage) Load(_ context.Context, id string) (*RunResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result, ok := s.results[id]
	if !ok {
		return nil, fmt.Errorf("result not found: %s", id)
	}

	// Return a copy to prevent external modification
	copy := *result
	return &copy, nil
}

// List returns all run IDs from memory.
func (s *MemoryStorage) List(_ context.Context, opts ...ListOption) ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	options := &ListOptions{}
	for _, opt := range opts {
		opt(options)
	}

	var ids []string
	for id, result := range s.results {
		if !options.After.IsZero() && result.EndTime.Before(options.After) {
			continue
		}
		if !options.Before.IsZero() && result.EndTime.After(options.Before) {
			continue
		}
		if options.NameMatch != "" && !matchesPattern(result.Name, options.NameMatch) {
			continue
		}
		if len(options.Tags) > 0 && !hasAllTags(result.Config.Tags, options.Tags) {
			continue
		}

		ids = append(ids, id)

		if options.Limit > 0 && len(ids) >= options.Limit {
			break
		}
	}

	return ids, nil
}

// Delete removes a result from memory.
func (s *MemoryStorage) Delete(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.results[id]; !ok {
		return fmt.Errorf("result not found: %s", id)
	}

	delete(s.results, id)
	return nil
}

// Clear removes all results from memory.
func (s *MemoryStorage) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.results = make(map[string]*RunResult)
}

// Count returns the number of stored results.
func (s *MemoryStorage) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.results)
}

// matchesPattern checks if a string matches a simple pattern.
// Supports * as wildcard at the start or end of the pattern.
func matchesPattern(s, pattern string) bool {
	if pattern == "" || pattern == "*" {
		return true
	}

	if pattern[0] == '*' {
		// Suffix match
		suffix := pattern[1:]
		return len(s) >= len(suffix) && s[len(s)-len(suffix):] == suffix
	}

	if pattern[len(pattern)-1] == '*' {
		// Prefix match
		prefix := pattern[:len(pattern)-1]
		return len(s) >= len(prefix) && s[:len(prefix)] == prefix
	}

	// Exact match
	return s == pattern
}

// hasAllTags checks if all required tags are present.
func hasAllTags(have, want []string) bool {
	tagSet := make(map[string]bool)
	for _, t := range have {
		tagSet[t] = true
	}

	for _, t := range want {
		if !tagSet[t] {
			return false
		}
	}
	return true
}

// MultiStorage writes to multiple storage backends.
type MultiStorage struct {
	storages []Storage
}

// NewMultiStorage creates a storage that writes to multiple backends.
func NewMultiStorage(storages ...Storage) *MultiStorage {
	return &MultiStorage{storages: storages}
}

// Save writes to all backends.
func (s *MultiStorage) Save(ctx context.Context, result *RunResult) error {
	var errs []error
	for _, storage := range s.storages {
		if err := storage.Save(ctx, result); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("multi-storage save errors: %v", errs)
	}
	return nil
}

// Load reads from the first backend that has the result.
func (s *MultiStorage) Load(ctx context.Context, id string) (*RunResult, error) {
	for _, storage := range s.storages {
		result, err := storage.Load(ctx, id)
		if err == nil {
			return result, nil
		}
	}
	return nil, fmt.Errorf("result not found in any storage: %s", id)
}

// List returns IDs from the first storage.
func (s *MultiStorage) List(ctx context.Context, opts ...ListOption) ([]string, error) {
	if len(s.storages) == 0 {
		return nil, nil
	}
	return s.storages[0].List(ctx, opts...)
}

// Delete removes from all backends.
func (s *MultiStorage) Delete(ctx context.Context, id string) error {
	var errs []error
	for _, storage := range s.storages {
		if err := storage.Delete(ctx, id); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("multi-storage delete errors: %v", errs)
	}
	return nil
}

// StorageWithRetention wraps a storage with automatic retention policy.
type StorageWithRetention struct {
	Storage
	maxAge   time.Duration
	maxCount int
}

// NewStorageWithRetention wraps a storage with retention policy.
func NewStorageWithRetention(s Storage, maxAge time.Duration, maxCount int) *StorageWithRetention {
	return &StorageWithRetention{
		Storage:  s,
		maxAge:   maxAge,
		maxCount: maxCount,
	}
}

// Save stores the result and enforces retention policy.
func (s *StorageWithRetention) Save(ctx context.Context, result *RunResult) error {
	if err := s.Storage.Save(ctx, result); err != nil {
		return err
	}

	// Enforce retention in the background
	go s.enforceRetention(context.Background())
	return nil
}

// enforceRetention removes old results based on the retention policy.
func (s *StorageWithRetention) enforceRetention(ctx context.Context) {
	ids, err := s.List(ctx)
	if err != nil {
		return
	}

	// Remove by age
	if s.maxAge > 0 {
		cutoff := time.Now().Add(-s.maxAge)
		for _, id := range ids {
			result, err := s.Load(ctx, id)
			if err != nil {
				continue
			}
			if result.EndTime.Before(cutoff) {
				_ = s.Delete(ctx, id)
			}
		}
	}

	// Remove by count (keep newest)
	if s.maxCount > 0 {
		ids, err = s.List(ctx) // Refresh list after age cleanup
		if err != nil {
			return
		}

		if len(ids) > s.maxCount {
			// This is a simple approach - for production use, should sort by time
			toDelete := len(ids) - s.maxCount
			for i := 0; i < toDelete; i++ {
				_ = s.Delete(ctx, ids[i])
			}
		}
	}
}
