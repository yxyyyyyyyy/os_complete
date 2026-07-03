package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"aort-r/internal/config"
	"aort-r/internal/events"
)

func NewServer(cfg config.Config) http.Handler {
	return NewServerWithEvents(cfg, events.NewHub(32))
}

func NewServerWithEvents(cfg config.Config, hub *events.Hub) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{
			"status": "ok",
			"mode":   cfg.Mode,
		})
	})
	mux.HandleFunc("/api/events", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		streamEvents(w, r, hub)
	})
	return mux
}

func streamEvents(w http.ResponseWriter, r *http.Request, hub *events.Hub) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	ch, cancel := hub.Subscribe()
	defer cancel()
	flusher, _ := w.(http.Flusher)
	for {
		select {
		case <-r.Context().Done():
			return
		case event, ok := <-ch:
			if !ok {
				return
			}
			data, err := json.Marshal(event)
			if err != nil {
				continue
			}
			_, _ = fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event.Type, data)
			if flusher != nil {
				flusher.Flush()
			}
		}
	}
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
