package server

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"telegram-manager/internal/tg"
)

// Server holds dependencies for the HTTP server
type Server struct {
	tgClient *tg.Client
}

// NewServer creates a new HTTP server
func NewServer(tgClient *tg.Client) *Server {
	return &Server{
		tgClient: tgClient,
	}
}

// Start starts the HTTP server on the given port
func (s *Server) Start(ctx context.Context, port string) error {
	mux := http.NewServeMux()
	mux.Handle("/", http.FileServer(http.Dir("./static")))
	mux.HandleFunc("/api/messages", s.handleGetMessages)
	mux.HandleFunc("/api/delete", s.handleDeleteMessages)
	mux.HandleFunc("/api/media", s.handleGetMedia)

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	// Create a channel to catch server start errors
	serverError := make(chan error, 1)

	go func() {
		log.Printf("Server starting on http://localhost:%s", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serverError <- err
		}
	}()

	// Wait for context cancellation or server error
	select {
	case <-ctx.Done():
		log.Println("Shutting down HTTP server...")
		return srv.Shutdown(context.Background())
	case err := <-serverError:
		return err
	}
}

func (s *Server) handleGetMedia(w http.ResponseWriter, r *http.Request) {
	// ... existing ...
	// Just ensure it's kept or I can just use existing logic if I didn't verify lines match perfectly.
	// I will replace handleGetMedia just to be safe if I'm replacing the block including it.
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		http.Error(w, "id required", http.StatusBadRequest)
		return
	}

	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	// Log media access? Maybe too verbose. User asked for activity log.
	// "Downloading media for ID ..."
	log.Printf("Activity: Fetching media for message %d", id)

	data, contentType, err := s.tgClient.GetMessageMedia(r.Context(), id)
	if err != nil {
		log.Printf("Error fetching media for %d: %v", id, err)
		http.Error(w, "Failed to get media", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", contentType)
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

func (s *Server) handleGetMessages(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	offsetIDStr := r.URL.Query().Get("offset_id")
	limitStr := r.URL.Query().Get("limit")

	offsetID := 0
	if offsetIDStr != "" {
		var err error
		offsetID, err = strconv.Atoi(offsetIDStr)
		if err != nil {
			http.Error(w, "Invalid offset_id", http.StatusBadRequest)
			return
		}
	}

	limit := 20
	if limitStr != "" {
		var err error
		limit, err = strconv.Atoi(limitStr)
		if err != nil {
			http.Error(w, "Invalid limit", http.StatusBadRequest)
			return
		}
	}

	log.Printf("Activity: Fetching messages (Limit: %d, Offset: %d)", limit, offsetID)

	messages, total, err := s.tgClient.GetSavedMessages(r.Context(), offsetID, limit)
	if err != nil {
		log.Printf("Error fetching messages: %v", err)
		http.Error(w, "Failed to fetch messages", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"messages": messages,
		"total":    total,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

type DeleteRequest struct {
	IDs []int `json:"ids"`
}

func (s *Server) handleDeleteMessages(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req DeleteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := s.tgClient.DeleteMessages(r.Context(), req.IDs); err != nil {
		log.Printf("Error deleting messages: %v", err)
		http.Error(w, "Failed to delete messages", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
