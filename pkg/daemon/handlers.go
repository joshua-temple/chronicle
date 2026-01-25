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
