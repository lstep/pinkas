package sse

import (
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/pinkas/pinkas/internal/auth"
)

// Handler serves the SSE endpoint.
type Handler struct {
	hub    *Hub
	logger *slog.Logger
}

// NewHandler creates a new SSE handler.
func NewHandler(hub *Hub, logger *slog.Logger) *Handler {
	return &Handler{hub: hub, logger: logger}
}

// RegisterRoutes registers the SSE route.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/events", auth.RequireAuth(h.ServeSSE))
}

// ServeSSE handles Server-Sent Events connections.
func (h *Handler) ServeSSE(w http.ResponseWriter, r *http.Request) {
	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*") // TODO: restrict to frontend origin in production

	// Get Last-Event-ID from header or query param
	lastIDStr := r.Header.Get("Last-Event-ID")
	if lastIDStr == "" {
		lastIDStr = r.URL.Query().Get("lastEventId")
	}
	lastID, _ := strconv.ParseInt(lastIDStr, 10, 64)

	// Subscribe to hub
	events, unsub := h.hub.Subscribe(lastID)
	defer unsub()

	// Flush headers immediately
	flusher, ok := w.(http.Flusher)
	if !ok {
		h.logger.Error("streaming unsupported")
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}
	flusher.Flush()

	// Send initial retry directive
	fmt.Fprintf(w, "retry: 5000\n\n")
	flusher.Flush()

	// Stream events
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case ev, ok := <-events:
			if !ok {
				return
			}
			fmt.Fprint(w, FormatSSE(ev))
			flusher.Flush()
		case <-ticker.C:
			// Send keep-alive comment to prevent connection timeout
			fmt.Fprintf(w, ":keepalive\n\n")
			flusher.Flush()
		case <-r.Context().Done():
			return
		}
	}
}
