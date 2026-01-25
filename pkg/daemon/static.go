package daemon

import (
	"io/fs"
	"net/http"
	"strings"

	"github.com/joshua-temple/chronicle/web"
)

// WebFS holds the embedded web UI files.
// It's initialized from web.WebFS with the "dist" subdirectory extracted.
var WebFS fs.FS

func init() {
	// Extract the dist subdirectory from the embedded FS.
	// The web.WebFS embeds files as "dist/*", so we need fs.Sub to serve them at root.
	subFS, err := fs.Sub(web.WebFS, "dist")
	if err != nil {
		// If dist doesn't exist (e.g., not built yet), leave WebFS nil
		// and devModeHandler will be used instead.
		return
	}
	WebFS = subFS
}

// spaHandler wraps a file server to handle SPA routing.
// For any path that doesn't match a real file, it serves index.html.
func spaHandler(fsys fs.FS) http.Handler {
	fileServer := http.FileServer(http.FS(fsys))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/")
		if path == "" {
			path = "index.html"
		}

		// Check if file exists using Stat (avoids opening file descriptor)
		_, err := fs.Stat(fsys, path)
		if err != nil {
			// File not found - serve index.html for SPA routing
			r.URL.Path = "/"
			fileServer.ServeHTTP(w, r)
			return
		}

		// File exists, serve it
		fileServer.ServeHTTP(w, r)
	})
}

// devModeHandler returns a handler for when no embedded files are available.
func devModeHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`<!DOCTYPE html>
<html>
<head><title>Chronicle</title></head>
<body style="font-family: system-ui; max-width: 600px; margin: 50px auto; padding: 20px;">
<h1>Chronicle API Server</h1>
<p>Web UI not embedded. For development:</p>
<pre>cd web && npm run dev</pre>
<p>API available at <a href="/api/v1/health">/api/v1/health</a></p>
</body>
</html>`))
	})
}
