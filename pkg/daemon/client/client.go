// Package client provides a client for interacting with Chronicle daemon.
package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client is a Chronicle daemon client.
type Client struct {
	baseURL    string
	httpClient *http.Client
	apiKey     string
}

// Option configures a Client.
type Option func(*Client)

// WithAPIKey sets the API key for authentication.
func WithAPIKey(key string) Option {
	return func(c *Client) {
		c.apiKey = key
	}
}

// WithTimeout sets the HTTP client timeout.
func WithTimeout(timeout time.Duration) Option {
	return func(c *Client) {
		c.httpClient.Timeout = timeout
	}
}

// New creates a new daemon client.
func New(baseURL string, opts ...Option) *Client {
	c := &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

// HealthResponse represents the health check response.
type HealthResponse struct {
	Status    string `json:"status"`
	Timestamp string `json:"timestamp"`
	Version   string `json:"version"`
}

// Health checks if the daemon is healthy.
func (c *Client) Health(ctx context.Context) (*HealthResponse, error) {
	resp, err := c.get(ctx, "/health")
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	var health HealthResponse
	if err := json.NewDecoder(resp.Body).Decode(&health); err != nil {
		return nil, fmt.Errorf("decode health response: %w", err)
	}

	return &health, nil
}

// IsHealthy returns true if the daemon is healthy.
func (c *Client) IsHealthy(ctx context.Context) bool {
	health, err := c.Health(ctx)
	if err != nil {
		return false
	}
	return health.Status == "healthy"
}

// RunRequest represents a request to run scenarios.
type RunRequest struct {
	// Single scenario name
	ScenarioName string `json:"scenario_name,omitempty"`
	// Batch options
	Scenarios   []string       `json:"scenarios,omitempty"`
	Suite       string         `json:"suite,omitempty"`
	Tags        []string       `json:"tags,omitempty"`
	ExcludeTags []string       `json:"exclude_tags,omitempty"`
	Flags       map[string]any `json:"flags,omitempty"`
	Parallel    int            `json:"parallel,omitempty"`
	FailFast    bool           `json:"fail_fast,omitempty"`
	Timeout     string         `json:"timeout,omitempty"`
}

// RunResponse represents a run response.
type RunResponse struct {
	ID         string    `json:"id"`
	Status     string    `json:"status"`
	ScenarioID string    `json:"scenario_id,omitempty"`
	Scenarios  []string  `json:"scenarios,omitempty"`
	StartTime  time.Time `json:"start_time"`
	EndTime    *time.Time `json:"end_time,omitempty"`
	Duration   string    `json:"duration,omitempty"`
	Error      string    `json:"error,omitempty"`
}

// RunScenario runs a single scenario.
func (c *Client) RunScenario(ctx context.Context, req *RunRequest) (*RunResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	resp, err := c.post(ctx, "/runs", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	var run RunResponse
	if err := json.NewDecoder(resp.Body).Decode(&run); err != nil {
		return nil, fmt.Errorf("decode run response: %w", err)
	}

	return &run, nil
}

// RunBatch runs multiple scenarios as a batch.
func (c *Client) RunBatch(ctx context.Context, req *RunRequest) (*RunResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	resp, err := c.post(ctx, "/runs/batch", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	var run RunResponse
	if err := json.NewDecoder(resp.Body).Decode(&run); err != nil {
		return nil, fmt.Errorf("decode run response: %w", err)
	}

	return &run, nil
}

// GetRun gets the status of a run.
func (c *Client) GetRun(ctx context.Context, id string) (*RunResponse, error) {
	resp, err := c.get(ctx, "/runs/"+id)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	var run RunResponse
	if err := json.NewDecoder(resp.Body).Decode(&run); err != nil {
		return nil, fmt.Errorf("decode run response: %w", err)
	}

	return &run, nil
}

// CancelRun cancels a running scenario.
func (c *Client) CancelRun(ctx context.Context, id string) error {
	resp, err := c.delete(ctx, "/runs/"+id)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	return nil
}

// WaitForRun waits for a run to complete.
func (c *Client) WaitForRun(ctx context.Context, id string, pollInterval time.Duration) (*RunResponse, error) {
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
			run, err := c.GetRun(ctx, id)
			if err != nil {
				return nil, err
			}
			if run.Status == "completed" || run.Status == "failed" || run.Status == "canceled" {
				return run, nil
			}
		}
	}
}

// ScenarioResponse represents a scenario.
type ScenarioResponse struct {
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	Timeout     string   `json:"timeout,omitempty"`
	FlowCount   int      `json:"flow_count"`
}

// ListScenarios lists all scenarios.
func (c *Client) ListScenarios(ctx context.Context) ([]ScenarioResponse, error) {
	resp, err := c.get(ctx, "/scenarios")
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	var result struct {
		Scenarios []ScenarioResponse `json:"scenarios"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode scenarios: %w", err)
	}

	return result.Scenarios, nil
}

// SuiteResponse represents a suite.
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

// ListSuites lists all suites.
func (c *Client) ListSuites(ctx context.Context) ([]SuiteResponse, error) {
	resp, err := c.get(ctx, "/suites")
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	var result struct {
		Suites []SuiteResponse `json:"suites"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode suites: %w", err)
	}

	return result.Suites, nil
}

// GetSuite gets a specific suite.
func (c *Client) GetSuite(ctx context.Context, name string) (*SuiteResponse, error) {
	resp, err := c.get(ctx, "/suites/"+name)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	var suite SuiteResponse
	if err := json.NewDecoder(resp.Body).Decode(&suite); err != nil {
		return nil, fmt.Errorf("decode suite: %w", err)
	}

	return &suite, nil
}

// Helper methods for HTTP requests

func (c *Client) get(ctx context.Context, path string) (*http.Response, error) {
	return c.doRequest(ctx, http.MethodGet, path, nil)
}

func (c *Client) post(ctx context.Context, path string, body io.Reader) (*http.Response, error) {
	return c.doRequest(ctx, http.MethodPost, path, body)
}

func (c *Client) delete(ctx context.Context, path string) (*http.Response, error) {
	return c.doRequest(ctx, http.MethodDelete, path, nil)
}

func (c *Client) doRequest(ctx context.Context, method, path string, body io.Reader) (*http.Response, error) {
	url := c.baseURL + "/api/v1" + path

	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	if resp.StatusCode >= 400 {
		defer func() { _ = resp.Body.Close() }()
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(bodyBytes))
	}

	return resp, nil
}
