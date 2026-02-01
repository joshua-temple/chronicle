package standalone

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestRegistry_AddProject(t *testing.T) {
	t.Run("adds project successfully", func(t *testing.T) {
		tempDir := t.TempDir()
		registryPath := filepath.Join(tempDir, "projects.json")

		r, err := NewRegistry(registryPath)
		if err != nil {
			t.Fatalf("NewRegistry failed: %v", err)
		}

		project := Project{
			Name:           "Test Project",
			Path:           "/path/to/project",
			RemoteURL:      "https://github.com/user/repo",
			AutoDiscovered: false,
		}

		id, err := r.Add(project)
		if err != nil {
			t.Fatalf("Add failed: %v", err)
		}

		// Verify ID was generated
		if id == "" {
			t.Error("Expected non-empty ID")
		}

		// Verify ID is a valid UUID
		if _, err := uuid.Parse(id); err != nil {
			t.Errorf("Expected valid UUID, got: %v", err)
		}

		// Verify project was added
		added := r.Get(id)
		if added == nil {
			t.Fatal("Expected project to be retrievable")
		}

		if added.Name != project.Name {
			t.Errorf("Expected name %s, got %s", project.Name, added.Name)
		}

		if added.Path != project.Path {
			t.Errorf("Expected path %s, got %s", project.Path, added.Path)
		}

		if added.RemoteURL != project.RemoteURL {
			t.Errorf("Expected remote URL %s, got %s", project.RemoteURL, added.RemoteURL)
		}

		if added.AutoDiscovered != project.AutoDiscovered {
			t.Errorf("Expected auto_discovered %v, got %v", project.AutoDiscovered, added.AutoDiscovered)
		}

		// Verify AddedAt was set
		if added.AddedAt.IsZero() {
			t.Error("Expected AddedAt to be set")
		}

		// Verify Preferences was initialized
		if added.Preferences == nil {
			t.Error("Expected Preferences to be initialized")
		}
	})

	t.Run("prevents duplicate paths", func(t *testing.T) {
		tempDir := t.TempDir()
		registryPath := filepath.Join(tempDir, "projects.json")

		r, err := NewRegistry(registryPath)
		if err != nil {
			t.Fatalf("NewRegistry failed: %v", err)
		}

		project := Project{
			Name: "Test Project",
			Path: "/path/to/project",
		}

		_, err = r.Add(project)
		if err != nil {
			t.Fatalf("First Add failed: %v", err)
		}

		// Try to add a project with the same path
		duplicate := Project{
			Name: "Duplicate Project",
			Path: "/path/to/project",
		}

		_, err = r.Add(duplicate)
		if err == nil {
			t.Error("Expected error when adding duplicate path")
		}
	})

	t.Run("initializes nil preferences", func(t *testing.T) {
		tempDir := t.TempDir()
		registryPath := filepath.Join(tempDir, "projects.json")

		r, err := NewRegistry(registryPath)
		if err != nil {
			t.Fatalf("NewRegistry failed: %v", err)
		}

		project := Project{
			Name:        "Test Project",
			Path:        "/path/to/project",
			Preferences: nil,
		}

		id, err := r.Add(project)
		if err != nil {
			t.Fatalf("Add failed: %v", err)
		}

		added := r.Get(id)
		if added.Preferences == nil {
			t.Error("Expected Preferences to be initialized")
		}
	})
}

func TestRegistry_Persistence(t *testing.T) {
	t.Run("data survives new registry instance", func(t *testing.T) {
		tempDir := t.TempDir()
		registryPath := filepath.Join(tempDir, "projects.json")

		// Create first registry and add projects
		r1, err := NewRegistry(registryPath)
		if err != nil {
			t.Fatalf("NewRegistry failed: %v", err)
		}

		project1 := Project{
			Name:           "Project 1",
			Path:           "/path/to/project1",
			RemoteURL:      "https://github.com/user/repo1",
			LastScenarios:  5,
			AutoDiscovered: true,
		}

		project2 := Project{
			Name:           "Project 2",
			Path:           "/path/to/project2",
			LastScenarios:  10,
			AutoDiscovered: false,
		}

		id1, err := r1.Add(project1)
		if err != nil {
			t.Fatalf("Add project1 failed: %v", err)
		}

		id2, err := r1.Add(project2)
		if err != nil {
			t.Fatalf("Add project2 failed: %v", err)
		}

		// Update settings
		newSettings := Settings{
			AutoDiscover:         false,
			PollIntervalMs:       60000,
			ActivePollIntervalMs: 10000,
		}
		if err := r1.UpdateSettings(newSettings); err != nil {
			t.Fatalf("UpdateSettings failed: %v", err)
		}

		// Create second registry instance
		r2, err := NewRegistry(registryPath)
		if err != nil {
			t.Fatalf("Second NewRegistry failed: %v", err)
		}

		// Verify projects were loaded
		loaded1 := r2.Get(id1)
		if loaded1 == nil {
			t.Fatal("Expected project1 to be loaded")
		}

		if loaded1.Name != project1.Name {
			t.Errorf("Expected name %s, got %s", project1.Name, loaded1.Name)
		}

		if loaded1.Path != project1.Path {
			t.Errorf("Expected path %s, got %s", project1.Path, loaded1.Path)
		}

		if loaded1.RemoteURL != project1.RemoteURL {
			t.Errorf("Expected remote URL %s, got %s", project1.RemoteURL, loaded1.RemoteURL)
		}

		if loaded1.LastScenarios != project1.LastScenarios {
			t.Errorf("Expected last_scenarios %d, got %d", project1.LastScenarios, loaded1.LastScenarios)
		}

		if loaded1.AutoDiscovered != project1.AutoDiscovered {
			t.Errorf("Expected auto_discovered %v, got %v", project1.AutoDiscovered, loaded1.AutoDiscovered)
		}

		loaded2 := r2.Get(id2)
		if loaded2 == nil {
			t.Fatal("Expected project2 to be loaded")
		}

		if loaded2.Name != project2.Name {
			t.Errorf("Expected name %s, got %s", project2.Name, loaded2.Name)
		}

		// Verify settings were loaded
		loadedSettings := r2.Settings()
		if loadedSettings.AutoDiscover != newSettings.AutoDiscover {
			t.Errorf("Expected AutoDiscover %v, got %v", newSettings.AutoDiscover, loadedSettings.AutoDiscover)
		}

		if loadedSettings.PollIntervalMs != newSettings.PollIntervalMs {
			t.Errorf("Expected PollIntervalMs %d, got %d", newSettings.PollIntervalMs, loadedSettings.PollIntervalMs)
		}

		if loadedSettings.ActivePollIntervalMs != newSettings.ActivePollIntervalMs {
			t.Errorf("Expected ActivePollIntervalMs %d, got %d", newSettings.ActivePollIntervalMs, loadedSettings.ActivePollIntervalMs)
		}
	})

	t.Run("creates directory if needed", func(t *testing.T) {
		tempDir := t.TempDir()
		registryPath := filepath.Join(tempDir, "subdir", "nested", "projects.json")

		r, err := NewRegistry(registryPath)
		if err != nil {
			t.Fatalf("NewRegistry failed: %v", err)
		}

		project := Project{
			Name: "Test Project",
			Path: "/path/to/project",
		}

		_, err = r.Add(project)
		if err != nil {
			t.Fatalf("Add failed: %v", err)
		}

		// Verify file was created
		if _, err := os.Stat(registryPath); os.IsNotExist(err) {
			t.Error("Expected registry file to be created")
		}

		// Verify directory was created
		dir := filepath.Dir(registryPath)
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			t.Error("Expected directory to be created")
		}
	})
}

func TestRegistry_Remove(t *testing.T) {
	t.Run("removes project successfully", func(t *testing.T) {
		tempDir := t.TempDir()
		registryPath := filepath.Join(tempDir, "projects.json")

		r, err := NewRegistry(registryPath)
		if err != nil {
			t.Fatalf("NewRegistry failed: %v", err)
		}

		project := Project{
			Name: "Test Project",
			Path: "/path/to/project",
		}

		id, err := r.Add(project)
		if err != nil {
			t.Fatalf("Add failed: %v", err)
		}

		// Remove the project
		if err := r.Remove(id); err != nil {
			t.Fatalf("Remove failed: %v", err)
		}

		// Verify project was removed
		if removed := r.Get(id); removed != nil {
			t.Error("Expected project to be removed")
		}

		// Verify removal persisted
		r2, err := NewRegistry(registryPath)
		if err != nil {
			t.Fatalf("Second NewRegistry failed: %v", err)
		}

		if loaded := r2.Get(id); loaded != nil {
			t.Error("Expected removed project to not be loaded")
		}
	})

	t.Run("returns error for non-existent project", func(t *testing.T) {
		tempDir := t.TempDir()
		registryPath := filepath.Join(tempDir, "projects.json")

		r, err := NewRegistry(registryPath)
		if err != nil {
			t.Fatalf("NewRegistry failed: %v", err)
		}

		err = r.Remove("non-existent-id")
		if err == nil {
			t.Error("Expected error when removing non-existent project")
		}
	})
}

func TestRegistry_Update(t *testing.T) {
	t.Run("updates project successfully", func(t *testing.T) {
		tempDir := t.TempDir()
		registryPath := filepath.Join(tempDir, "projects.json")

		r, err := NewRegistry(registryPath)
		if err != nil {
			t.Fatalf("NewRegistry failed: %v", err)
		}

		project := Project{
			Name:          "Test Project",
			Path:          "/path/to/project",
			LastScenarios: 5,
		}

		id, err := r.Add(project)
		if err != nil {
			t.Fatalf("Add failed: %v", err)
		}

		// Get the added project to preserve generated fields
		added := r.Get(id)
		if added == nil {
			t.Fatal("Expected project to be retrievable")
		}

		// Update the project
		added.Name = "Updated Project"
		added.LastScenarios = 10
		added.LastOpened = time.Now()
		added.RemoteURL = "https://github.com/user/repo"

		if err := r.Update(*added); err != nil {
			t.Fatalf("Update failed: %v", err)
		}

		// Verify update
		updated := r.Get(id)
		if updated == nil {
			t.Fatal("Expected project to be retrievable after update")
		}

		if updated.Name != "Updated Project" {
			t.Errorf("Expected name 'Updated Project', got %s", updated.Name)
		}

		if updated.LastScenarios != 10 {
			t.Errorf("Expected last_scenarios 10, got %d", updated.LastScenarios)
		}

		if updated.RemoteURL != "https://github.com/user/repo" {
			t.Errorf("Expected remote URL to be updated, got %s", updated.RemoteURL)
		}

		// Verify update persisted
		r2, err := NewRegistry(registryPath)
		if err != nil {
			t.Fatalf("Second NewRegistry failed: %v", err)
		}

		loaded := r2.Get(id)
		if loaded == nil {
			t.Fatal("Expected updated project to be loaded")
		}

		if loaded.Name != "Updated Project" {
			t.Errorf("Expected loaded name 'Updated Project', got %s", loaded.Name)
		}
	})

	t.Run("returns error for non-existent project", func(t *testing.T) {
		tempDir := t.TempDir()
		registryPath := filepath.Join(tempDir, "projects.json")

		r, err := NewRegistry(registryPath)
		if err != nil {
			t.Fatalf("NewRegistry failed: %v", err)
		}

		project := Project{
			ID:   "non-existent-id",
			Name: "Test Project",
			Path: "/path/to/project",
		}

		err = r.Update(project)
		if err == nil {
			t.Error("Expected error when updating non-existent project")
		}
	})
}

func TestRegistry_Get(t *testing.T) {
	t.Run("retrieves project by ID", func(t *testing.T) {
		tempDir := t.TempDir()
		registryPath := filepath.Join(tempDir, "projects.json")

		r, err := NewRegistry(registryPath)
		if err != nil {
			t.Fatalf("NewRegistry failed: %v", err)
		}

		project := Project{
			Name: "Test Project",
			Path: "/path/to/project",
		}

		id, err := r.Add(project)
		if err != nil {
			t.Fatalf("Add failed: %v", err)
		}

		retrieved := r.Get(id)
		if retrieved == nil {
			t.Fatal("Expected project to be retrieved")
		}

		if retrieved.ID != id {
			t.Errorf("Expected ID %s, got %s", id, retrieved.ID)
		}

		if retrieved.Name != project.Name {
			t.Errorf("Expected name %s, got %s", project.Name, retrieved.Name)
		}
	})

	t.Run("returns nil for non-existent ID", func(t *testing.T) {
		tempDir := t.TempDir()
		registryPath := filepath.Join(tempDir, "projects.json")

		r, err := NewRegistry(registryPath)
		if err != nil {
			t.Fatalf("NewRegistry failed: %v", err)
		}

		retrieved := r.Get("non-existent-id")
		if retrieved != nil {
			t.Error("Expected nil for non-existent ID")
		}
	})

	t.Run("returns copy to prevent external modification", func(t *testing.T) {
		tempDir := t.TempDir()
		registryPath := filepath.Join(tempDir, "projects.json")

		r, err := NewRegistry(registryPath)
		if err != nil {
			t.Fatalf("NewRegistry failed: %v", err)
		}

		project := Project{
			Name: "Test Project",
			Path: "/path/to/project",
		}

		id, err := r.Add(project)
		if err != nil {
			t.Fatalf("Add failed: %v", err)
		}

		retrieved := r.Get(id)
		if retrieved == nil {
			t.Fatal("Expected project to be retrieved")
		}

		// Modify the retrieved project
		retrieved.Name = "Modified Name"

		// Verify original is unchanged
		original := r.Get(id)
		if original.Name != "Test Project" {
			t.Error("Expected original project to be unchanged")
		}
	})
}

func TestRegistry_List(t *testing.T) {
	t.Run("lists all projects", func(t *testing.T) {
		tempDir := t.TempDir()
		registryPath := filepath.Join(tempDir, "projects.json")

		r, err := NewRegistry(registryPath)
		if err != nil {
			t.Fatalf("NewRegistry failed: %v", err)
		}

		// Add multiple projects
		for i := 1; i <= 3; i++ {
			project := Project{
				Name: "Project " + string(rune('0'+i)),
				Path: "/path/to/project" + string(rune('0'+i)),
			}

			if _, err := r.Add(project); err != nil {
				t.Fatalf("Add project %d failed: %v", i, err)
			}
		}

		projects := r.List()
		if len(projects) != 3 {
			t.Errorf("Expected 3 projects, got %d", len(projects))
		}
	})

	t.Run("returns empty list when no projects", func(t *testing.T) {
		tempDir := t.TempDir()
		registryPath := filepath.Join(tempDir, "projects.json")

		r, err := NewRegistry(registryPath)
		if err != nil {
			t.Fatalf("NewRegistry failed: %v", err)
		}

		projects := r.List()
		if len(projects) != 0 {
			t.Errorf("Expected empty list, got %d projects", len(projects))
		}
	})
}

func TestRegistry_Settings(t *testing.T) {
	t.Run("returns default settings for new registry", func(t *testing.T) {
		tempDir := t.TempDir()
		registryPath := filepath.Join(tempDir, "projects.json")

		r, err := NewRegistry(registryPath)
		if err != nil {
			t.Fatalf("NewRegistry failed: %v", err)
		}

		settings := r.Settings()

		if !settings.AutoDiscover {
			t.Error("Expected default AutoDiscover to be true")
		}

		if settings.PollIntervalMs != 30000 {
			t.Errorf("Expected default PollIntervalMs 30000, got %d", settings.PollIntervalMs)
		}

		if settings.ActivePollIntervalMs != 5000 {
			t.Errorf("Expected default ActivePollIntervalMs 5000, got %d", settings.ActivePollIntervalMs)
		}
	})

	t.Run("updates and retrieves settings", func(t *testing.T) {
		tempDir := t.TempDir()
		registryPath := filepath.Join(tempDir, "projects.json")

		r, err := NewRegistry(registryPath)
		if err != nil {
			t.Fatalf("NewRegistry failed: %v", err)
		}

		newSettings := Settings{
			AutoDiscover:         false,
			PollIntervalMs:       60000,
			ActivePollIntervalMs: 10000,
		}

		if err := r.UpdateSettings(newSettings); err != nil {
			t.Fatalf("UpdateSettings failed: %v", err)
		}

		settings := r.Settings()

		if settings.AutoDiscover {
			t.Error("Expected AutoDiscover to be false")
		}

		if settings.PollIntervalMs != 60000 {
			t.Errorf("Expected PollIntervalMs 60000, got %d", settings.PollIntervalMs)
		}

		if settings.ActivePollIntervalMs != 10000 {
			t.Errorf("Expected ActivePollIntervalMs 10000, got %d", settings.ActivePollIntervalMs)
		}
	})
}

func TestRegistry_DefaultSettings(t *testing.T) {
	settings := defaultSettings()

	if !settings.AutoDiscover {
		t.Error("Expected default AutoDiscover to be true")
	}

	if settings.PollIntervalMs != 30000 {
		t.Errorf("Expected default PollIntervalMs 30000, got %d", settings.PollIntervalMs)
	}

	if settings.ActivePollIntervalMs != 5000 {
		t.Errorf("Expected default ActivePollIntervalMs 5000, got %d", settings.ActivePollIntervalMs)
	}
}
