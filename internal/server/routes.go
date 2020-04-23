package server

import (
	"github.com/gorilla/mux"
)

func (s *Server) routes() (*Server, error) {
	// Create and set up http router
	s.router = mux.NewRouter()
	generateRoute, err := s.handleGenerate()
	if err != nil {
		return nil, err
	}
	s.router.HandleFunc("/generate", generateRoute).Methods("POST")
	s.router.HandleFunc("/register", s.handleRegister()).Methods("POST")
	s.router.HandleFunc("/ping", s.handlePing()).Methods("GET")
	return s, nil
}
