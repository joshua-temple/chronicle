package daemon

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/joshua-temple/chronicle/pkg/core"
	"github.com/joshua-temple/chronicle/pkg/execution"
	"github.com/joshua-temple/chronicle/pkg/results"
	"github.com/joshua-temple/chronicle/pkg/scenario"
)

// CreateRunRequest represents a request to create a new run.
type CreateRunRequest struct {
	ScenarioName string         `json:"scenario_name"`
	Flags        map[string]any `json:"flags,omitempty"`
	Tags         []string       `json:"tags,omitempty"`
	Timeout      string         `json:"timeout,omitempty"`
}

// BatchRunRequest represents a request to run multiple scenarios.
type BatchRunRequest struct {
	// Scenarios to run (by name)
	Scenarios []string `json:"scenarios,omitempty"`
	// Suite to run (by name)
	Suite string `json:"suite,omitempty"`
	// Tags to filter scenarios (include)
	Tags []string `json:"tags,omitempty"`
	// ExcludeTags to filter scenarios (exclude)
	ExcludeTags []string `json:"exclude_tags,omitempty"`
	// Flags to apply to all scenarios
	Flags map[string]any `json:"flags,omitempty"`
	// Parallelism for running scenarios
	Parallel int `json:"parallel,omitempty"`
	// FailFast stops on first failure
	FailFast bool `json:"fail_fast,omitempty"`
	// Timeout for the entire batch
	Timeout string `json:"timeout,omitempty"`
}

// BatchRunResponse represents the response from a batch run request.
type BatchRunResponse struct {
	ID         string   `json:"id"`
	Status     string   `json:"status"`
	Scenarios  []string `json:"scenarios"`
	StartTime  time.Time `json:"start_time"`
}

// SuiteResponse represents a suite in API responses.
type SuiteResponse struct {
	Name              string   `json:"name"`
	Description       string   `json:"description,omitempty"`
	Scenarios         []string `json:"scenarios,omitempty"`
	Tags              []string `json:"tags,omitempty"`
	ExcludeTags       []string `json:"exclude_tags,omitempty"`
	Parallel          int      `json:"parallel,omitempty"`
	FailFast          bool     `json:"fail_fast,omitempty"`
	ResolvedScenarios []string `json:"resolved_scenarios,omitempty"`
}

// RunResponse represents a run in API responses.
type RunResponse struct {
	ID          string     `json:"id"`
	Status      string     `json:"status"`
	ScenarioID  string     `json:"scenario_id"`
	StartTime   time.Time  `json:"start_time"`
	EndTime     *time.Time `json:"end_time,omitempty"`
	Duration    string     `json:"duration,omitempty"`
	Error       string     `json:"error,omitempty"`
	ResultID    string     `json:"result_id,omitempty"`
}

// ScenarioResponse represents a scenario in API responses.
type ScenarioResponse struct {
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	Timeout     string   `json:"timeout,omitempty"`
	FlowCount   int      `json:"flow_count"`
}

// ComponentResponse represents a component in API responses.
type ComponentResponse struct {
	Name         string   `json:"name"`
	Type         string   `json:"type"`
	SourceFile   string   `json:"source_file"`
	Dependencies []string `json:"dependencies,omitempty"`
	Tags         []string `json:"tags,omitempty"`
}

// handleHealth returns server health status.
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"status":    "healthy",
		"timestamp": time.Now().UTC(),
		"version":   "1.0.0",
	})
}

// handleCreateRun creates and starts a new scenario run.
func (s *Server) handleCreateRun(w http.ResponseWriter, r *http.Request) {
	var req CreateRunRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.ScenarioName == "" {
		writeError(w, http.StatusBadRequest, "scenario_name is required")
		return
	}

	// Find and resolve scenario
	s.mu.RLock()
	scenarios, err := s.resolver.ResolveAll()
	s.mu.RUnlock()

	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to resolve scenarios")
		return
	}

	var targetScenario *scenario.Scenario
	for _, sc := range scenarios {
		if sc.Name == req.ScenarioName {
			targetScenario = sc
			break
		}
	}

	if targetScenario == nil {
		writeError(w, http.StatusNotFound, "scenario not found")
		return
	}

	// Apply flags from request
	if req.Flags != nil {
		for k, v := range req.Flags {
			targetScenario.Flags[k] = v
		}
	}

	// Parse timeout
	timeout := 30 * time.Minute
	if req.Timeout != "" {
		if parsed, err := time.ParseDuration(req.Timeout); err == nil {
			timeout = parsed
		}
	}

	// Create run context
	runID := uuid.New().String()
	ctx, cancel := context.WithTimeout(context.Background(), timeout)

	// Track the run
	runInfo := &RunInfo{
		ID:         runID,
		Status:     "running",
		ScenarioID: req.ScenarioName,
		StartTime:  time.Now(),
		Cancel:     cancel,
	}

	s.mu.Lock()
	s.activeRuns[runID] = runInfo
	s.mu.Unlock()

	// Publish start event
	s.eventBus.Publish(Event{
		Type:      EventRunStarted,
		Timestamp: time.Now(),
		Data: map[string]any{
			"run_id":   runID,
			"scenario": req.ScenarioName,
		},
	})

	// Execute asynchronously
	go func() {
		defer cancel()

		result := s.executor.Execute(ctx, targetScenario)

		// Update run status
		s.mu.Lock()
		if info, ok := s.activeRuns[runID]; ok {
			if result.State == execution.StateCompleted {
				info.Status = "completed"
			} else {
				info.Status = "failed"
			}

			// Store result
			runResult := results.NewRunResult(s.config.Name, []*execution.ScenarioResult{result})
			runResult.ID = runID
			if err := s.storage.Save(context.Background(), runResult); err != nil {
				info.Status = "error"
			}
		}
		s.mu.Unlock()

		// Publish completion event
		eventType := EventRunCompleted
		if result.State != execution.StateCompleted {
			eventType = EventRunFailed
		}
		s.eventBus.Publish(Event{
			Type:      eventType,
			Timestamp: time.Now(),
			Data: map[string]any{
				"run_id":   runID,
				"scenario": req.ScenarioName,
				"state":    result.State.String(),
			},
		})
	}()

	writeJSON(w, http.StatusAccepted, RunResponse{
		ID:         runID,
		Status:     "running",
		ScenarioID: req.ScenarioName,
		StartTime:  runInfo.StartTime,
	})
}

// handleListRuns lists all runs.
func (s *Server) handleListRuns(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	runs := make([]RunResponse, 0, len(s.activeRuns))
	for _, info := range s.activeRuns {
		runs = append(runs, RunResponse{
			ID:         info.ID,
			Status:     info.Status,
			ScenarioID: info.ScenarioID,
			StartTime:  info.StartTime,
		})
	}
	s.mu.RUnlock()

	writeJSON(w, http.StatusOK, map[string]any{
		"runs":  runs,
		"count": len(runs),
	})
}

// handleGetRun gets details of a specific run.
func (s *Server) handleGetRun(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	s.mu.RLock()
	info, ok := s.activeRuns[id]
	s.mu.RUnlock()

	if !ok {
		writeError(w, http.StatusNotFound, "run not found")
		return
	}

	resp := RunResponse{
		ID:         info.ID,
		Status:     info.Status,
		ScenarioID: info.ScenarioID,
		StartTime:  info.StartTime,
	}

	if info.Status == "completed" || info.Status == "failed" {
		endTime := time.Now()
		resp.EndTime = &endTime
		resp.Duration = endTime.Sub(info.StartTime).String()
		resp.ResultID = info.ID // Result ID matches run ID
	}

	writeJSON(w, http.StatusOK, resp)
}

// handleDeleteRun cancels and removes a run.
func (s *Server) handleDeleteRun(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	s.mu.Lock()
	info, ok := s.activeRuns[id]
	if ok {
		if info.Cancel != nil {
			info.Cancel()
		}
		delete(s.activeRuns, id)
	}
	s.mu.Unlock()

	if !ok {
		writeError(w, http.StatusNotFound, "run not found")
		return
	}

	// Publish cancel event
	s.eventBus.Publish(Event{
		Type:      EventRunCanceled,
		Timestamp: time.Now(),
		Data: map[string]any{
			"run_id": id,
		},
	})

	writeJSON(w, http.StatusOK, map[string]string{"status": "canceled"})
}

// handleListScenarios lists all available scenarios.
func (s *Server) handleListScenarios(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	scenarios, err := s.resolver.ResolveAll()
	s.mu.RUnlock()

	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to resolve scenarios")
		return
	}

	resp := make([]ScenarioResponse, 0, len(scenarios))
	for _, sc := range scenarios {
		resp = append(resp, ScenarioResponse{
			Name:        sc.Name,
			Description: sc.Description,
			Tags:        sc.Tags,
			Timeout:     sc.Timeout.String(),
			FlowCount:   len(sc.Flow),
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"scenarios": resp,
		"count":     len(resp),
	})
}

// handleGetScenario gets details of a specific scenario.
func (s *Server) handleGetScenario(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")

	s.mu.RLock()
	scenarios, err := s.resolver.ResolveAll()
	s.mu.RUnlock()

	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to resolve scenarios")
		return
	}

	for _, sc := range scenarios {
		if sc.Name == name {
			writeJSON(w, http.StatusOK, map[string]any{
				"name":        sc.Name,
				"description": sc.Description,
				"tags":        sc.Tags,
				"timeout":     sc.Timeout.String(),
				"flow":        sc.Flow,
				"flags":       sc.Flags,
				"options":     sc.Options,
			})
			return
		}
	}

	writeError(w, http.StatusNotFound, "scenario not found")
}

// handleListComponents lists all discovered components.
func (s *Server) handleListComponents(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	components := s.registry.Components
	s.mu.RUnlock()

	resp := make([]ComponentResponse, 0, len(components))
	for _, comp := range components {
		deps := make([]string, 0, len(comp.Requires))
		for _, req := range comp.Requires {
			deps = append(deps, req.Key)
		}
		resp = append(resp, ComponentResponse{
			Name:         comp.Name,
			Type:         string(comp.Type),
			SourceFile:   comp.SourceFile,
			Dependencies: deps,
			Tags:         comp.Tags,
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"components": resp,
		"count":      len(resp),
	})
}

// handleGetComponent gets details of a specific component.
func (s *Server) handleGetComponent(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")

	s.mu.RLock()
	comp, ok := s.registry.GetComponent(core.ComponentID(name))
	s.mu.RUnlock()

	if !ok {
		writeError(w, http.StatusNotFound, "component not found")
		return
	}

	deps := make([]string, 0, len(comp.Requires))
	for _, req := range comp.Requires {
		deps = append(deps, req.Key)
	}

	writeJSON(w, http.StatusOK, ComponentResponse{
		Name:         comp.Name,
		Type:         string(comp.Type),
		SourceFile:   comp.SourceFile,
		Dependencies: deps,
		Tags:         comp.Tags,
	})
}

// handleListResults lists stored results.
func (s *Server) handleListResults(w http.ResponseWriter, r *http.Request) {
	resultIDs, err := s.storage.List(context.Background())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list results")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"results": resultIDs,
		"count":   len(resultIDs),
	})
}

// handleGetResult gets a specific result.
func (s *Server) handleGetResult(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	result, err := s.storage.Load(context.Background(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "result not found")
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// handleDeleteResult deletes a specific result.
func (s *Server) handleDeleteResult(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	if err := s.storage.Delete(context.Background(), id); err != nil {
		writeError(w, http.StatusNotFound, "result not found")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// handleGetConfig returns the current configuration.
func (s *Server) handleGetConfig(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	cfg := s.config
	s.mu.RUnlock()

	writeJSON(w, http.StatusOK, map[string]any{
		"name":       cfg.Name,
		"version":    cfg.Version,
		"discovery":  cfg.Discovery,
		"execution":  cfg.Execution,
		"results":    cfg.Results,
		"scenarios":  len(cfg.Scenarios),
		"configPath": s.configPath,
	})
}

// handleReloadConfig reloads the configuration.
func (s *Server) handleReloadConfig(w http.ResponseWriter, r *http.Request) {
	if err := s.ReloadConfig(); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "reloaded"})
}

// handleBatchRun creates and starts a batch of scenario runs.
func (s *Server) handleBatchRun(w http.ResponseWriter, r *http.Request) {
	var req BatchRunRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Collect scenarios to run
	var scenarioNames []string

	// If suite is specified, get scenarios from it
	if req.Suite != "" {
		s.mu.RLock()
		suiteScenarios, ok := s.config.GetSuiteScenarios(req.Suite)
		s.mu.RUnlock()

		if !ok {
			writeError(w, http.StatusNotFound, "suite not found")
			return
		}
		scenarioNames = append(scenarioNames, suiteScenarios...)
	}

	// Add explicitly specified scenarios
	scenarioNames = append(scenarioNames, req.Scenarios...)

	// Resolve all scenarios
	s.mu.RLock()
	allScenarios, err := s.resolver.ResolveAll()
	s.mu.RUnlock()

	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to resolve scenarios")
		return
	}

	// Filter by tags if specified
	if len(req.Tags) > 0 || len(req.ExcludeTags) > 0 {
		tagSet := make(map[string]bool)
		for _, t := range req.Tags {
			tagSet[t] = true
		}
		excludeSet := make(map[string]bool)
		for _, t := range req.ExcludeTags {
			excludeSet[t] = true
		}

		for _, sc := range allScenarios {
			// Skip if already in list
			found := false
			for _, name := range scenarioNames {
				if name == sc.Name {
					found = true
					break
				}
			}
			if found {
				continue
			}

			// Check include tags (any match)
			if len(req.Tags) > 0 {
				hasTag := false
				for _, t := range sc.Tags {
					if tagSet[t] {
						hasTag = true
						break
					}
				}
				if !hasTag {
					continue
				}
			}

			// Check exclude tags (any match excludes)
			if len(req.ExcludeTags) > 0 {
				excluded := false
				for _, t := range sc.Tags {
					if excludeSet[t] {
						excluded = true
						break
					}
				}
				if excluded {
					continue
				}
			}

			scenarioNames = append(scenarioNames, sc.Name)
		}
	}

	// If no scenarios and no filters specified, run all
	if len(scenarioNames) == 0 && len(req.Tags) == 0 && req.Suite == "" && len(req.Scenarios) == 0 {
		for _, sc := range allScenarios {
			scenarioNames = append(scenarioNames, sc.Name)
		}
	}

	if len(scenarioNames) == 0 {
		writeError(w, http.StatusBadRequest, "no scenarios match the criteria")
		return
	}

	// Deduplicate scenario names
	seen := make(map[string]bool)
	var uniqueNames []string
	for _, name := range scenarioNames {
		if !seen[name] {
			seen[name] = true
			uniqueNames = append(uniqueNames, name)
		}
	}
	scenarioNames = uniqueNames

	// Find matching scenarios
	var targetScenarios []*scenario.Scenario
	for _, name := range scenarioNames {
		for _, sc := range allScenarios {
			if sc.Name == name {
				// Apply flags from request
				if req.Flags != nil {
					for k, v := range req.Flags {
						sc.Flags[k] = v
					}
				}
				targetScenarios = append(targetScenarios, sc)
				break
			}
		}
	}

	// Parse timeout
	timeout := 30 * time.Minute
	if req.Timeout != "" {
		if parsed, err := time.ParseDuration(req.Timeout); err == nil {
			timeout = parsed
		}
	}

	// Create batch run context
	batchID := uuid.New().String()
	ctx, cancel := context.WithTimeout(context.Background(), timeout)

	// Track the batch run
	runInfo := &RunInfo{
		ID:         batchID,
		Status:     "running",
		ScenarioID: "batch:" + req.Suite,
		StartTime:  time.Now(),
		Cancel:     cancel,
	}

	s.mu.Lock()
	s.activeRuns[batchID] = runInfo
	s.mu.Unlock()

	// Publish start event
	s.eventBus.Publish(Event{
		Type:      EventRunStarted,
		Timestamp: time.Now(),
		Data: map[string]any{
			"run_id":    batchID,
			"scenarios": scenarioNames,
			"batch":     true,
		},
	})

	// Execute asynchronously
	go func() {
		defer cancel()

		// Set up executor with parallelism
		parallel := req.Parallel
		if parallel <= 0 {
			parallel = 1
		}

		executor := execution.NewExecutor(
			execution.WithParallelism(parallel),
			execution.WithFailFast(req.FailFast),
			execution.WithDefaultTimeout(s.config.Execution.DefaultTimeout),
		)

		// Register components
		s.mu.RLock()
		for _, comp := range s.registry.Components {
			executor.RegisterComponent(comp)
		}
		s.mu.RUnlock()

		// Execute all scenarios
		execResults := executor.ExecuteMultiple(ctx, targetScenarios)

		// Collect results
		s.mu.Lock()
		if info, ok := s.activeRuns[batchID]; ok {
			allPassed := true
			for _, result := range execResults {
				if result.State != execution.StateCompleted {
					allPassed = false
					break
				}
			}

			if allPassed {
				info.Status = "completed"
			} else {
				info.Status = "failed"
			}

			// Store result
			runResult := results.NewRunResult(s.config.Name, execResults)
			runResult.ID = batchID
			if err := s.storage.Save(context.Background(), runResult); err != nil {
				info.Status = "error"
			}
		}
		s.mu.Unlock()

		// Publish completion event
		eventType := EventRunCompleted
		allPassed := true
		for _, result := range execResults {
			if result.State != execution.StateCompleted {
				allPassed = false
				eventType = EventRunFailed
				break
			}
		}
		s.eventBus.Publish(Event{
			Type:      eventType,
			Timestamp: time.Now(),
			Data: map[string]any{
				"run_id":    batchID,
				"scenarios": scenarioNames,
				"batch":     true,
				"passed":    allPassed,
			},
		})
	}()

	writeJSON(w, http.StatusAccepted, BatchRunResponse{
		ID:        batchID,
		Status:    "running",
		Scenarios: scenarioNames,
		StartTime: runInfo.StartTime,
	})
}

// handleListSuites lists all available suites.
func (s *Server) handleListSuites(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	suites := s.config.Suites
	s.mu.RUnlock()

	resp := make([]SuiteResponse, 0, len(suites))
	for name, suite := range suites {
		scenarios, _ := s.config.GetSuiteScenarios(name)
		resp = append(resp, SuiteResponse{
			Name:              name,
			Description:       suite.Description,
			Scenarios:         suite.Scenarios,
			Tags:              suite.Tags,
			ExcludeTags:       suite.ExcludeTags,
			Parallel:          suite.Parallel,
			FailFast:          suite.FailFast,
			ResolvedScenarios: scenarios,
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"suites": resp,
		"count":  len(resp),
	})
}

// handleGetSuite gets details of a specific suite.
func (s *Server) handleGetSuite(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")

	s.mu.RLock()
	suite, ok := s.config.GetSuite(name)
	scenarios, _ := s.config.GetSuiteScenarios(name)
	s.mu.RUnlock()

	if !ok {
		writeError(w, http.StatusNotFound, "suite not found")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"name":            name,
		"description":     suite.Description,
		"scenarios":       suite.Scenarios,
		"tags":            suite.Tags,
		"exclude_tags":    suite.ExcludeTags,
		"parallel":        suite.Parallel,
		"fail_fast":       suite.FailFast,
		"resolved_scenarios": scenarios,
	})
}
