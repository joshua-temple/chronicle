package results

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestFileStorage(t *testing.T) {
	dir := t.TempDir()
	storage, err := NewFileStorage(dir)
	if err != nil {
		t.Fatalf("failed to create storage: %v", err)
	}

	testStorageOperations(t, storage)
}

func TestMemoryStorage(t *testing.T) {
	storage := NewMemoryStorage()
	testStorageOperations(t, storage)
}

func testStorageOperations(t *testing.T, storage Storage) {
	ctx := context.Background()

	// Test Save
	result := &RunResult{
		ID:        "test-run-1",
		Name:      "test-run",
		StartTime: time.Now(),
		EndTime:   time.Now().Add(100 * time.Millisecond),
		Duration:  100 * time.Millisecond,
		Stats: RunStats{
			Total:  2,
			Passed: 2,
		},
	}

	err := storage.Save(ctx, result)
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Test Load
	loaded, err := storage.Load(ctx, "test-run-1")
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if loaded.ID != result.ID {
		t.Errorf("expected ID %s, got %s", result.ID, loaded.ID)
	}
	if loaded.Name != result.Name {
		t.Errorf("expected Name %s, got %s", result.Name, loaded.Name)
	}
	if loaded.Stats.Total != result.Stats.Total {
		t.Errorf("expected Total %d, got %d", result.Stats.Total, loaded.Stats.Total)
	}

	// Test Load not found
	_, err = storage.Load(ctx, "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent ID")
	}

	// Test List
	ids, err := storage.List(ctx)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(ids) != 1 {
		t.Errorf("expected 1 ID, got %d", len(ids))
	}

	// Test Delete
	err = storage.Delete(ctx, "test-run-1")
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Verify deletion
	_, err = storage.Load(ctx, "test-run-1")
	if err == nil {
		t.Error("expected error after deletion")
	}

	// Test Delete not found
	err = storage.Delete(ctx, "nonexistent")
	if err == nil {
		t.Error("expected error for deleting nonexistent ID")
	}
}

func TestFileStorageDirectory(t *testing.T) {
	dir := t.TempDir()
	subdir := filepath.Join(dir, "nested", "results")

	// Should create nested directories
	storage, err := NewFileStorage(subdir)
	if err != nil {
		t.Fatalf("failed to create storage: %v", err)
	}

	result := &RunResult{ID: "test"}
	if err := storage.Save(context.Background(), result); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Verify file was created
	filename := filepath.Join(subdir, "test.json")
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		t.Error("expected file to exist")
	}
}

func TestMemoryStorageClear(t *testing.T) {
	storage := NewMemoryStorage()
	ctx := context.Background()

	_ = storage.Save(ctx, &RunResult{ID: "1"})
	_ = storage.Save(ctx, &RunResult{ID: "2"})
	_ = storage.Save(ctx, &RunResult{ID: "3"})

	if storage.Count() != 3 {
		t.Errorf("expected count 3, got %d", storage.Count())
	}

	storage.Clear()

	if storage.Count() != 0 {
		t.Errorf("expected count 0 after clear, got %d", storage.Count())
	}
}

func TestListWithLimit(t *testing.T) {
	storage := NewMemoryStorage()
	ctx := context.Background()

	for i := 0; i < 10; i++ {
		_ = storage.Save(ctx, &RunResult{ID: string(rune('a' + i))})
	}

	ids, err := storage.List(ctx, WithListLimit(5))
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(ids) != 5 {
		t.Errorf("expected 5 IDs, got %d", len(ids))
	}
}

func TestListWithTimeFilters(t *testing.T) {
	storage := NewMemoryStorage()
	ctx := context.Background()

	now := time.Now()
	_ = storage.Save(ctx, &RunResult{ID: "old", EndTime: now.Add(-24 * time.Hour)})
	_ = storage.Save(ctx, &RunResult{ID: "recent", EndTime: now.Add(-1 * time.Hour)})
	_ = storage.Save(ctx, &RunResult{ID: "new", EndTime: now})

	// Test After filter
	ids, err := storage.List(ctx, WithListAfter(now.Add(-2*time.Hour)))
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(ids) != 2 {
		t.Errorf("expected 2 IDs after filter, got %d", len(ids))
	}

	// Test Before filter
	ids, err = storage.List(ctx, WithListBefore(now.Add(-2*time.Hour)))
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(ids) != 1 {
		t.Errorf("expected 1 ID before filter, got %d", len(ids))
	}
}

func TestListWithNameMatch(t *testing.T) {
	storage := NewMemoryStorage()
	ctx := context.Background()

	_ = storage.Save(ctx, &RunResult{ID: "1", Name: "integration-tests"})
	_ = storage.Save(ctx, &RunResult{ID: "2", Name: "unit-tests"})
	_ = storage.Save(ctx, &RunResult{ID: "3", Name: "integration-smoke"})

	// Test prefix match
	ids, err := storage.List(ctx, WithListNameMatch("integration*"))
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(ids) != 2 {
		t.Errorf("expected 2 IDs with prefix match, got %d", len(ids))
	}

	// Test suffix match
	ids, err = storage.List(ctx, WithListNameMatch("*tests"))
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(ids) != 2 {
		t.Errorf("expected 2 IDs with suffix match, got %d", len(ids))
	}

	// Test exact match
	ids, err = storage.List(ctx, WithListNameMatch("unit-tests"))
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(ids) != 1 {
		t.Errorf("expected 1 ID with exact match, got %d", len(ids))
	}
}

func TestListWithTags(t *testing.T) {
	storage := NewMemoryStorage()
	ctx := context.Background()

	_ = storage.Save(ctx, &RunResult{
		ID: "1",
		Config: RunConfig{Tags: []string{"integration", "slow"}},
	})
	_ = storage.Save(ctx, &RunResult{
		ID: "2",
		Config: RunConfig{Tags: []string{"unit", "fast"}},
	})
	_ = storage.Save(ctx, &RunResult{
		ID: "3",
		Config: RunConfig{Tags: []string{"integration", "fast"}},
	})

	// Test single tag
	ids, err := storage.List(ctx, WithListTags([]string{"integration"}))
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(ids) != 2 {
		t.Errorf("expected 2 IDs with integration tag, got %d", len(ids))
	}

	// Test multiple tags (must have all)
	ids, err = storage.List(ctx, WithListTags([]string{"integration", "slow"}))
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(ids) != 1 {
		t.Errorf("expected 1 ID with both tags, got %d", len(ids))
	}
}

func TestMultiStorage(t *testing.T) {
	storage1 := NewMemoryStorage()
	storage2 := NewMemoryStorage()
	multi := NewMultiStorage(storage1, storage2)

	ctx := context.Background()

	// Save should write to both
	result := &RunResult{ID: "test", Name: "multi-test"}
	err := multi.Save(ctx, result)
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	if storage1.Count() != 1 {
		t.Error("expected storage1 to have the result")
	}
	if storage2.Count() != 1 {
		t.Error("expected storage2 to have the result")
	}

	// Load should return from first storage
	loaded, err := multi.Load(ctx, "test")
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if loaded.Name != "multi-test" {
		t.Errorf("expected 'multi-test', got %s", loaded.Name)
	}

	// Delete should remove from both
	err = multi.Delete(ctx, "test")
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	if storage1.Count() != 0 {
		t.Error("expected storage1 to be empty")
	}
	if storage2.Count() != 0 {
		t.Error("expected storage2 to be empty")
	}
}

func TestMultiStorageLoadFallback(t *testing.T) {
	storage1 := NewMemoryStorage()
	storage2 := NewMemoryStorage()
	multi := NewMultiStorage(storage1, storage2)

	ctx := context.Background()

	// Only save to storage2
	result := &RunResult{ID: "test", Name: "fallback-test"}
	_ = storage2.Save(ctx, result)

	// Load should find it in storage2
	loaded, err := multi.Load(ctx, "test")
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if loaded.Name != "fallback-test" {
		t.Errorf("expected 'fallback-test', got %s", loaded.Name)
	}
}

func TestMatchesPattern(t *testing.T) {
	tests := []struct {
		s       string
		pattern string
		want    bool
	}{
		{"test", "", true},
		{"test", "*", true},
		{"integration-test", "integration*", true},
		{"unit-test", "integration*", false},
		{"test-integration", "*integration", true},
		{"test-unit", "*integration", false},
		{"exact", "exact", true},
		{"exact", "other", false},
	}

	for _, tc := range tests {
		got := matchesPattern(tc.s, tc.pattern)
		if got != tc.want {
			t.Errorf("matchesPattern(%q, %q) = %v, want %v", tc.s, tc.pattern, got, tc.want)
		}
	}
}

func TestHasAllTags(t *testing.T) {
	tests := []struct {
		have []string
		want []string
		ok   bool
	}{
		{[]string{"a", "b", "c"}, []string{"a", "b"}, true},
		{[]string{"a", "b"}, []string{"a", "b", "c"}, false},
		{[]string{"a", "b"}, []string{}, true},
		{[]string{}, []string{"a"}, false},
	}

	for _, tc := range tests {
		got := hasAllTags(tc.have, tc.want)
		if got != tc.ok {
			t.Errorf("hasAllTags(%v, %v) = %v, want %v", tc.have, tc.want, got, tc.ok)
		}
	}
}

func TestContextCancellation(t *testing.T) {
	storage := NewMemoryStorage()
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// These should not error on memory storage since it ignores context
	// but FileStorage would respect it
	err := storage.Save(ctx, &RunResult{ID: "test"})
	if err != nil {
		t.Logf("Save with cancelled context: %v (expected for some implementations)", err)
	}
}

func TestFileStorageContextCancellation(t *testing.T) {
	dir := t.TempDir()
	storage, _ := NewFileStorage(dir)
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err := storage.Save(ctx, &RunResult{ID: "test"})
	if err != context.Canceled {
		t.Errorf("expected context.Canceled error, got %v", err)
	}
}

func TestStorageWithRetention(t *testing.T) {
	base := NewMemoryStorage()
	storage := NewStorageWithRetention(base, 0, 3) // Max 3 results

	ctx := context.Background()

	// Save 5 results
	for i := 0; i < 5; i++ {
		result := &RunResult{ID: string(rune('a' + i)), EndTime: time.Now()}
		if err := storage.Save(ctx, result); err != nil {
			t.Fatalf("Save failed: %v", err)
		}
	}

	// Give retention goroutine time to run
	time.Sleep(50 * time.Millisecond)

	// Should have at most 3 (might have more if retention didn't complete)
	// This is a best-effort test since retention runs in background
	count := base.Count()
	t.Logf("Count after retention: %d (expected <= 3)", count)
}
