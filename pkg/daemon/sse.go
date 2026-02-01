package daemon

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// handleEvents handles Server-Sent Events connections for real-time updates.
func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Get flusher for streaming
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "SSE not supported", http.StatusInternalServerError)
		return
	}

	// Create event channel with buffer to prevent blocking
	eventCh := make(chan Event, 100)

	// Subscribe to events
	unsubscribe := s.eventBus.Subscribe(func(e Event) {
		select {
		case eventCh <- e:
		default:
			// Channel full, skip event to prevent blocking
		}
	})
	defer unsubscribe()

	// Send initial connection event
	_, _ = fmt.Fprintf(w, "event: connected\ndata: {\"status\":\"connected\"}\n\n")
	flusher.Flush()

	// Stream events until client disconnects
	for {
		select {
		case <-r.Context().Done():
			return
		case event := <-eventCh:
			data, err := json.Marshal(map[string]any{
				"type":      string(event.Type),
				"timestamp": event.Timestamp,
				"data":      event.Data,
			})
			if err != nil {
				continue
			}
			_, _ = fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event.Type, data)
			flusher.Flush()
		}
	}
}
