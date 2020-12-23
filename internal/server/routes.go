package server

import (
	"net/http"

	"github.com/gorilla/mux"
)

func (s *Server) routes() *Server {
	// Create and set up http router
	s.router = mux.NewRouter()
	s.router.HandleFunc("/generate", s.handleGenerate()).Methods("POST")
	s.router.HandleFunc("/register", s.handleRegister()).Methods("POST")
	s.router.HandleFunc("/ping", s.handlePing()).Methods("GET")
	return s
}

func (s *Server) handlePing() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("PONG"))
	}
}
