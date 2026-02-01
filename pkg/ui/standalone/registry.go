// Package standalone provides the project registry and management for Chronicle's standalone UI mode.
package standalone

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Project represents a Chronicle project tracked by the standalone UI.
type Project struct {
	// ID is the unique identifier for this project.
	ID string `json:"id"`
	// Name is the display name for this project.
	Name string `json:"name"`
	// Path is the absolute filesystem path to the project directory.
	Path string `json:"path"`
	// RemoteURL is the git remote URL if available.
	RemoteURL string `json:"remote_url,omitempty"`
	// AddedAt is when this project was first added to the registry.
	AddedAt time.Time `json:"added_at"`
	// LastOpened is when this project was last accessed.
	LastOpened time.Time `json:"last_opened,omitempty"`
	// LastScenarios is the count of scenarios from the last check.
	LastScenarios int `json:"last_scenarios"`
	// Preferences stores project-specific settings.
	Preferences map[string]interface{} `json:"preferences,omitempty"`
	// AutoDiscovered indicates if this project was found via auto-discovery.
	AutoDiscovered bool `json:"auto_discovered"`
}

// Settings contains global configuration for the standalone UI.
type Settings struct {
	// AutoDiscover enables automatic discovery of Chronicle projects.
	AutoDiscover bool `json:"auto_discover"`
	// PollIntervalMs is the polling interval in milliseconds for inactive projects.
	PollIntervalMs int `json:"poll_interval_ms"`
	// ActivePollIntervalMs is the polling interval for the currently active project.
	ActivePollIntervalMs int `json:"active_poll_interval_ms"`
}

// defaultSettings returns the default settings for the standalone UI.
func defaultSettings() Settings {
	return Settings{
		AutoDiscover:         true,
		PollIntervalMs:       30000,
		ActivePollIntervalMs: 5000,
	}
}

// registryData is the persisted structure stored in projects.json.
type registryData struct {
	Projects []Project `json:"projects"`
	Settings Settings  `json:"settings"`
}

// Registry manages Chronicle projects and persists them to disk.
type Registry struct {
	mu       sync.RWMutex
	path     string
	projects map[string]*Project
	settings Settings
}

// NewRegistry creates a new project registry that persists to the given path.
// If the path exists, it loads existing data. If not, it initializes with defaults.
func NewRegistry(path string) (*Registry, error) {
	r := &Registry{
		path:     path,
		projects: make(map[string]*Project),
		settings: defaultSettings(),
	}

	// Try to load existing data
	if err := r.load(); err != nil {
		// If the file doesn't exist, that's fine - we'll create it on first save
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("failed to load registry: %w", err)
		}
	}

	return r, nil
}

// Add adds a new project to the registry. It generates a unique ID and sets AddedAt.
// Returns an error if a project with the same path already exists.
func (r *Registry) Add(project Project) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Check if a project with this path already exists
	for _, p := range r.projects {
		if p.Path == project.Path {
			return "", fmt.Errorf("project with path %s already exists", project.Path)
		}
	}

	// Generate ID and set timestamp
	project.ID = uuid.New().String()
	project.AddedAt = time.Now()

	// Initialize preferences if nil
	if project.Preferences == nil {
		project.Preferences = make(map[string]interface{})
	}

	r.projects[project.ID] = &project

	if err := r.save(); err != nil {
		return "", fmt.Errorf("failed to save registry: %w", err)
	}

	return project.ID, nil
}

// List returns all projects in the registry.
func (r *Registry) List() []Project {
	r.mu.RLock()
	defer r.mu.RUnlock()

	projects := make([]Project, 0, len(r.projects))
	for _, p := range r.projects {
		projects = append(projects, *p)
	}

	return projects
}

// Get retrieves a project by its ID. Returns nil if not found.
func (r *Registry) Get(id string) *Project {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if p, ok := r.projects[id]; ok {
		// Return a copy to prevent external modification
		project := *p
		return &project
	}

	return nil
}

// Remove deletes a project from the registry by its ID.
// Returns an error if the project doesn't exist.
func (r *Registry) Remove(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.projects[id]; !ok {
		return fmt.Errorf("project with id %s not found", id)
	}

	delete(r.projects, id)

	if err := r.save(); err != nil {
		return fmt.Errorf("failed to save registry: %w", err)
	}

	return nil
}

// Update updates an existing project in the registry.
// Returns an error if the project doesn't exist.
func (r *Registry) Update(project Project) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.projects[project.ID]; !ok {
		return fmt.Errorf("project with id %s not found", project.ID)
	}

	r.projects[project.ID] = &project

	if err := r.save(); err != nil {
		return fmt.Errorf("failed to save registry: %w", err)
	}

	return nil
}

// Settings returns a copy of the current settings.
func (r *Registry) Settings() Settings {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.settings
}

// UpdateSettings updates the registry settings and persists them.
func (r *Registry) UpdateSettings(settings Settings) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.settings = settings

	if err := r.save(); err != nil {
		return fmt.Errorf("failed to save registry: %w", err)
	}

	return nil
}

// load reads the registry data from disk.
func (r *Registry) load() error {
	data, err := os.ReadFile(r.path)
	if err != nil {
		return err
	}

	var rd registryData
	if err := json.Unmarshal(data, &rd); err != nil {
		return fmt.Errorf("failed to unmarshal registry: %w", err)
	}

	// Rebuild the projects map
	r.projects = make(map[string]*Project)
	for i := range rd.Projects {
		r.projects[rd.Projects[i].ID] = &rd.Projects[i]
	}

	r.settings = rd.Settings

	return nil
}

// save writes the registry data to disk, creating the directory if needed.
func (r *Registry) save() error {
	// Create directory if it doesn't exist
	dir := filepath.Dir(r.path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Convert map to slice for serialization
	projects := make([]Project, 0, len(r.projects))
	for _, p := range r.projects {
		projects = append(projects, *p)
	}

	rd := registryData{
		Projects: projects,
		Settings: r.settings,
	}

	data, err := json.MarshalIndent(rd, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal registry: %w", err)
	}

	if err := os.WriteFile(r.path, data, 0644); err != nil {
		return fmt.Errorf("failed to write registry: %w", err)
	}

	return nil
}
