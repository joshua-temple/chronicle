package daemon

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/joshua-temple/chronicle/pkg/config"
	"github.com/joshua-temple/chronicle/pkg/core"
	"github.com/joshua-temple/chronicle/pkg/discovery"
	"github.com/joshua-temple/chronicle/pkg/execution"
	"github.com/joshua-temple/chronicle/pkg/results"
	"github.com/joshua-temple/chronicle/pkg/scenario"
)

// mockStorage implements a simple in-memory storage for testing
type mockStorage struct {
	results map[string]*results.RunResult
}

func newMockStorage() *mockStorage {
	return &mockStorage{results: make(map[string]*results.RunResult)}
}

func (m *mockStorage) Save(ctx context.Context, result *results.RunResult) error {
	m.results[result.ID] = result
	return nil
}

func (m *mockStorage) Load(ctx context.Context, id string) (*results.RunResult, error) {
	if r, ok := m.results[id]; ok {
		return r, nil
	}
	return nil, fmt.Errorf("result not found: %s", id)
}

func (m *mockStorage) Delete(ctx context.Context, id string) error {
	delete(m.results, id)
	return nil
}

func (m *mockStorage) List(ctx context.Context, opts ...results.ListOption) ([]string, error) {
	var ids []string
	for id := range m.results {
		ids = append(ids, id)
	}
	return ids, nil
}

// createTestServer creates a Server for testing with mock dependencies
func createTestServer(t *testing.T) *Server {
	t.Helper()

	registry := &discovery.Registry{
		Components: map[core.ComponentID]*core.Component{
			"TestComponent": {
				Name:       "TestComponent",
				Type:       core.ComponentSetup,
				SourceFile: "test.go",
				Tags:       []string{"test"},
			},
		},
	}

	cfg := &config.Config{
		Name:    "test-project",
		Version: "1.0.0",
		Scenarios: []config.ScenarioConfig{
			{
				Name:        "test-scenario",
				Description: "A test scenario",
				Tags:        []string{"test"},
				Timeout:     5 * time.Minute,
			},
		},
		Discovery: config.DiscoveryConfig{
			Paths: []string{"./"},
		},
		Execution: config.ExecutionConfig{
			DefaultTimeout: 5 * time.Minute,
			Parallelism:    1,
		},
	}

	resolver := scenario.NewResolver(cfg, registry)
	executor := execution.NewExecutor()
	storage := newMockStorage()
	eventBus := NewEmbeddedEventBus()

	s := &Server{
		config:     cfg,
		configPath: "/test/chronicle.yaml",
		registry:   registry,
		resolver:   resolver,
		executor:   executor,
		storage:    storage,
		eventBus:   eventBus,
		activeRuns: make(map[string]*RunInfo),
	}

	return s
}

func TestServer_handleHealth(t *testing.T) {
	s := createTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	rr := httptest.NewRecorder()

	s.handleHealth(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("handleHealth() status = %d, expected %d", rr.Code, http.StatusOK)
	}

	var response map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if response["status"] != "healthy" {
		t.Errorf("handleHealth() status = %q, expected 'healthy'", response["status"])
	}

	if response["version"] != "1.0.0" {
		t.Errorf("handleHealth() version = %q, expected '1.0.0'", response["version"])
	}
}

func TestServer_handleListScenarios(t *testing.T) {
	s := createTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/scenarios", nil)
	rr := httptest.NewRecorder()

	s.handleListScenarios(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("handleListScenarios() status = %d, expected %d", rr.Code, http.StatusOK)
	}

	var response map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	scenarios, ok := response["scenarios"].([]any)
	if !ok {
		t.Fatal("Response missing 'scenarios' array")
	}

	if len(scenarios) == 0 {
		t.Error("Expected at least one scenario")
	}

	count, ok := response["count"].(float64)
	if !ok || int(count) != len(scenarios) {
		t.Errorf("Response count = %v, expected %d", count, len(scenarios))
	}
}

func TestServer_handleGetScenario(t *testing.T) {
	s := createTestServer(t)

	tests := []struct {
		name         string
		scenarioName string
		expectedCode int
	}{
		{
			name:         "existing scenario",
			scenarioName: "test-scenario",
			expectedCode: http.StatusOK,
		},
		{
			name:         "non-existent scenario",
			scenarioName: "non-existent",
			expectedCode: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/scenarios/"+tt.scenarioName, nil)
			req.SetPathValue("name", tt.scenarioName)
			rr := httptest.NewRecorder()

			s.handleGetScenario(rr, req)

			if rr.Code != tt.expectedCode {
				t.Errorf("handleGetScenario() status = %d, expected %d", rr.Code, tt.expectedCode)
			}
		})
	}
}

func TestServer_handleListComponents(t *testing.T) {
	s := createTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/components", nil)
	rr := httptest.NewRecorder()

	s.handleListComponents(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("handleListComponents() status = %d, expected %d", rr.Code, http.StatusOK)
	}

	var response map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	components, ok := response["components"].([]any)
	if !ok {
		t.Fatal("Response missing 'components' array")
	}

	if len(components) != 1 {
		t.Errorf("Expected 1 component, got %d", len(components))
	}
}

func TestServer_handleGetComponent(t *testing.T) {
	s := createTestServer(t)

	tests := []struct {
		name         string
		compName     string
		expectedCode int
	}{
		{
			name:         "existing component",
			compName:     "TestComponent",
			expectedCode: http.StatusOK,
		},
		{
			name:         "non-existent component",
			compName:     "NonExistent",
			expectedCode: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/components/"+tt.compName, nil)
			req.SetPathValue("name", tt.compName)
			rr := httptest.NewRecorder()

			s.handleGetComponent(rr, req)

			if rr.Code != tt.expectedCode {
				t.Errorf("handleGetComponent() status = %d, expected %d", rr.Code, tt.expectedCode)
			}
		})
	}
}

func TestServer_handleListRuns(t *testing.T) {
	s := createTestServer(t)

	// Add some test runs
	s.activeRuns["run-1"] = &RunInfo{
		ID:         "run-1",
		Status:     "running",
		ScenarioID: "test-scenario",
		StartTime:  time.Now(),
	}
	s.activeRuns["run-2"] = &RunInfo{
		ID:         "run-2",
		Status:     "completed",
		ScenarioID: "test-scenario",
		StartTime:  time.Now().Add(-time.Minute),
	}

	req := httptest.NewRequest(http.MethodGet, "/api/runs", nil)
	rr := httptest.NewRecorder()

	s.handleListRuns(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("handleListRuns() status = %d, expected %d", rr.Code, http.StatusOK)
	}

	var response map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	runs, ok := response["runs"].([]any)
	if !ok {
		t.Fatal("Response missing 'runs' array")
	}

	if len(runs) != 2 {
		t.Errorf("Expected 2 runs, got %d", len(runs))
	}
}

func TestServer_handleGetRun(t *testing.T) {
	s := createTestServer(t)

	s.activeRuns["run-1"] = &RunInfo{
		ID:         "run-1",
		Status:     "running",
		ScenarioID: "test-scenario",
		StartTime:  time.Now(),
	}

	tests := []struct {
		name         string
		runID        string
		expectedCode int
	}{
		{
			name:         "existing run",
			runID:        "run-1",
			expectedCode: http.StatusOK,
		},
		{
			name:         "non-existent run",
			runID:        "run-999",
			expectedCode: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/runs/"+tt.runID, nil)
			req.SetPathValue("id", tt.runID)
			rr := httptest.NewRecorder()

			s.handleGetRun(rr, req)

			if rr.Code != tt.expectedCode {
				t.Errorf("handleGetRun() status = %d, expected %d", rr.Code, tt.expectedCode)
			}
		})
	}
}

func TestServer_handleDeleteRun(t *testing.T) {
	s := createTestServer(t)

	cancelCalled := false
	s.activeRuns["run-1"] = &RunInfo{
		ID:         "run-1",
		Status:     "running",
		ScenarioID: "test-scenario",
		StartTime:  time.Now(),
		Cancel:     func() { cancelCalled = true },
	}

	tests := []struct {
		name         string
		runID        string
		expectedCode int
	}{
		{
			name:         "existing run",
			runID:        "run-1",
			expectedCode: http.StatusOK,
		},
		{
			name:         "non-existent run",
			runID:        "run-999",
			expectedCode: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodDelete, "/api/runs/"+tt.runID, nil)
			req.SetPathValue("id", tt.runID)
			rr := httptest.NewRecorder()

			s.handleDeleteRun(rr, req)

			if rr.Code != tt.expectedCode {
				t.Errorf("handleDeleteRun() status = %d, expected %d", rr.Code, tt.expectedCode)
			}
		})
	}

	if !cancelCalled {
		t.Error("handleDeleteRun() did not call cancel function")
	}

	if _, exists := s.activeRuns["run-1"]; exists {
		t.Error("handleDeleteRun() did not remove run from activeRuns")
	}
}

func TestServer_handleCreateRun_InvalidBody(t *testing.T) {
	s := createTestServer(t)

	req := httptest.NewRequest(http.MethodPost, "/api/runs", bytes.NewBufferString("invalid json"))
	rr := httptest.NewRecorder()

	s.handleCreateRun(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("handleCreateRun() status = %d, expected %d", rr.Code, http.StatusBadRequest)
	}
}

func TestServer_handleCreateRun_MissingScenarioName(t *testing.T) {
	s := createTestServer(t)

	body := CreateRunRequest{}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/runs", bytes.NewBuffer(jsonBody))
	rr := httptest.NewRecorder()

	s.handleCreateRun(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("handleCreateRun() status = %d, expected %d", rr.Code, http.StatusBadRequest)
	}
}

func TestServer_handleCreateRun_ScenarioNotFound(t *testing.T) {
	s := createTestServer(t)

	body := CreateRunRequest{
		ScenarioName: "non-existent-scenario",
	}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/runs", bytes.NewBuffer(jsonBody))
	rr := httptest.NewRecorder()

	s.handleCreateRun(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("handleCreateRun() status = %d, expected %d", rr.Code, http.StatusNotFound)
	}
}

func TestServer_handleListResults(t *testing.T) {
	s := createTestServer(t)

	// Add some test results
	storage := s.storage.(*mockStorage)
	storage.results["result-1"] = &results.RunResult{ID: "result-1"}
	storage.results["result-2"] = &results.RunResult{ID: "result-2"}

	req := httptest.NewRequest(http.MethodGet, "/api/results", nil)
	rr := httptest.NewRecorder()

	s.handleListResults(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("handleListResults() status = %d, expected %d", rr.Code, http.StatusOK)
	}

	var response map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	resultIDs, ok := response["results"].([]any)
	if !ok {
		t.Fatal("Response missing 'results' array")
	}

	if len(resultIDs) != 2 {
		t.Errorf("Expected 2 results, got %d", len(resultIDs))
	}
}

func TestServer_handleGetResult(t *testing.T) {
	s := createTestServer(t)

	storage := s.storage.(*mockStorage)
	storage.results["result-1"] = &results.RunResult{
		ID:        "result-1",
		Name:      "test-run",
		StartTime: time.Now(),
	}

	tests := []struct {
		name         string
		resultID     string
		expectedCode int
	}{
		{
			name:         "existing result",
			resultID:     "result-1",
			expectedCode: http.StatusOK,
		},
		{
			name:         "non-existent result",
			resultID:     "result-999",
			expectedCode: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/results/"+tt.resultID, nil)
			req.SetPathValue("id", tt.resultID)
			rr := httptest.NewRecorder()

			s.handleGetResult(rr, req)

			if rr.Code != tt.expectedCode {
				t.Errorf("handleGetResult() status = %d, expected %d", rr.Code, tt.expectedCode)
			}
		})
	}
}

func TestServer_handleDeleteResult(t *testing.T) {
	s := createTestServer(t)

	storage := s.storage.(*mockStorage)
	storage.results["result-1"] = &results.RunResult{ID: "result-1"}

	req := httptest.NewRequest(http.MethodDelete, "/api/results/result-1", nil)
	req.SetPathValue("id", "result-1")
	rr := httptest.NewRecorder()

	s.handleDeleteResult(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("handleDeleteResult() status = %d, expected %d", rr.Code, http.StatusOK)
	}

	if _, exists := storage.results["result-1"]; exists {
		t.Error("handleDeleteResult() did not remove result from storage")
	}
}

func TestServer_handleGetConfig(t *testing.T) {
	s := createTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/config", nil)
	rr := httptest.NewRecorder()

	s.handleGetConfig(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("handleGetConfig() status = %d, expected %d", rr.Code, http.StatusOK)
	}

	var response map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if response["name"] != "test-project" {
		t.Errorf("handleGetConfig() name = %q, expected 'test-project'", response["name"])
	}

	if response["version"] != "1.0.0" {
		t.Errorf("handleGetConfig() version = %q, expected '1.0.0'", response["version"])
	}
}

func TestCreateRunRequest_JSON(t *testing.T) {
	req := CreateRunRequest{
		ScenarioName: "test-scenario",
		Flags:        map[string]any{"debug": true},
		Tags:         []string{"integration"},
		Timeout:      "10m",
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Failed to marshal CreateRunRequest: %v", err)
	}

	var parsed CreateRunRequest
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Failed to unmarshal CreateRunRequest: %v", err)
	}

	if parsed.ScenarioName != req.ScenarioName {
		t.Errorf("ScenarioName = %q, expected %q", parsed.ScenarioName, req.ScenarioName)
	}

	if parsed.Timeout != req.Timeout {
		t.Errorf("Timeout = %q, expected %q", parsed.Timeout, req.Timeout)
	}
}

func TestRunResponse_JSON(t *testing.T) {
	now := time.Now()
	resp := RunResponse{
		ID:         "run-1",
		Status:     "running",
		ScenarioID: "test-scenario",
		StartTime:  now,
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Failed to marshal RunResponse: %v", err)
	}

	var parsed RunResponse
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Failed to unmarshal RunResponse: %v", err)
	}

	if parsed.ID != resp.ID {
		t.Errorf("ID = %q, expected %q", parsed.ID, resp.ID)
	}

	if parsed.Status != resp.Status {
		t.Errorf("Status = %q, expected %q", parsed.Status, resp.Status)
	}
}

func TestScenarioResponse_JSON(t *testing.T) {
	resp := ScenarioResponse{
		Name:        "test-scenario",
		Description: "A test scenario",
		Tags:        []string{"test", "integration"},
		Timeout:     "5m",
		FlowCount:   3,
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Failed to marshal ScenarioResponse: %v", err)
	}

	var parsed ScenarioResponse
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Failed to unmarshal ScenarioResponse: %v", err)
	}

	if parsed.Name != resp.Name {
		t.Errorf("Name = %q, expected %q", parsed.Name, resp.Name)
	}

	if parsed.FlowCount != resp.FlowCount {
		t.Errorf("FlowCount = %d, expected %d", parsed.FlowCount, resp.FlowCount)
	}
}

func TestComponentResponse_JSON(t *testing.T) {
	resp := ComponentResponse{
		Name:         "TestComponent",
		Type:         "setup",
		SourceFile:   "test.go",
		Dependencies: []string{"db", "cache"},
		Tags:         []string{"test"},
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Failed to marshal ComponentResponse: %v", err)
	}

	var parsed ComponentResponse
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Failed to unmarshal ComponentResponse: %v", err)
	}

	if parsed.Name != resp.Name {
		t.Errorf("Name = %q, expected %q", parsed.Name, resp.Name)
	}

	if len(parsed.Dependencies) != len(resp.Dependencies) {
		t.Errorf("Dependencies length = %d, expected %d", len(parsed.Dependencies), len(resp.Dependencies))
	}
}

// createTestServerWithSuites creates a test server with suite configuration
func createTestServerWithSuites(t *testing.T) *Server {
	t.Helper()

	registry := &discovery.Registry{
		Components: map[core.ComponentID]*core.Component{
			"SetupUser":    {Name: "SetupUser", Type: core.ComponentSetup, SourceFile: "test.go"},
			"DoTask":       {Name: "DoTask", Type: core.ComponentTask, SourceFile: "test.go"},
			"ValidateData": {Name: "ValidateData", Type: core.ComponentValidation, SourceFile: "test.go"},
		},
	}

	cfg := &config.Config{
		Name:    "test-project",
		Version: "1.0.0",
		Scenarios: []config.ScenarioConfig{
			{
				Name:    "smoke-test-1",
				Tags:    []string{"smoke"},
				Timeout: 5 * time.Minute,
				Flow:    []config.FlowItemConfig{{Setup: "SetupUser"}},
			},
			{
				Name:    "smoke-test-2",
				Tags:    []string{"smoke", "fast"},
				Timeout: 5 * time.Minute,
				Flow:    []config.FlowItemConfig{{Task: "DoTask"}},
			},
			{
				Name:    "integration-test",
				Tags:    []string{"integration"},
				Timeout: 10 * time.Minute,
				Flow:    []config.FlowItemConfig{{Validation: "ValidateData"}},
			},
			{
				Name:    "full-test",
				Tags:    []string{"full"},
				Timeout: 15 * time.Minute,
				Flow: []config.FlowItemConfig{
					{Setup: "SetupUser"},
					{Task: "DoTask"},
					{Validation: "ValidateData"},
				},
			},
		},
		Suites: map[string]config.SuiteConfig{
			"smoke-suite": {
				Description: "Run all smoke tests",
				Tags:        []string{"smoke"},
				Parallel:    2,
				FailFast:    true,
			},
			"explicit-suite": {
				Description: "Explicit scenario list",
				Scenarios:   []string{"smoke-test-1", "integration-test"},
			},
			"fast-smoke": {
				Description: "Fast smoke tests only",
				Tags:        []string{"smoke"},
				ExcludeTags: []string{"integration"},
			},
			"mixed-suite": {
				Description: "Mixed explicit and tags",
				Scenarios:   []string{"full-test"},
				Tags:        []string{"smoke"},
			},
		},
		Discovery: config.DiscoveryConfig{
			Paths: []string{"./"},
		},
		Execution: config.ExecutionConfig{
			DefaultTimeout: 5 * time.Minute,
			Parallelism:    1,
		},
	}

	resolver := scenario.NewResolver(cfg, registry)
	executor := execution.NewExecutor()
	storage := newMockStorage()
	eventBus := NewEmbeddedEventBus()

	s := &Server{
		config:     cfg,
		configPath: "/test/chronicle.yaml",
		registry:   registry,
		resolver:   resolver,
		executor:   executor,
		storage:    storage,
		eventBus:   eventBus,
		activeRuns: make(map[string]*RunInfo),
	}

	return s
}

func TestServer_handleListSuites(t *testing.T) {
	s := createTestServerWithSuites(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/suites", nil)
	rr := httptest.NewRecorder()

	s.handleListSuites(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("handleListSuites() status = %d, expected %d", rr.Code, http.StatusOK)
	}

	var response struct {
		Suites []SuiteResponse `json:"suites"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if len(response.Suites) != 4 {
		t.Errorf("handleListSuites() suites count = %d, expected 4", len(response.Suites))
	}

	// Verify suite properties
	var smokeSuite *SuiteResponse
	for i := range response.Suites {
		if response.Suites[i].Name == "smoke-suite" {
			smokeSuite = &response.Suites[i]
			break
		}
	}

	if smokeSuite == nil {
		t.Fatal("smoke-suite not found in response")
	}

	if smokeSuite.Description != "Run all smoke tests" {
		t.Errorf("smoke-suite description = %q, expected 'Run all smoke tests'", smokeSuite.Description)
	}

	if smokeSuite.Parallel != 2 {
		t.Errorf("smoke-suite parallel = %d, expected 2", smokeSuite.Parallel)
	}

	if !smokeSuite.FailFast {
		t.Error("smoke-suite FailFast should be true")
	}

	// Should have resolved scenarios
	if len(smokeSuite.ResolvedScenarios) != 2 {
		t.Errorf("smoke-suite resolved scenarios = %d, expected 2", len(smokeSuite.ResolvedScenarios))
	}
}

func TestServer_handleGetSuite(t *testing.T) {
	s := createTestServerWithSuites(t)

	tests := []struct {
		name                string
		suiteName           string
		expectedCode        int
		expectedScenarios   int
		expectedDescription string
	}{
		{
			name:                "existing smoke suite",
			suiteName:           "smoke-suite",
			expectedCode:        http.StatusOK,
			expectedScenarios:   2,
			expectedDescription: "Run all smoke tests",
		},
		{
			name:              "explicit suite",
			suiteName:         "explicit-suite",
			expectedCode:      http.StatusOK,
			expectedScenarios: 2,
		},
		{
			name:              "mixed suite",
			suiteName:         "mixed-suite",
			expectedCode:      http.StatusOK,
			expectedScenarios: 3, // full-test + 2 smoke tests
		},
		{
			name:         "non-existent suite",
			suiteName:    "non-existent",
			expectedCode: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/suites/"+tt.suiteName, nil)
			req.SetPathValue("name", tt.suiteName)
			rr := httptest.NewRecorder()

			s.handleGetSuite(rr, req)

			if rr.Code != tt.expectedCode {
				t.Errorf("handleGetSuite() status = %d, expected %d", rr.Code, tt.expectedCode)
			}

			if tt.expectedCode == http.StatusAccepted {
				var suite SuiteResponse
				if err := json.Unmarshal(rr.Body.Bytes(), &suite); err != nil {
					t.Fatalf("Failed to parse response: %v", err)
				}

				if suite.Name != tt.suiteName {
					t.Errorf("suite name = %q, expected %q", suite.Name, tt.suiteName)
				}

				if len(suite.ResolvedScenarios) != tt.expectedScenarios {
					t.Errorf("resolved scenarios = %d, expected %d", len(suite.ResolvedScenarios), tt.expectedScenarios)
				}

				if tt.expectedDescription != "" && suite.Description != tt.expectedDescription {
					t.Errorf("description = %q, expected %q", suite.Description, tt.expectedDescription)
				}
			}
		})
	}
}

func TestServer_handleBatchRun(t *testing.T) {
	s := createTestServerWithSuites(t)

	tests := []struct {
		name              string
		requestBody       BatchRunRequest
		expectedCode      int
		expectedScenarios int
	}{
		{
			name:              "batch run by suite",
			requestBody:       BatchRunRequest{Suite: "smoke-suite"},
			expectedCode:      http.StatusAccepted,
			expectedScenarios: 2,
		},
		{
			name:              "batch run by explicit scenarios",
			requestBody:       BatchRunRequest{Scenarios: []string{"smoke-test-1", "integration-test"}},
			expectedCode:      http.StatusAccepted,
			expectedScenarios: 2,
		},
		{
			name:              "batch run by tags",
			requestBody:       BatchRunRequest{Tags: []string{"smoke"}},
			expectedCode:      http.StatusAccepted,
			expectedScenarios: 2,
		},
		{
			name:              "batch run with exclude tags",
			requestBody:       BatchRunRequest{Tags: []string{"smoke"}, ExcludeTags: []string{"fast"}},
			expectedCode:      http.StatusAccepted,
			expectedScenarios: 1, // smoke-test-2 has "fast" tag
		},
		{
			name:              "batch run all scenarios",
			requestBody:       BatchRunRequest{},
			expectedCode:      http.StatusAccepted,
			expectedScenarios: 4,
		},
		{
			name:         "batch run non-existent suite",
			requestBody:  BatchRunRequest{Suite: "non-existent"},
			expectedCode: http.StatusNotFound,
		},
		{
			name:              "batch run non-existent scenario",
			requestBody:       BatchRunRequest{Scenarios: []string{"non-existent"}},
			expectedCode:      http.StatusAccepted,
			expectedScenarios: 1, // The handler doesn't validate scenario names
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/runs/batch", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()

			s.handleBatchRun(rr, req)

			if rr.Code != tt.expectedCode {
				t.Errorf("handleBatchRun() status = %d, expected %d: %s", rr.Code, tt.expectedCode, rr.Body.String())
			}

			if tt.expectedCode == http.StatusAccepted {
				var resp BatchRunResponse
				if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
					t.Fatalf("Failed to parse response: %v", err)
				}

				if resp.ID == "" {
					t.Error("response should have non-empty ID")
				}

				if resp.Status != "running" {
					t.Errorf("status = %q, expected 'running'", resp.Status)
				}

				if len(resp.Scenarios) != tt.expectedScenarios {
					t.Errorf("scenarios = %d, expected %d", len(resp.Scenarios), tt.expectedScenarios)
				}
			}
		})
	}
}

func TestServer_handleBatchRun_WithOptions(t *testing.T) {
	s := createTestServerWithSuites(t)

	t.Run("batch run with parallel and fail_fast", func(t *testing.T) {
		body, _ := json.Marshal(BatchRunRequest{
			Scenarios: []string{"smoke-test-1", "smoke-test-2"},
			Parallel:  4,
			FailFast:  true,
			Timeout:   "20m",
		})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/runs/batch", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()

		s.handleBatchRun(rr, req)

		if rr.Code != http.StatusAccepted {
			t.Errorf("handleBatchRun() status = %d, expected %d", rr.Code, http.StatusOK)
		}

		var resp BatchRunResponse
		_ = json.Unmarshal(rr.Body.Bytes(), &resp)

		if len(resp.Scenarios) != 2 {
			t.Errorf("scenarios = %d, expected 2", len(resp.Scenarios))
		}
	})

	t.Run("batch run with flags", func(t *testing.T) {
		body, _ := json.Marshal(BatchRunRequest{
			Scenarios: []string{"smoke-test-1"},
			Flags:     map[string]any{"debug": true, "env": "test"},
		})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/runs/batch", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()

		s.handleBatchRun(rr, req)

		if rr.Code != http.StatusAccepted {
			t.Errorf("handleBatchRun() status = %d, expected %d", rr.Code, http.StatusOK)
		}
	})
}

func TestBatchRunRequest_JSON(t *testing.T) {
	req := BatchRunRequest{
		Suite:       "smoke-suite",
		Scenarios:   []string{"test-1", "test-2"},
		Tags:        []string{"smoke", "fast"},
		ExcludeTags: []string{"slow"},
		Flags:       map[string]any{"debug": true},
		Parallel:    4,
		FailFast:    true,
		Timeout:     "10m",
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Failed to marshal BatchRunRequest: %v", err)
	}

	var parsed BatchRunRequest
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Failed to unmarshal BatchRunRequest: %v", err)
	}

	if parsed.Suite != req.Suite {
		t.Errorf("Suite = %q, expected %q", parsed.Suite, req.Suite)
	}

	if len(parsed.Scenarios) != len(req.Scenarios) {
		t.Errorf("Scenarios length = %d, expected %d", len(parsed.Scenarios), len(req.Scenarios))
	}

	if len(parsed.Tags) != len(req.Tags) {
		t.Errorf("Tags length = %d, expected %d", len(parsed.Tags), len(req.Tags))
	}

	if parsed.Parallel != req.Parallel {
		t.Errorf("Parallel = %d, expected %d", parsed.Parallel, req.Parallel)
	}

	if parsed.FailFast != req.FailFast {
		t.Errorf("FailFast = %v, expected %v", parsed.FailFast, req.FailFast)
	}
}

func TestBatchRunResponse_JSON(t *testing.T) {
	now := time.Now()
	resp := BatchRunResponse{
		ID:        "batch-123",
		Status:    "running",
		Scenarios: []string{"test-1", "test-2", "test-3"},
		StartTime: now,
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Failed to marshal BatchRunResponse: %v", err)
	}

	var parsed BatchRunResponse
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Failed to unmarshal BatchRunResponse: %v", err)
	}

	if parsed.ID != resp.ID {
		t.Errorf("ID = %q, expected %q", parsed.ID, resp.ID)
	}

	if len(parsed.Scenarios) != len(resp.Scenarios) {
		t.Errorf("Scenarios length = %d, expected %d", len(parsed.Scenarios), len(resp.Scenarios))
	}
}

func TestSuiteResponse_JSON(t *testing.T) {
	resp := SuiteResponse{
		Name:              "smoke-suite",
		Description:       "Run smoke tests",
		Scenarios:         []string{"test-1"},
		Tags:              []string{"smoke"},
		ExcludeTags:       []string{"slow"},
		Parallel:          2,
		FailFast:          true,
		ResolvedScenarios: []string{"test-1", "test-2", "test-3"},
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Failed to marshal SuiteResponse: %v", err)
	}

	var parsed SuiteResponse
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Failed to unmarshal SuiteResponse: %v", err)
	}

	if parsed.Name != resp.Name {
		t.Errorf("Name = %q, expected %q", parsed.Name, resp.Name)
	}

	if parsed.Description != resp.Description {
		t.Errorf("Description = %q, expected %q", parsed.Description, resp.Description)
	}

	if len(parsed.ResolvedScenarios) != len(resp.ResolvedScenarios) {
		t.Errorf("ResolvedScenarios length = %d, expected %d", len(parsed.ResolvedScenarios), len(resp.ResolvedScenarios))
	}

	if parsed.Parallel != resp.Parallel {
		t.Errorf("Parallel = %d, expected %d", parsed.Parallel, resp.Parallel)
	}

	if parsed.FailFast != resp.FailFast {
		t.Errorf("FailFast = %v, expected %v", parsed.FailFast, resp.FailFast)
	}
}
