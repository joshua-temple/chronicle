package infrastructure

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestComputeKey(t *testing.T) {
	t.Run("generates consistent key", func(t *testing.T) {
		config := map[string]any{
			"image": "postgres:15",
			"port":  5432,
		}

		key1 := ComputeKey("postgres", config)
		key2 := ComputeKey("postgres", config)

		if key1 != key2 {
			t.Errorf("expected consistent keys, got %s and %s", key1, key2)
		}
	})

	t.Run("different configs produce different keys", func(t *testing.T) {
		config1 := map[string]any{"image": "postgres:15"}
		config2 := map[string]any{"image": "postgres:16"}

		key1 := ComputeKey("postgres", config1)
		key2 := ComputeKey("postgres", config2)

		if key1 == key2 {
			t.Error("expected different keys for different configs")
		}
	})

	t.Run("different providers produce different keys", func(t *testing.T) {
		config := map[string]any{"port": 6379}

		key1 := ComputeKey("redis", config)
		key2 := ComputeKey("memcached", config)

		if key1 == key2 {
			t.Error("expected different keys for different providers")
		}
	})
}

func TestReuseManager(t *testing.T) {
	ctx := context.Background()

	t.Run("creates new entry when none exists", func(t *testing.T) {
		rm := NewReuseManager()

		config := ReuseConfig{
			Enabled: true,
			TTL:     time.Hour,
			Config:  map[string]any{"image": "postgres:15"},
		}

		entry, existed, err := rm.GetOrCreate(ctx, "postgres", config)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if existed {
			t.Error("expected new entry")
		}
		if entry == nil {
			t.Fatal("expected entry")
		}
		if entry.Provider != "postgres" {
			t.Errorf("expected provider 'postgres', got %s", entry.Provider)
		}
	})

	t.Run("returns existing entry", func(t *testing.T) {
		rm := NewReuseManager()

		config := ReuseConfig{
			Enabled: true,
			TTL:     time.Hour,
			Config:  map[string]any{"image": "postgres:15"},
		}

		entry1, existed1, err := rm.GetOrCreate(ctx, "postgres", config)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if existed1 {
			t.Error("expected new entry on first call")
		}

		entry2, existed2, err := rm.GetOrCreate(ctx, "postgres", config)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !existed2 {
			t.Error("expected existing entry on second call")
		}
		if entry1.Key != entry2.Key {
			t.Error("expected same entry")
		}
	})

	t.Run("returns nil when disabled", func(t *testing.T) {
		rm := NewReuseManager()

		config := ReuseConfig{
			Enabled: false,
			Config:  map[string]any{"image": "postgres:15"},
		}

		entry, existed, err := rm.GetOrCreate(ctx, "postgres", config)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if existed {
			t.Error("expected not existed")
		}
		if entry != nil {
			t.Error("expected nil entry when disabled")
		}
	})

	t.Run("respects explicit key", func(t *testing.T) {
		rm := NewReuseManager()

		config := ReuseConfig{
			Enabled: true,
			TTL:     time.Hour,
			Key:     "my-custom-key",
			Config:  map[string]any{"image": "postgres:15"},
		}

		entry, _, err := rm.GetOrCreate(ctx, "postgres", config)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if entry.Key != "my-custom-key" {
			t.Errorf("expected key 'my-custom-key', got %s", entry.Key)
		}
	})

	t.Run("expires entries", func(t *testing.T) {
		rm := NewReuseManager()

		config := ReuseConfig{
			Enabled: true,
			TTL:     1 * time.Millisecond, // Very short TTL
			Config:  map[string]any{"image": "postgres:15"},
		}

		entry1, existed1, err := rm.GetOrCreate(ctx, "postgres", config)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if existed1 {
			t.Error("expected new entry")
		}

		// Wait for expiration
		time.Sleep(10 * time.Millisecond)

		_, existed2, err := rm.GetOrCreate(ctx, "postgres", config)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if existed2 {
			t.Error("expected entry to be expired")
		}
		_ = entry1 // silence unused warning
	})

	t.Run("updates endpoints", func(t *testing.T) {
		rm := NewReuseManager()

		config := ReuseConfig{
			Enabled: true,
			TTL:     time.Hour,
			Key:     "test-key",
			Config:  map[string]any{},
		}

		_, _, _ = rm.GetOrCreate(ctx, "postgres", config)

		endpoints := map[string]string{
			"default": "localhost:5432",
		}
		if err := rm.Update("test-key", endpoints); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		entry, ok := rm.Get("test-key")
		if !ok {
			t.Fatal("expected entry")
		}
		if entry.Endpoints["default"] != "localhost:5432" {
			t.Errorf("expected endpoint 'localhost:5432', got %s", entry.Endpoints["default"])
		}
	})

	t.Run("removes entry", func(t *testing.T) {
		rm := NewReuseManager()

		config := ReuseConfig{
			Enabled: true,
			TTL:     time.Hour,
			Key:     "test-key",
			Config:  map[string]any{},
		}

		_, _, _ = rm.GetOrCreate(ctx, "postgres", config)
		rm.Remove("test-key")

		_, ok := rm.Get("test-key")
		if ok {
			t.Error("expected entry to be removed")
		}
	})

	t.Run("cleans expired entries", func(t *testing.T) {
		rm := NewReuseManager()

		// Create one entry with short TTL
		config1 := ReuseConfig{
			Enabled: true,
			TTL:     1 * time.Millisecond,
			Key:     "expires-soon",
			Config:  map[string]any{},
		}
		_, _, _ = rm.GetOrCreate(ctx, "postgres", config1)

		// Create one entry with long TTL
		config2 := ReuseConfig{
			Enabled: true,
			TTL:     time.Hour,
			Key:     "expires-later",
			Config:  map[string]any{},
		}
		_, _, _ = rm.GetOrCreate(ctx, "postgres", config2)

		// Wait for short TTL to expire
		time.Sleep(10 * time.Millisecond)

		removed := rm.CleanExpired()
		if removed != 1 {
			t.Errorf("expected 1 removed, got %d", removed)
		}

		_, ok1 := rm.Get("expires-soon")
		if ok1 {
			t.Error("expected expired entry to be removed")
		}

		_, ok2 := rm.Get("expires-later")
		if !ok2 {
			t.Error("expected non-expired entry to remain")
		}
	})

	t.Run("touches entry", func(t *testing.T) {
		rm := NewReuseManager()

		config := ReuseConfig{
			Enabled: true,
			TTL:     100 * time.Millisecond,
			Key:     "test-key",
			Config:  map[string]any{},
		}

		_, _, _ = rm.GetOrCreate(ctx, "postgres", config)
		time.Sleep(50 * time.Millisecond)

		// Touch to extend TTL
		if err := rm.Touch("test-key"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Wait another 75ms (would have expired without touch)
		time.Sleep(75 * time.Millisecond)

		_, ok := rm.Get("test-key")
		if !ok {
			t.Error("expected entry to still exist after touch")
		}
	})

	t.Run("clears all entries", func(t *testing.T) {
		rm := NewReuseManager()

		config := ReuseConfig{
			Enabled: true,
			TTL:     time.Hour,
			Config:  map[string]any{},
		}
		_, _, _ = rm.GetOrCreate(ctx, "postgres", config)
		_, _, _ = rm.GetOrCreate(ctx, "redis", config)

		rm.Clear()

		entries := rm.Entries()
		if len(entries) != 0 {
			t.Errorf("expected 0 entries, got %d", len(entries))
		}
	})
}

func TestReuseEntry(t *testing.T) {
	t.Run("IsExpired returns true for expired entry", func(t *testing.T) {
		entry := &ReuseEntry{
			ExpiresAt: time.Now().Add(-time.Hour),
		}
		if !entry.IsExpired() {
			t.Error("expected entry to be expired")
		}
	})

	t.Run("IsExpired returns false for non-expired entry", func(t *testing.T) {
		entry := &ReuseEntry{
			ExpiresAt: time.Now().Add(time.Hour),
		}
		if entry.IsExpired() {
			t.Error("expected entry to not be expired")
		}
	})

	t.Run("TimeRemaining returns positive for non-expired entry", func(t *testing.T) {
		entry := &ReuseEntry{
			ExpiresAt: time.Now().Add(time.Hour),
		}
		remaining := entry.TimeRemaining()
		if remaining <= 0 {
			t.Error("expected positive time remaining")
		}
	})

	t.Run("TimeRemaining returns zero for expired entry", func(t *testing.T) {
		entry := &ReuseEntry{
			ExpiresAt: time.Now().Add(-time.Hour),
		}
		remaining := entry.TimeRemaining()
		if remaining != 0 {
			t.Errorf("expected 0, got %v", remaining)
		}
	})
}

func TestReuseManagerPersistence(t *testing.T) {
	ctx := context.Background()

	t.Run("saves and loads state", func(t *testing.T) {
		tmpDir := t.TempDir()

		rm1 := NewReuseManager()
		rm1.SetStorePath(tmpDir)

		config := ReuseConfig{
			Enabled: true,
			TTL:     time.Hour,
			Key:     "persistent-key",
			Config:  map[string]any{"image": "postgres:15"},
		}

		_, _, err := rm1.GetOrCreate(ctx, "postgres", config)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		endpoints := map[string]string{"default": "localhost:5432"}
		if err := rm1.Update("persistent-key", endpoints); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Save state
		if err := rm1.Save(); err != nil {
			t.Fatalf("unexpected error saving: %v", err)
		}

		// Verify file exists
		if _, err := os.Stat(filepath.Join(tmpDir, "reuse.json")); err != nil {
			t.Fatalf("state file not created: %v", err)
		}

		// Load into new manager
		rm2 := NewReuseManager()
		rm2.SetStorePath(tmpDir)
		if err := rm2.Load(); err != nil {
			t.Fatalf("unexpected error loading: %v", err)
		}

		entry, ok := rm2.Get("persistent-key")
		if !ok {
			t.Fatal("expected entry to be loaded")
		}
		if entry.Provider != "postgres" {
			t.Errorf("expected provider 'postgres', got %s", entry.Provider)
		}
		if entry.Endpoints["default"] != "localhost:5432" {
			t.Errorf("expected endpoint 'localhost:5432', got %s", entry.Endpoints["default"])
		}
	})

	t.Run("load with no file returns nil error", func(t *testing.T) {
		tmpDir := t.TempDir()

		rm := NewReuseManager()
		rm.SetStorePath(tmpDir)

		if err := rm.Load(); err != nil {
			t.Errorf("expected nil error, got %v", err)
		}
	})

	t.Run("expired entries not loaded", func(t *testing.T) {
		tmpDir := t.TempDir()

		rm1 := NewReuseManager()
		rm1.SetStorePath(tmpDir)

		config := ReuseConfig{
			Enabled: true,
			TTL:     1 * time.Millisecond,
			Key:     "expires-quick",
			Config:  map[string]any{},
		}

		_, _, _ = rm1.GetOrCreate(ctx, "postgres", config)

		// Save before expiration
		if err := rm1.Save(); err != nil {
			t.Fatalf("unexpected error saving: %v", err)
		}

		// Wait for expiration
		time.Sleep(10 * time.Millisecond)

		// Load into new manager
		rm2 := NewReuseManager()
		rm2.SetStorePath(tmpDir)
		if err := rm2.Load(); err != nil {
			t.Fatalf("unexpected error loading: %v", err)
		}

		_, ok := rm2.Get("expires-quick")
		if ok {
			t.Error("expected expired entry to not be loaded")
		}
	})
}
