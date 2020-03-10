package server

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"io"
	"log"
	"net/http"
	"os"
)

type Server struct {
	router  *mux.Router
	rootDir string
	db      DB
	cmd     string
	errLog  *log.Logger
	infoLog *log.Logger
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.router.ServeHTTP(w, r)
}

func (s *Server) respond(w http.ResponseWriter, payload interface{}, code int) {
	w.WriteHeader(code)
	if payload == nil {
		return
	}
	switch payload.(type) {
	case string:
		w.Write([]byte(payload.(string)))
		return
	case []byte:
		w.Write(payload.([]byte))
		return
	case io.ReadCloser:
		_, err := io.Copy(w, payload.(io.ReadCloser))
		if err != nil {
			s.errLog.Println(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		return
	default:
		err := json.NewEncoder(w).Encode(payload)
		if err != nil {
			s.errLog.Println(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		return
	}
}

func NewServer(root, cmd string, db DB, err, info *log.Logger) (*Server, error) {
	// Ping db to ensure connection
	if db != nil {
		if err := db.Ping(context.Background()); err != nil {
			return nil, fmt.Errorf("error while pinging database: %v", err)
		}
		info.Println("successfully connected to database")
	}
	s := &Server{rootDir: root, db: db, errLog: err, infoLog: info}
	// Ensure root directory exists
	if _, err := os.Stat(root); os.IsNotExist(err) {
		if err = os.Mkdir(root, 0755); err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	}
	s.cmd = cmd
	s.routes()
	return s, nil
}
