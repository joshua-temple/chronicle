package daemon

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
)

func TestSpaHandler_ServesIndexForRoot(t *testing.T) {
	mockFS := fstest.MapFS{
		"index.html": &fstest.MapFile{
			Data: []byte("<html>Test</html>"),
		},
	}

	handler := spaHandler(mockFS)
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	body := w.Body.String()
	if !strings.Contains(body, "<html>Test</html>") {
		t.Errorf("expected index.html content, got %s", body)
	}
}

func TestSpaHandler_ServesStaticFiles(t *testing.T) {
	mockFS := fstest.MapFS{
		"index.html": &fstest.MapFile{
			Data: []byte("<html>Index</html>"),
		},
		"assets/main.js": &fstest.MapFile{
			Data: []byte("console.log('test')"),
		},
	}

	handler := spaHandler(mockFS)
	req := httptest.NewRequest("GET", "/assets/main.js", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	body := w.Body.String()
	if !strings.Contains(body, "console.log") {
		t.Errorf("expected JS content, got %s", body)
	}
}

func TestSpaHandler_FallsBackToIndexForUnknownPaths(t *testing.T) {
	mockFS := fstest.MapFS{
		"index.html": &fstest.MapFile{
			Data: []byte("<html>SPA</html>"),
		},
	}

	handler := spaHandler(mockFS)
	// Request a path that doesn't exist - should serve index.html for SPA routing
	req := httptest.NewRequest("GET", "/some/spa/route", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	body := w.Body.String()
	if !strings.Contains(body, "<html>SPA</html>") {
		t.Errorf("expected index.html content for SPA route, got %s", body)
	}
}

func TestDevModeHandler_ReturnsHTML(t *testing.T) {
	handler := devModeHandler()
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "text/html" {
		t.Errorf("expected Content-Type text/html, got %s", contentType)
	}

	body := w.Body.String()
	if !strings.Contains(body, "Chronicle API Server") {
		t.Errorf("expected dev mode message, got %s", body)
	}
	if !strings.Contains(body, "npm run dev") {
		t.Errorf("expected npm run dev instruction, got %s", body)
	}
}

func TestWebFS_InitializedWhenDistExists(t *testing.T) {
	// WebFS should be initialized from embedded files when web/dist exists.
	// The init() function in static.go extracts the "dist" subdirectory from web.WebFS.
	// If the dist directory exists (i.e., web frontend was built), WebFS will be non-nil.
	// If dist doesn't exist, WebFS will be nil and devModeHandler is used instead.
	//
	// This test verifies the current state - when dist exists, WebFS should be set.
	if WebFS == nil {
		t.Skip("WebFS is nil - web/dist may not be built (run 'make web-build')")
	}

	// Verify we can read files from the embedded FS
	_, err := WebFS.Open("index.html")
	if err != nil {
		t.Errorf("expected to find index.html in embedded FS: %v", err)
	}
}
