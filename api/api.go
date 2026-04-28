package api

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/ikem-legend/blockchain-indexer/storage"
)

type Server struct {
	router	*mux.Router
	storage	storage.Storage
	port	string
}

func New(stor storage.Storage, port string) *Server {
	router := mux.NewRouter()
	s := &Server{
		router:		router,
		storage: 	stor,
		port:		port,
	}
	s.SetupRoutes()
	return s
}

func (s *Server) SetupRoutes() {
	s.router.HandleFunc("/", homeHandler).Methods("GET")
	s.router.HandleFunc("/events", s.getEvents).Methods("GET")
	s.router.HandleFunc("/health", s.health).Methods("GET")
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(map[string]string{"message": "Blockchain indexer server"}); err != nil {
		log.Printf("failed to write home response: %v", err)
	}
}

func (s *Server) getEvents(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	events, err := s.storage.GetEvents(ctx, 100)
	if err != nil {
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(events); err != nil {
		log.Printf("failed to encode events response: %v", err)
	}
}

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]string{"status": "ok"}); err != nil {
		log.Printf("failed to encode health response: %v", err)
	}
}

func (s *Server) Start() error {
	srv := &http.Server{
		Addr:              ":" + s.port,
		Handler:           s.router,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       120 * time.Second,
	}
	log.Printf("Starting API server on :%s\n", s.port)
	return srv.ListenAndServe()
}
