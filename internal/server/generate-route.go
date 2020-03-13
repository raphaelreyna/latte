package server

import (
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/raphaelreyna/latte/internal/compile"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"text/template"
)

func (s *Server) handleGenerate() http.HandlerFunc {
	type request struct {
		// Template is base64 encoded .tex file
		Template string `json:"template"`
		// Details must be a json object
		Details map[string]interface{} `json:"details"`
		// Resources must be a json object whose keys are the resources file names and value is the base64 encoded string of the file
		Resources map[string]string `json:"resources"`
	}
	type job struct {
		tmpl    *template.Template
		details map[string]interface{}
		dir     string
	}
	type templates struct {
		t map[string]*template.Template
		sync.Mutex
	}
	type resources struct {
		r map[string]string
		sync.Mutex
	}
	tmpls := &templates{t: map[string]*template.Template{}}
	rscs := &resources{r: map[string]string{}}
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
		j := job{dir: workDir, details: map[string]interface{}{}}
		// Grab any data sent as JSON
		if r.Header.Get("Content-Type") == "application/json" {
			var req request
			err := json.NewDecoder(r.Body).Decode(&req)
			switch {
			case err == io.EOF:
				s.respond(w, "request header Content-Type set to application/json; received empty body", http.StatusBadRequest)
				return
			case err != nil:
				s.errLog.Println(err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			r.Body.Close()
			if req.Template != "" {
				// Check if we've already parsed this template; if not, parse it and cache the results
				tHash := md5.Sum([]byte(req.Template))
				cid := hex.EncodeToString(tHash[:])
				tmpls.Lock()
				t, exists := tmpls.t[cid]
				if !exists {
					tBytes, err := base64.StdEncoding.DecodeString(req.Template)
					if err != nil {
						tmpls.Unlock()
						s.errLog.Println(err)
						http.Error(w, err.Error(), http.StatusInternalServerError)
						return
					}
					t, err = template.New(cid).Delims("#!", "!#").Parse(string(tBytes))
					if err != nil {
						tmpls.Unlock()
						s.errLog.Println(err)
						http.Error(w, err.Error(), http.StatusInternalServerError)
						return
					}
					tmpls.t[cid] = t
				}
				j.tmpl = t
				tmpls.Unlock()
			}
			// Grab details if they were provided
			if len(req.Details) > 0 {
				j.details = req.Details
			}
			// Write resources files into working directory
			for name, data := range req.Resources {
				fname := filepath.Join(workDir, name)
				bytes, err := base64.StdEncoding.DecodeString(data)
				if err != nil {
					s.errLog.Println(err)
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				err = ioutil.WriteFile(fname, bytes, os.ModePerm)
				if err != nil {
					s.errLog.Println(err)
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
			}
		}
		// Grab any ids sent over the URL
		q := r.URL.Query()
		// Grab template being requested in the URL
		if tmplID := q.Get("tmpl"); j.tmpl == nil && tmplID != "" {
			tmpls.Lock()
			t, exists := tmpls.t[tmplID]
			if !exists {
				// Try loading the template file from local disk, downloading it if it doesn't exist
				tmplPath := filepath.Join(s.rootDir, tmplID)
				var tmplBytes []byte
				_, err := os.Stat(tmplPath)
				if os.IsNotExist(err) {
					if s.db == nil {
						tmpls.Unlock()
						msg := fmt.Sprintf("template with id %s not found", tmplID)
						s.respond(w, msg, http.StatusBadRequest)
						return
					}
					rawData, err := s.db.Fetch(r.Context(), tmplID)
					switch err.(type) {
					case *NotFoundError:
						tmpls.Unlock()
						msg := fmt.Sprintf("template with id %s not found", tmplID)
						http.Error(w, msg, http.StatusInternalServerError)
						return
					default:
						if err != nil {
							tmpls.Unlock()
							s.errLog.Println(err)
							http.Error(w, err.Error(), http.StatusInternalServerError)
							return
						}
					}
					err = toDisk(rawData, tmplPath)
					if err != nil {
						tmpls.Unlock()
						s.errLog.Printf("error while writing to %s: %v", tmplPath, err)
						http.Error(w, err.Error(), http.StatusInternalServerError)
						return
					}
				} else if err != nil {
					tmpls.Unlock()
					s.errLog.Println(err)
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				if tmplBytes == nil {
					tmplBytes, err = ioutil.ReadFile(tmplPath)
					if err != nil {
						tmpls.Unlock()
						s.errLog.Println(err)
						http.Error(w, err.Error(), http.StatusInternalServerError)
						return
					}
				}
				t, err = template.New(tmplID).Delims("#!", "!#").Parse(string(tmplBytes))
				if err != nil {
					tmpls.Unlock()
					s.errLog.Println(err)
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				tmpls.t[tmplID] = t
			}
			j.tmpl = t
			tmpls.Unlock()
		} else if j.tmpl == nil {
			err = errors.New("no template provided")
			s.errLog.Println(err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		// Symlink resources into the working directory, downloading those that aren't in the root directory
		rscsIDs := q["rsc"]
		for _, rscID := range rscsIDs {
			// Prevent other routines from downloading this resource if its not found and we're already downloading it.
			rscs.Lock()
			rscPath, exists := rscs.r[rscID]
			if _, err = os.Stat(rscPath); os.IsNotExist(err) || !exists {
				if s.db == nil {
					rscs.Unlock()
					msg := fmt.Sprintf("resource with id %s not found", rscID)
					s.respond(w, msg, http.StatusBadRequest)
					return
				}
				// If path not in memory, then file doesn't exit on local disk (but lets double check) and we need to download it.
				rscData, err := s.db.Fetch(r.Context(), rscID)
				switch err.(type) {
				case *NotFoundError:
					rscs.Unlock()
					msg := fmt.Sprintf("resource with id %s not found", rscID)
					http.Error(w, msg, http.StatusInternalServerError)
					return
				default:
					if err != nil {
						rscs.Unlock()
						s.errLog.Println(err)
						http.Error(w, err.Error(), http.StatusInternalServerError)
						return
					}
				}
				rscPath = filepath.Join(s.rootDir, rscID)
				err = toDisk(rscData, rscPath)
				if err != nil {
					tmpls.Unlock()
					s.errLog.Printf("error while writing to %s: %v", rscPath, err)
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				rscs.r[rscID] = rscPath
			}
			rscs.Unlock()
			err = os.Symlink(rscPath, filepath.Join(workDir, rscID))
			if err != nil {
				s.errLog.Println(err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
		// Load and parse details json from local disk, downloading it from the db if not found on local disk
		if dtID := q.Get("dtls"); len(j.details) == 0 && dtID != "" {
			dtlsPath := filepath.Join(s.rootDir, dtID)
			_, err = os.Stat(dtlsPath)
			if os.IsNotExist(err) {
				if s.db == nil {
					msg := fmt.Sprintf("details json with id %s not found", dtID)
					s.respond(w, msg, http.StatusBadRequest)
					return
				}
				dtlsData, err := s.db.Fetch(r.Context(), dtID)
				switch err.(type) {
				case *NotFoundError:
					msg := fmt.Sprintf("details json with id %s not found", dtID)
					http.Error(w, msg, http.StatusInternalServerError)
					return
				default:
					if err != nil {
						s.errLog.Println(err)
						http.Error(w, err.Error(), http.StatusInternalServerError)
						return
					}
				}
				err = toDisk(dtlsData, dtlsPath)
				if err != nil {
					s.errLog.Printf("error while writing to %s: %v", dtlsPath, err)
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				switch dtlsData.(type) {
				case []byte:
					err = json.Unmarshal(dtlsData.([]byte), &j.details)
					if err != nil {
						s.errLog.Println(err)
						http.Error(w, err.Error(), http.StatusInternalServerError)
						return
					}
				case io.ReadCloser:
					rc := dtlsData.(io.ReadCloser)
					err = json.NewDecoder(rc).Decode(&j.details)
					if err != nil {
						s.errLog.Println(err)
						http.Error(w, err.Error(), http.StatusInternalServerError)
						return
					}
					rc.Close()
				}
			} else if err != nil {
				s.errLog.Println(err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			if len(j.details) == 0 {
				f, err := os.Open(dtlsPath)
				if err != nil {
					s.errLog.Println(err)
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				err = json.NewDecoder(f).Decode(&j.details)
				if err != nil {
					s.errLog.Println(err)
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				f.Close()
			}
		}
		// Compile pdf
		pdfPath, err := compile.Compile(r.Context(), j.tmpl, j.details, j.dir, s.cmd)
		if err != nil {
			s.errLog.Printf("error while compiling: %v\nCOMPILER OUTPUT: %s", err, pdfPath)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		pdf, err := os.Open(filepath.Join(workDir, pdfPath))
		if err != nil {
			s.errLog.Println(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/pdf")
		io.Copy(w, pdf)
		pdf.Close()
	}
}
