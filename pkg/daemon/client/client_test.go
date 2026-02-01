package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	t.Run("creates client with default timeout", func(t *testing.T) {
		c := New("http://localhost:8080")
		if c.baseURL != "http://localhost:8080" {
			t.Errorf("expected baseURL 'http://localhost:8080', got %s", c.baseURL)
		}
		if c.httpClient.Timeout != 30*time.Second {
			t.Errorf("expected timeout 30s, got %v", c.httpClient.Timeout)
		}
	})

	t.Run("creates client with custom timeout", func(t *testing.T) {
		c := New("http://localhost:8080", WithTimeout(10*time.Second))
		if c.httpClient.Timeout != 10*time.Second {
			t.Errorf("expected timeout 10s, got %v", c.httpClient.Timeout)
		}
	})

	t.Run("creates client with API key", func(t *testing.T) {
		c := New("http://localhost:8080", WithAPIKey("test-key"))
		if c.apiKey != "test-key" {
			t.Errorf("expected API key 'test-key', got %s", c.apiKey)
		}
	})
}

func TestHealth(t *testing.T) {
	t.Run("returns health status on success", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/api/v1/health" {
				t.Errorf("unexpected path: %s", r.URL.Path)
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(HealthResponse{
				Status:    "healthy",
				Timestamp: "2024-01-01T00:00:00Z",
				Version:   "1.0.0",
			})
		}))
		defer server.Close()

		c := New(server.URL)
		health, err := c.Health(context.Background())
		if err != nil {
			t.Fatalf("Health failed: %v", err)
		}
		if health.Status != "healthy" {
			t.Errorf("expected status 'healthy', got %s", health.Status)
		}
		if health.Version != "1.0.0" {
			t.Errorf("expected version '1.0.0', got %s", health.Version)
		}
	})

	t.Run("returns error on server error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "internal error"})
		}))
		defer server.Close()

		c := New(server.URL)
		_, err := c.Health(context.Background())
		if err == nil {
			t.Error("expected error on server error")
		}
	})
}

func TestIsHealthy(t *testing.T) {
	t.Run("returns true when healthy", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(HealthResponse{Status: "healthy"})
		}))
		defer server.Close()

		c := New(server.URL)
		if !c.IsHealthy(context.Background()) {
			t.Error("expected IsHealthy to return true")
		}
	})

	t.Run("returns false when unhealthy", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(HealthResponse{Status: "unhealthy"})
		}))
		defer server.Close()

		c := New(server.URL)
		if c.IsHealthy(context.Background()) {
			t.Error("expected IsHealthy to return false")
		}
	})

	t.Run("returns false on connection error", func(t *testing.T) {
		c := New("http://localhost:99999")
		if c.IsHealthy(context.Background()) {
			t.Error("expected IsHealthy to return false on connection error")
		}
	})
}

func TestRunScenario(t *testing.T) {
	t.Run("creates run on success", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != "POST" {
				t.Errorf("expected POST, got %s", r.Method)
			}
			if r.URL.Path != "/api/v1/runs" {
				t.Errorf("unexpected path: %s", r.URL.Path)
			}

			var req RunRequest
			_ = json.NewDecoder(r.Body).Decode(&req)
			if req.ScenarioName != "test-scenario" {
				t.Errorf("expected scenario_name 'test-scenario', got %s", req.ScenarioName)
			}

			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(RunResponse{
				ID:         "run-123",
				Status:     "running",
				ScenarioID: "test-scenario",
				StartTime:  time.Now(),
			})
		}))
		defer server.Close()

		c := New(server.URL)
		run, err := c.RunScenario(context.Background(), &RunRequest{ScenarioName: "test-scenario"})
		if err != nil {
			t.Fatalf("RunScenario failed: %v", err)
		}
		if run.ID != "run-123" {
			t.Errorf("expected ID 'run-123', got %s", run.ID)
		}
		if run.Status != "running" {
			t.Errorf("expected status 'running', got %s", run.Status)
		}
	})
}

func TestRunBatch(t *testing.T) {
	t.Run("creates batch run on success", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != "POST" {
				t.Errorf("expected POST, got %s", r.Method)
			}
			if r.URL.Path != "/api/v1/runs/batch" {
				t.Errorf("unexpected path: %s", r.URL.Path)
			}

			var req RunRequest
			_ = json.NewDecoder(r.Body).Decode(&req)
			if req.Suite != "smoke-suite" {
				t.Errorf("expected suite 'smoke-suite', got %s", req.Suite)
			}

			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(RunResponse{
				ID:        "batch-123",
				Status:    "running",
				Scenarios: []string{"s1", "s2", "s3"},
				StartTime: time.Now(),
			})
		}))
		defer server.Close()

		c := New(server.URL)
		run, err := c.RunBatch(context.Background(), &RunRequest{Suite: "smoke-suite"})
		if err != nil {
			t.Fatalf("RunBatch failed: %v", err)
		}
		if run.ID != "batch-123" {
			t.Errorf("expected ID 'batch-123', got %s", run.ID)
		}
		if len(run.Scenarios) != 3 {
			t.Errorf("expected 3 scenarios, got %d", len(run.Scenarios))
		}
	})

	t.Run("creates batch run by scenarios", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var req RunRequest
			_ = json.NewDecoder(r.Body).Decode(&req)
			if len(req.Scenarios) != 2 {
				t.Errorf("expected 2 scenarios, got %d", len(req.Scenarios))
			}

			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(RunResponse{
				ID:        "batch-456",
				Status:    "running",
				Scenarios: req.Scenarios,
			})
		}))
		defer server.Close()

		c := New(server.URL)
		run, err := c.RunBatch(context.Background(), &RunRequest{
			Scenarios: []string{"s1", "s2"},
		})
		if err != nil {
			t.Fatalf("RunBatch failed: %v", err)
		}
		if len(run.Scenarios) != 2 {
			t.Errorf("expected 2 scenarios, got %d", len(run.Scenarios))
		}
	})

	t.Run("creates batch run by tags", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var req RunRequest
			_ = json.NewDecoder(r.Body).Decode(&req)
			if len(req.Tags) != 1 || req.Tags[0] != "smoke" {
				t.Errorf("expected tags [smoke], got %v", req.Tags)
			}

			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(RunResponse{
				ID:        "batch-789",
				Status:    "running",
				Scenarios: []string{"s1", "s2"},
			})
		}))
		defer server.Close()

		c := New(server.URL)
		run, err := c.RunBatch(context.Background(), &RunRequest{
			Tags: []string{"smoke"},
		})
		if err != nil {
			t.Fatalf("RunBatch failed: %v", err)
		}
		if run.ID != "batch-789" {
			t.Errorf("expected ID 'batch-789', got %s", run.ID)
		}
	})
}

func TestGetRun(t *testing.T) {
	t.Run("gets run status", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/api/v1/runs/run-123" {
				t.Errorf("unexpected path: %s", r.URL.Path)
			}

			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(RunResponse{
				ID:       "run-123",
				Status:   "completed",
				Duration: "5.2s",
			})
		}))
		defer server.Close()

		c := New(server.URL)
		run, err := c.GetRun(context.Background(), "run-123")
		if err != nil {
			t.Fatalf("GetRun failed: %v", err)
		}
		if run.Status != "completed" {
			t.Errorf("expected status 'completed', got %s", run.Status)
		}
	})

	t.Run("returns error for non-existent run", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "run not found"})
		}))
		defer server.Close()

		c := New(server.URL)
		_, err := c.GetRun(context.Background(), "non-existent")
		if err == nil {
			t.Error("expected error for non-existent run")
		}
	})
}

func TestCancelRun(t *testing.T) {
	t.Run("cancels running run", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != "DELETE" {
				t.Errorf("expected DELETE, got %s", r.Method)
			}
			if r.URL.Path != "/api/v1/runs/run-123" {
				t.Errorf("unexpected path: %s", r.URL.Path)
			}

			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]string{"status": "canceled"})
		}))
		defer server.Close()

		c := New(server.URL)
		err := c.CancelRun(context.Background(), "run-123")
		if err != nil {
			t.Fatalf("CancelRun failed: %v", err)
		}
	})
}

func TestWaitForRun(t *testing.T) {
	t.Run("waits for completion", func(t *testing.T) {
		callCount := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callCount++
			status := "running"
			if callCount >= 3 {
				status = "completed"
			}

			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(RunResponse{
				ID:     "run-123",
				Status: status,
			})
		}))
		defer server.Close()

		c := New(server.URL)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		run, err := c.WaitForRun(ctx, "run-123", 10*time.Millisecond)
		if err != nil {
			t.Fatalf("WaitForRun failed: %v", err)
		}
		if run.Status != "completed" {
			t.Errorf("expected status 'completed', got %s", run.Status)
		}
		if callCount < 3 {
			t.Errorf("expected at least 3 calls, got %d", callCount)
		}
	})

	t.Run("returns on failure", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(RunResponse{
				ID:     "run-123",
				Status: "failed",
				Error:  "test failure",
			})
		}))
		defer server.Close()

		c := New(server.URL)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		run, err := c.WaitForRun(ctx, "run-123", 10*time.Millisecond)
		if err != nil {
			t.Fatalf("WaitForRun failed: %v", err)
		}
		if run.Status != "failed" {
			t.Errorf("expected status 'failed', got %s", run.Status)
		}
	})

	t.Run("respects context cancellation", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(RunResponse{
				ID:     "run-123",
				Status: "running",
			})
		}))
		defer server.Close()

		c := New(server.URL)
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		_, err := c.WaitForRun(ctx, "run-123", 10*time.Millisecond)
		if err == nil {
			t.Error("expected error on context cancellation")
		}
	})
}

func TestListScenarios(t *testing.T) {
	t.Run("lists scenarios", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/api/v1/scenarios" {
				t.Errorf("unexpected path: %s", r.URL.Path)
			}

			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"scenarios": []ScenarioResponse{
					{Name: "s1", Description: "Test 1", FlowCount: 3},
					{Name: "s2", Description: "Test 2", FlowCount: 5},
				},
			})
		}))
		defer server.Close()

		c := New(server.URL)
		scenarios, err := c.ListScenarios(context.Background())
		if err != nil {
			t.Fatalf("ListScenarios failed: %v", err)
		}
		if len(scenarios) != 2 {
			t.Errorf("expected 2 scenarios, got %d", len(scenarios))
		}
	})
}

func TestListSuites(t *testing.T) {
	t.Run("lists suites", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/api/v1/suites" {
				t.Errorf("unexpected path: %s", r.URL.Path)
			}

			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"suites": []SuiteResponse{
					{Name: "smoke", Description: "Smoke tests", Scenarios: []string{"s1", "s2"}},
					{Name: "full", Description: "Full suite"},
				},
			})
		}))
		defer server.Close()

		c := New(server.URL)
		suites, err := c.ListSuites(context.Background())
		if err != nil {
			t.Fatalf("ListSuites failed: %v", err)
		}
		if len(suites) != 2 {
			t.Errorf("expected 2 suites, got %d", len(suites))
		}
		if suites[0].Name != "smoke" {
			t.Errorf("expected first suite 'smoke', got %s", suites[0].Name)
		}
	})
}

func TestGetSuite(t *testing.T) {
	t.Run("gets suite details", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/api/v1/suites/smoke" {
				t.Errorf("unexpected path: %s", r.URL.Path)
			}

			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(SuiteResponse{
				Name:              "smoke",
				Description:       "Smoke tests",
				Tags:              []string{"smoke"},
				Parallel:          2,
				FailFast:          true,
				ResolvedScenarios: []string{"s1", "s2", "s3"},
			})
		}))
		defer server.Close()

		c := New(server.URL)
		suite, err := c.GetSuite(context.Background(), "smoke")
		if err != nil {
			t.Fatalf("GetSuite failed: %v", err)
		}
		if suite.Name != "smoke" {
			t.Errorf("expected name 'smoke', got %s", suite.Name)
		}
		if suite.Parallel != 2 {
			t.Errorf("expected parallel 2, got %d", suite.Parallel)
		}
		if len(suite.ResolvedScenarios) != 3 {
			t.Errorf("expected 3 resolved scenarios, got %d", len(suite.ResolvedScenarios))
		}
	})

	t.Run("returns error for non-existent suite", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "suite not found"})
		}))
		defer server.Close()

		c := New(server.URL)
		_, err := c.GetSuite(context.Background(), "non-existent")
		if err == nil {
			t.Error("expected error for non-existent suite")
		}
	})
}

func TestAPIKeyAuth(t *testing.T) {
	t.Run("includes API key in header", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			auth := r.Header.Get("Authorization")
			if auth != "Bearer test-api-key" {
				t.Errorf("expected 'Bearer test-api-key', got '%s'", auth)
			}

			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(HealthResponse{Status: "healthy"})
		}))
		defer server.Close()

		c := New(server.URL, WithAPIKey("test-api-key"))
		_, err := c.Health(context.Background())
		if err != nil {
			t.Fatalf("Health failed: %v", err)
		}
	})
}
