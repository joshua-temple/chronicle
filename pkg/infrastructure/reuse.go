package infrastructure

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// ReuseManager tracks reusable infrastructure across test runs.
type ReuseManager struct {
	mu        sync.RWMutex
	entries   map[string]*ReuseEntry
	storePath string
	enabled   bool
}

// ReuseEntry represents a reusable infrastructure resource.
type ReuseEntry struct {
	Key        string            `json:"key"`
	Provider   string            `json:"provider"`
	Config     map[string]any    `json:"config"`
	Endpoints  map[string]string `json:"endpoints"` // Service endpoints (e.g., host:port)
	CreatedAt  time.Time         `json:"created_at"`
	LastUsedAt time.Time         `json:"last_used_at"`
	TTL        time.Duration     `json:"ttl"`
	ExpiresAt  time.Time         `json:"expires_at"`
}

// ReuseConfig configures reuse behavior.
type ReuseConfig struct {
	Enabled   bool              `json:"enabled"`
	TTL       time.Duration     `json:"ttl"`
	Key       string            `json:"key"`       // Explicit key, or computed from config
	StorePath string            `json:"store_path"` // Path to store reuse state
	Config    map[string]any    `json:"-"`         // Config used to compute key
}

// DefaultReuseTTL is the default TTL for reusable containers.
const DefaultReuseTTL = 1 * time.Hour

// DefaultReuseStorePath is the default path to store reuse state.
var DefaultReuseStorePath = filepath.Join(os.TempDir(), "chronicle-reuse")

// NewReuseManager creates a new reuse manager.
func NewReuseManager() *ReuseManager {
	return &ReuseManager{
		entries:   make(map[string]*ReuseEntry),
		storePath: DefaultReuseStorePath,
		enabled:   true,
	}
}

// SetStorePath sets the path for persisting reuse state.
func (m *ReuseManager) SetStorePath(path string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.storePath = path
}

// SetEnabled enables or disables the reuse manager.
func (m *ReuseManager) SetEnabled(enabled bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.enabled = enabled
}

// ComputeKey generates a unique key based on provider type and configuration.
func ComputeKey(providerType string, config map[string]any) string {
	// Create a deterministic hash of the configuration
	data := map[string]any{
		"provider": providerType,
		"config":   config,
	}
	bytes, _ := json.Marshal(data)
	hash := sha256.Sum256(bytes)
	return hex.EncodeToString(hash[:])[:16] // Use first 16 chars for brevity
}

// GetOrCreate returns an existing entry or creates a new one.
func (m *ReuseManager) GetOrCreate(ctx context.Context, providerType string, config ReuseConfig) (*ReuseEntry, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.enabled || !config.Enabled {
		return nil, false, nil
	}

	// Compute key if not provided
	key := config.Key
	if key == "" {
		key = ComputeKey(providerType, config.Config)
	}

	// Check for existing entry
	if existingEntry, exists := m.entries[key]; exists {
		// Check if expired
		if time.Now().After(existingEntry.ExpiresAt) {
			delete(m.entries, key)
			// Fall through to create new entry
		} else {
			// Update last used time and extend TTL
			existingEntry.LastUsedAt = time.Now()
			ttl := config.TTL
			if ttl == 0 {
				ttl = DefaultReuseTTL
			}
			existingEntry.ExpiresAt = time.Now().Add(ttl)
			return existingEntry, true, nil
		}
	}

	// Create new entry
	ttl := config.TTL
	if ttl == 0 {
		ttl = DefaultReuseTTL
	}

	newEntry := &ReuseEntry{
		Key:        key,
		Provider:   providerType,
		Config:     config.Config,
		Endpoints:  make(map[string]string),
		CreatedAt:  time.Now(),
		LastUsedAt: time.Now(),
		TTL:        ttl,
		ExpiresAt:  time.Now().Add(ttl),
	}
	m.entries[key] = newEntry

	return newEntry, false, nil
}

// Get retrieves an entry by key.
func (m *ReuseManager) Get(key string) (*ReuseEntry, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	entry, ok := m.entries[key]
	if !ok {
		return nil, false
	}

	// Check if expired
	if time.Now().After(entry.ExpiresAt) {
		return nil, false
	}

	return entry, true
}

// Update updates an entry's endpoints.
func (m *ReuseManager) Update(key string, endpoints map[string]string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	entry, ok := m.entries[key]
	if !ok {
		return errors.New("entry not found")
	}

	entry.Endpoints = endpoints
	entry.LastUsedAt = time.Now()

	return nil
}

// Remove removes an entry by key.
func (m *ReuseManager) Remove(key string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.entries, key)
}

// CleanExpired removes all expired entries.
func (m *ReuseManager) CleanExpired() int {
	m.mu.Lock()
	defer m.mu.Unlock()

	var removed int
	now := time.Now()
	for key, entry := range m.entries {
		if now.After(entry.ExpiresAt) {
			delete(m.entries, key)
			removed++
		}
	}
	return removed
}

// Touch updates the last used time and extends TTL for an entry.
func (m *ReuseManager) Touch(key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	entry, ok := m.entries[key]
	if !ok {
		return errors.New("entry not found")
	}

	entry.LastUsedAt = time.Now()
	entry.ExpiresAt = time.Now().Add(entry.TTL)

	return nil
}

// Save persists the reuse state to disk.
func (m *ReuseManager) Save() error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.storePath == "" {
		return nil // Persistence disabled
	}

	// Ensure directory exists
	if err := os.MkdirAll(m.storePath, 0755); err != nil {
		return fmt.Errorf("failed to create store directory: %w", err)
	}

	// Clean expired before saving
	entries := make(map[string]*ReuseEntry)
	now := time.Now()
	for key, entry := range m.entries {
		if !now.After(entry.ExpiresAt) {
			entries[key] = entry
		}
	}

	// Serialize entries
	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal entries: %w", err)
	}

	// Write to file
	path := filepath.Join(m.storePath, "reuse.json")
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write state file: %w", err)
	}

	return nil
}

// Load restores the reuse state from disk.
func (m *ReuseManager) Load() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.storePath == "" {
		return nil // Persistence disabled
	}

	path := filepath.Join(m.storePath, "reuse.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No state file, start fresh
		}
		return fmt.Errorf("failed to read state file: %w", err)
	}

	var entries map[string]*ReuseEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return fmt.Errorf("failed to unmarshal entries: %w", err)
	}

	// Filter out expired entries
	now := time.Now()
	m.entries = make(map[string]*ReuseEntry)
	for key, entry := range entries {
		if !now.After(entry.ExpiresAt) {
			m.entries[key] = entry
		}
	}

	return nil
}

// Clear removes all entries.
func (m *ReuseManager) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.entries = make(map[string]*ReuseEntry)
}

// Entries returns a snapshot of all entries.
func (m *ReuseManager) Entries() map[string]*ReuseEntry {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[string]*ReuseEntry, len(m.entries))
	for k, v := range m.entries {
		result[k] = v
	}
	return result
}

// IsExpired checks if an entry has expired.
func (e *ReuseEntry) IsExpired() bool {
	return time.Now().After(e.ExpiresAt)
}

// TimeRemaining returns the time remaining until expiration.
func (e *ReuseEntry) TimeRemaining() time.Duration {
	remaining := time.Until(e.ExpiresAt)
	if remaining < 0 {
		return 0
	}
	return remaining
}

// DefaultReuseManager is the global reuse manager.
var DefaultReuseManager = NewReuseManager()
