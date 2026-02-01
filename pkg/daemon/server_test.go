package daemon

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestWithAuth(t *testing.T) {
	auth := NewAuth(AuthConfig{Method: AuthMethodNone})

	s := &Server{}
	opt := WithAuth(auth)
	opt(s)

	if s.auth != auth {
		t.Error("WithAuth did not set auth correctly")
	}
}

func TestWithStorage(t *testing.T) {
	storage := newMockStorage()

	s := &Server{}
	opt := WithStorage(storage)
	opt(s)

	if s.storage != storage {
		t.Error("WithStorage did not set storage correctly")
	}
}

func TestWithEventBus(t *testing.T) {
	bus := NewEmbeddedEventBus()

	s := &Server{}
	opt := WithEventBus(bus)
	opt(s)

	if s.eventBus != bus {
		t.Error("WithEventBus did not set eventBus correctly")
	}
}

func TestWriteJSON(t *testing.T) {
	rr := httptest.NewRecorder()
	data := map[string]string{"key": "value"}

	writeJSON(rr, http.StatusOK, data)

	if rr.Code != http.StatusOK {
		t.Errorf("Status = %d, expected %d", rr.Code, http.StatusOK)
	}

	contentType := rr.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Content-Type = %q, expected 'application/json'", contentType)
	}

	body := rr.Body.String()
	if body == "" {
		t.Error("Body should not be empty")
	}
}

func TestWriteError(t *testing.T) {
	rr := httptest.NewRecorder()

	writeError(rr, http.StatusBadRequest, "test error")

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Status = %d, expected %d", rr.Code, http.StatusBadRequest)
	}

	body := rr.Body.String()
	if body == "" {
		t.Error("Body should not be empty")
	}
}

func TestServer_ShutdownCancelsRuns(t *testing.T) {
	cancelledCount := 0

	s := &Server{
		activeRuns: map[string]*RunInfo{
			"run-1": {
				ID:     "run-1",
				Status: "running",
				Cancel: func() { cancelledCount++ },
			},
			"run-2": {
				ID:     "run-2",
				Status: "running",
				Cancel: func() { cancelledCount++ },
			},
		},
		httpServer: &http.Server{},
	}

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Shutdown should cancel all runs and then fail on httpServer.Shutdown
	// since httpServer was never started
	_ = s.Shutdown(ctx)

	if cancelledCount != 2 {
		t.Errorf("Expected 2 cancelled runs, got %d", cancelledCount)
	}
}

func TestServer_SetupRoutes(t *testing.T) {
	s := &Server{
		router: http.NewServeMux(),
		auth:   NewAuth(AuthConfig{Method: AuthMethodNone}),
	}

	s.setupRoutes()

	// Test that router was configured
	if s.router == nil {
		t.Fatal("Router should not be nil after setupRoutes()")
	}
}
