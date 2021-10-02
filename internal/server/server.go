package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	"github.com/raphaelreyna/latte/internal/job"
	"github.com/rs/zerolog/log"
)

type Server struct {
	router    *mux.Router
	rootDir   string
	db        DB
	cmd       string
	tmplCache *job.TemplateCache
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.router.ServeHTTP(w, r)
}

func (s *Server) respond(w http.ResponseWriter, payload interface{}, code int) []byte {
	w.WriteHeader(code)
	if payload == nil {
		return nil
	}
	switch payload.(type) {
	case string:
		w.Write([]byte(payload.(string)))
		return nil
	case []byte:
		w.Write(payload.([]byte))
		return nil
	case io.ReadCloser:
		_, err := io.Copy(w, payload.(io.ReadCloser))
		if err != nil {
			log.Warn().
				Err(err).
				Send()

			http.Error(w, err.Error(), http.StatusInternalServerError)
			return nil
		}
		return nil
	default:
		payload, err := json.Marshal(payload)
		if err != nil {
			log.Warn().
				Err(err).
				Send()

			http.Error(w, err.Error(), http.StatusInternalServerError)
			return nil
		}
		w.Write(payload)
		return payload
	}
}

func NewServer(root, cmd string, db DB, tCacheSize int) (*Server, error) {
	var err error
	// Ping db to ensure connection
	if db != nil {
		if err = db.Ping(context.Background()); err != nil {
			return nil, fmt.Errorf("error while pinging database: %v", err)
		}
		log.Info().
			Msg("succesfully connected to database")
	}
	s := &Server{
		rootDir: root,
		db:      db,
	}

	// Create the template cache
	s.tmplCache, err = job.NewTemplateCache(tCacheSize)
	if err != nil {
		return nil, err
	}

	// Ensure root directory exists
	if _, err := os.Stat(root); os.IsNotExist(err) {
		if err = os.Mkdir(root, 0755); err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	}
	s.cmd = cmd
	return s.routes(), nil
}
