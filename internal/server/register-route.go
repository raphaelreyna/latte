package server

import (
	"encoding/base64"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"

	"github.com/rs/zerolog/log"
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
			log.Warn().
				Err(err).
				Msg("error parsing json body")

			msg := "error while parsing json body: " + err.Error()
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
			if s.db != nil {
				var datai interface{}
				// If file not found in local disk, check db
				datai, err = s.db.Fetch(r.Context(), req.ID)
				switch err.(type) {
				case *NotFoundError:
					break
				default:
					if err != nil {
						log.Warn().
							Err(err).
							Send()

						s.respond(w, err.Error(), http.StatusInternalServerError)
						return
					} else if datai != nil {
						go func() {
							err = toDisk(datai, fpath)
							if err != nil {
								log.Warn().Err(err).
									Str("path", fpath).
									Msg("error while creating file")

								return
							}
						}()
						w.Header().Set("Content-Type", "application/json")
						s.respond(w, &response{ID: req.ID}, http.StatusConflict)
						return
					}
				}
			}
			// File doesn't exist locally (or in db)
			bytes, err := base64.StdEncoding.DecodeString(req.Data)
			if err != nil {
				log.Warn().
					Err(err).
					Send()

				s.respond(w, err.Error(), http.StatusInternalServerError)
				return
			}
			if err = ioutil.WriteFile(fpath, bytes, os.ModePerm); err != nil {
				log.Warn().
					Err(err).
					Send()

				s.respond(w, err.Error(), http.StatusInternalServerError)
				return
			}

			if s.db != nil {
				if err = s.db.Store(r.Context(), req.ID, bytes); err != nil {
					log.Warn().
						Err(err).
						Send()

					s.respond(w, err.Error(), http.StatusInternalServerError)
					return
				}
			}

			w.Header().Set("Content-Type", "application/json")
			s.respond(w, &response{ID: req.ID}, http.StatusOK)
			return
		}
		s.respond(w, err.Error(), http.StatusInternalServerError)
	}
}
