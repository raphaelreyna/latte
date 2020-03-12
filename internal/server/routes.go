package server

import (
	"github.com/gorilla/mux"
)

func (s *Server) routes() {
	// Create and set up http router
	s.router = mux.NewRouter()
	s.router.HandleFunc("/", s.handleGenerate()).Methods("PUT")
	s.router.HandleFunc("/", s.handleRegister()).Methods("POST")
}
