package server

import (
	"encoding/base64"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
)

func (s *Server) handleRegister() http.HandlerFunc {
	type request struct {
		ID   string `json:"id"`
		Data string `json:"data"`
	}
	type response struct {
		ID string `json:"id"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		var req request
		var err error
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			msg := "error while parsing json body: " + err.Error()
			s.errLog.Println(msg)
			s.respond(w, msg, http.StatusInternalServerError)
			return
		}
		r.Body.Close()

		fpath := filepath.Join(s.rootDir, req.ID)
		if _, err = os.Stat(fpath); err == nil {
			w.Header().Set("Content-Type", "application/json")
			s.respond(w, &response{ID: req.ID}, http.StatusConflict)
			return
		} else if os.IsNotExist(err) {
			var datai interface{}
			// If file not found in local disk, check db
			if datai, err = s.db.Fetch(r.Context(), req.ID); err != nil {
				s.errLog.Println(err)
				s.respond(w, err.Error(), http.StatusInternalServerError)
				return
			} else if datai != nil {
				go func() {
					// Save file to local disk for next time
					bytes := datai.([]byte)
					if err = ioutil.WriteFile(fpath, bytes, os.ModePerm); err != nil {
						s.errLog.Println(err)
					}
					s.infoLog.Printf("saved file from database to local disk: %s", req.ID)
				}()
				w.Header().Set("Content-Type", "application/json")
				s.respond(w, &response{ID: req.ID}, http.StatusConflict)
				return
			}
			// File doesn't exist locally or in db
			bytes, err := base64.StdEncoding.DecodeString(req.Data)
			if err != nil {
				s.errLog.Println(err)
				s.respond(w, err.Error(), http.StatusInternalServerError)
				return
			}
			if err = ioutil.WriteFile(fpath, bytes, os.ModePerm); err != nil {
				s.errLog.Println(err)
				s.respond(w, err.Error(), http.StatusInternalServerError)
				return
			}
			s.infoLog.Printf("wrote new file to local disk: %s", req.ID)
			if err = s.db.Store(r.Context(), req.ID, bytes); err != nil {
				s.errLog.Println(err)
				s.respond(w, err.Error(), http.StatusInternalServerError)
				return
			}
			s.infoLog.Printf("sent new file to database; successfully completed registration: %s", req.ID)
			w.Header().Set("Content-Type", "application/json")
			s.respond(w, &response{ID: req.ID}, http.StatusOK)
			return
		}
		s.respond(w, err.Error(), http.StatusInternalServerError)
	}
}
