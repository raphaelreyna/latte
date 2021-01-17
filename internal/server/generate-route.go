package server

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"

	"github.com/raphaelreyna/go-recon/sources"
	"github.com/raphaelreyna/latte/internal/job"
)

func (s *Server) handleGenerate() http.HandlerFunc {
	type errorResponse struct {
		Error string `json:"error"`
		Data  string `json:"data,omitempty"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		// Create temporary directory into which we'll copy all of the required resource files
		// and eventually run pdflatex in.
		workDir, err := ioutil.TempDir(s.rootDir, "")
		if err != nil {
			s.errLog.Println(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		s.infoLog.Printf("created new temp directory: %s", workDir)
		defer func() {
			go func() {
				if err = os.RemoveAll(workDir); err != nil {
					s.errLog.Println(err)
				}
			}()
		}()

		// Create a new job for this request
		j := job.NewJob(workDir, sources.NewDirSourceChain(sources.SoftLink, s.rootDir))
		if s.db != nil {
			j.SourceChain = append(j.SourceChain, s.db)
		}

		// Grab any data sent as JSON
		if r.Header.Get("Content-Type") == "application/json" {
			var req job.Request
			defer r.Body.Close()
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				s.errLog.Println(err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			// Grab details if they were provided
			if j, err = req.NewJob(workDir, j.SourceChain, s.tmplCache); err != nil {
				s.errLog.Println(err)
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
		}

		// Check the url quuery values for a registered template, registered details or resources
		// as well as for compilation options and modify the Job accordingly.
		if err = j.ParseQuery(r.URL.Query(), s.tmplCache); err != nil {
			s.errLog.Println(err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Compile pdf
		pdfPath, err := j.Compile(r.Context())
		if err != nil {
			er := &errorResponse{Error: err.Error(), Data: string(pdfPath)}
			w.Header().Set("Content-Type", "application/json")
			s.errLog.Printf("%s", s.respond(w, er, http.StatusInternalServerError))
			return
		}

		// Send the newly rendered PDF to the client
		w.Header().Set("Content-Type", "application/pdf")
		http.ServeFile(w, r, filepath.Join(workDir, pdfPath))
	}
}
