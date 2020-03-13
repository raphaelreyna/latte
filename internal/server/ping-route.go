package server

import (
	"net/http"
)

func (s *Server) handlePing() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("PONG"))
	}
}
