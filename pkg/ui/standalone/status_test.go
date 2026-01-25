package standalone

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestHealthChecker_Check_Running(t *testing.T) {
	// Create a mock server that returns healthy status
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/health" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		resp := HealthResponse{
			Status:    "healthy",
			Version:   "1.0.0",
			Uptime:    123.45,
			Scenarios: 5,
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			t.Fatalf("failed to encode response: %v", err)
		}
	}))
	defer server.Close()

	checker := NewHealthChecker()
	project := &Project{
		ID:        "test-project",
		Name:      "Test Project",
		RemoteURL: server.URL,
	}

	ctx := context.Background()
	status := checker.Check(ctx, project)

	if status.State != StateRunning {
		t.Errorf("expected state %s, got %s", StateRunning, status.State)
	}
	if status.Version != "1.0.0" {
		t.Errorf("expected version 1.0.0, got %s", status.Version)
	}
	if status.Uptime != 123.45 {
		t.Errorf("expected uptime 123.45, got %f", status.Uptime)
	}
	if status.Scenarios != 5 {
		t.Errorf("expected 5 scenarios, got %d", status.Scenarios)
	}
	if status.Error != "" {
		t.Errorf("expected no error, got %s", status.Error)
	}
	if status.LastChecked.IsZero() {
		t.Error("expected LastChecked to be set")
	}
}

func TestHealthChecker_Check_Stopped(t *testing.T) {
	// Use a custom checker to avoid interference from local servers
	customChecker := &HealthChecker{
		client: &http.Client{
			Timeout: 1 * time.Second,
		},
		cache: make(map[string]*DaemonStatus),
	}

	// Create a project that points to an impossible URL
	project := &Project{
		ID:        "test-project",
		Name:      "Test Project",
		RemoteURL: "http://127.0.0.1:65535/health", // Port out of normal range
	}

	ctx := context.Background()
	status := customChecker.Check(ctx, project)

	// Should be stopped since nothing is running on that port
	if status.State != StateStopped {
		t.Errorf("expected state %s, got %s (port: %d, error: %s)", StateStopped, status.State, status.Port, status.Error)
	}
}

func TestHealthChecker_Check_Unhealthy(t *testing.T) {
	tests := []struct {
		name           string
		statusCode     int
		responseBody   interface{}
		expectedState  DaemonState
		expectedError  string
	}{
		{
			name:       "unhealthy status in response",
			statusCode: http.StatusOK,
			responseBody: HealthResponse{
				Status:  "unhealthy",
				Version: "1.0.0",
			},
			expectedState: StateUnhealthy,
			expectedError: "",
		},
		{
			name:          "server error",
			statusCode:    http.StatusInternalServerError,
			responseBody:  map[string]string{"error": "internal error"},
			expectedState: StateUnhealthy,
			expectedError: "HTTP 500",
		},
		{
			name:          "service unavailable",
			statusCode:    http.StatusServiceUnavailable,
			responseBody:  nil,
			expectedState: StateUnhealthy,
			expectedError: "HTTP 503",
		},
		{
			name:          "invalid JSON response",
			statusCode:    http.StatusOK,
			responseBody:  "invalid json",
			expectedState: StateUnhealthy,
			expectedError: "invalid health response",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				if tt.responseBody != nil {
					if str, ok := tt.responseBody.(string); ok {
						_, _ = w.Write([]byte(str))
					} else {
						_ = json.NewEncoder(w).Encode(tt.responseBody)
					}
				}
			}))
			defer server.Close()

			checker := NewHealthChecker()
			project := &Project{
				ID:        "test-project",
				Name:      "Test Project",
				RemoteURL: server.URL,
			}

			ctx := context.Background()
			status := checker.Check(ctx, project)

			if status.State != tt.expectedState {
				t.Errorf("expected state %s, got %s", tt.expectedState, status.State)
			}
			if tt.expectedError != "" {
				if status.Error == "" {
					t.Errorf("expected error containing %q, got no error", tt.expectedError)
				} else if len(tt.expectedError) > 0 && len(status.Error) > 0 {
					// Just check that error contains expected substring
					found := false
					for i := 0; i <= len(status.Error)-len(tt.expectedError); i++ {
						if status.Error[i:i+len(tt.expectedError)] == tt.expectedError {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("expected error containing %q, got %q", tt.expectedError, status.Error)
					}
				}
			}
		})
	}
}

func TestHealthChecker_Check_Timeout(t *testing.T) {
	// Create a checker with a very short timeout
	checker := &HealthChecker{
		client: &http.Client{
			Timeout: 100 * time.Millisecond,
		},
		cache: make(map[string]*DaemonStatus),
	}

	// Create a server that takes longer than the timeout
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(1 * time.Second) // Longer than 100ms timeout
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	project := &Project{
		ID:        "test-project",
		Name:      "Test Project",
		RemoteURL: server.URL,
	}

	ctx := context.Background()
	status := checker.Check(ctx, project)

	// Should timeout and return stopped
	if status.State != StateStopped {
		t.Errorf("expected state %s after timeout, got %s (error: %s)", StateStopped, status.State, status.Error)
	}
}

func TestHealthChecker_Check_LocalPorts(t *testing.T) {
	// Create a custom checker that avoids real local ports
	checker := &HealthChecker{
		client: &http.Client{
			Timeout: 500 * time.Millisecond,
		},
		cache: make(map[string]*DaemonStatus),
	}

	// Use a project with impossible RemoteURL so it falls back to port scanning
	// But with our short timeout, it should quickly determine nothing is running
	project := &Project{
		ID:        "test-project",
		Name:      "Test Project",
		RemoteURL: "http://127.0.0.1:65000/health", // Very high port unlikely to be used
	}

	ctx := context.Background()
	status := checker.Check(ctx, project)

	// Without a valid daemon, should return stopped
	if status.State != StateStopped {
		t.Errorf("expected state %s when no daemon found, got %s (port: %d)", StateStopped, status.State, status.Port)
	}
}

func TestHealthChecker_StatusCache(t *testing.T) {
	checker := NewHealthChecker()

	// Initially, no status should be cached
	status := checker.GetStatus("test-project")
	if status != nil {
		t.Error("expected no cached status initially")
	}

	// Set a status
	expectedStatus := &DaemonStatus{
		State:       StateRunning,
		Port:        8080,
		Version:     "1.0.0",
		LastChecked: time.Now(),
	}
	checker.SetStatus("test-project", expectedStatus)

	// Retrieve the status
	status = checker.GetStatus("test-project")
	if status == nil {
		t.Fatal("expected cached status to be returned")
	}

	if status.State != expectedStatus.State {
		t.Errorf("expected state %s, got %s", expectedStatus.State, status.State)
	}
	if status.Port != expectedStatus.Port {
		t.Errorf("expected port %d, got %d", expectedStatus.Port, status.Port)
	}
	if status.Version != expectedStatus.Version {
		t.Errorf("expected version %s, got %s", expectedStatus.Version, status.Version)
	}

	// Verify that modifying the returned status doesn't affect the cache
	status.State = StateStopped
	cachedStatus := checker.GetStatus("test-project")
	if cachedStatus.State != StateRunning {
		t.Error("cache was modified by external change")
	}

	// Clear the status
	checker.ClearStatus("test-project")
	status = checker.GetStatus("test-project")
	if status != nil {
		t.Error("expected status to be cleared")
	}
}

func TestHealthChecker_ConcurrentAccess(t *testing.T) {
	checker := NewHealthChecker()

	// Test concurrent reads and writes
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(id int) {
			projectID := "project-1"
			status := &DaemonStatus{
				State:       StateRunning,
				Port:        8080 + id,
				LastChecked: time.Now(),
			}
			checker.SetStatus(projectID, status)
			_ = checker.GetStatus(projectID)
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify we can still read
	status := checker.GetStatus("project-1")
	if status == nil {
		t.Fatal("expected status to be set")
	}
	if status.State != StateRunning {
		t.Errorf("expected state %s, got %s", StateRunning, status.State)
	}
}

func TestHealthChecker_ContextCancellation(t *testing.T) {
	// Create a server that takes a long time to respond
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	checker := NewHealthChecker()
	project := &Project{
		ID:        "test-project",
		Name:      "Test Project",
		RemoteURL: server.URL,
	}

	// Create a context that cancels immediately
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	status := checker.Check(ctx, project)

	// Should return stopped since context was cancelled
	if status.State != StateStopped {
		t.Errorf("expected state %s after context cancellation, got %s", StateStopped, status.State)
	}
}

func TestDaemonState_String(t *testing.T) {
	tests := []struct {
		state    DaemonState
		expected string
	}{
		{StateUnknown, "unknown"},
		{StateStopped, "stopped"},
		{StateStarting, "starting"},
		{StateRunning, "running"},
		{StateUnhealthy, "unhealthy"},
	}

	for _, tt := range tests {
		t.Run(string(tt.state), func(t *testing.T) {
			if string(tt.state) != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, string(tt.state))
			}
		})
	}
}

func TestHealthResponse_JSON(t *testing.T) {
	resp := HealthResponse{
		Status:    "healthy",
		Version:   "1.0.0",
		Uptime:    123.45,
		Scenarios: 10,
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var decoded HealthResponse
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if decoded.Status != resp.Status {
		t.Errorf("expected status %s, got %s", resp.Status, decoded.Status)
	}
	if decoded.Version != resp.Version {
		t.Errorf("expected version %s, got %s", resp.Version, decoded.Version)
	}
	if decoded.Uptime != resp.Uptime {
		t.Errorf("expected uptime %f, got %f", resp.Uptime, decoded.Uptime)
	}
	if decoded.Scenarios != resp.Scenarios {
		t.Errorf("expected scenarios %d, got %d", resp.Scenarios, decoded.Scenarios)
	}
}

func TestDaemonStatus_JSON(t *testing.T) {
	now := time.Now()
	status := DaemonStatus{
		State:       StateRunning,
		Port:        8080,
		Version:     "1.0.0",
		LastChecked: now,
		Uptime:      123.45,
		Scenarios:   10,
	}

	data, err := json.Marshal(status)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var decoded DaemonStatus
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if decoded.State != status.State {
		t.Errorf("expected state %s, got %s", status.State, decoded.State)
	}
	if decoded.Port != status.Port {
		t.Errorf("expected port %d, got %d", status.Port, decoded.Port)
	}
	if decoded.Version != status.Version {
		t.Errorf("expected version %s, got %s", status.Version, decoded.Version)
	}
	if decoded.Uptime != status.Uptime {
		t.Errorf("expected uptime %f, got %f", status.Uptime, decoded.Uptime)
	}
	if decoded.Scenarios != status.Scenarios {
		t.Errorf("expected scenarios %d, got %d", status.Scenarios, decoded.Scenarios)
	}
}
