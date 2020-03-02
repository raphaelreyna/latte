package server

import (
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"github.com/raphaelreyna/latte/compile"
	"io"
	"io/ioutil"
	"log"
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
		s.infoLog.Printf("Creating new temp directory: %s", workDir)
		if err != nil {
			log.Println(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		j := job{dir: workDir, details: map[string]interface{}{}}
		// Grab any data sent as JSON
		if r.Header.Get("Content-Type") == "application/json" {
			var req request
			err := json.NewDecoder(r.Body).Decode(&req)
			if err != nil {
				log.Println(err)
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
						s.errLog.Println(err)
						http.Error(w, err.Error(), http.StatusInternalServerError)
						return
					}
					t, err = template.New(cid).Delims("#!", "!#").Parse(string(tBytes))
					if err != nil {
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
				tmplPath := filepath.Join(workDir, tmplID)
				var tmplBytes []byte
				_, err = os.Stat(tmplPath)
				if os.IsNotExist(err) {
					rawData, err := s.db.Fetch(tmplID)
					if err != nil {
						s.errLog.Println(err)
						http.Error(w, err.Error(), http.StatusInternalServerError)
						return
					}
					tmplBytes = rawData.([]byte)
					err = ioutil.WriteFile(tmplPath, tmplBytes, os.ModePerm)
					if err != nil {
						s.errLog.Println(err)
						http.Error(w, err.Error(), http.StatusInternalServerError)
						return
					}
				} else if err != nil {
					s.errLog.Println(err)
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				if tmplBytes == nil {
					tmplBytes, err = ioutil.ReadFile(tmplPath)
					if err != nil {
						s.errLog.Println(err)
						http.Error(w, err.Error(), http.StatusInternalServerError)
						return
					}
				}
				t, err = template.New(tmplID).Delims("#!", "!#").Parse(string(tmplBytes))
				if err != nil {
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
			rscs.Lock()
			rscPath, exists := rscs.r[rscID]
			_, err = os.Stat(rscPath)
			if os.IsNotExist(err) == exists {
				s.errLog.Printf(
					`cache integrity error: {"rscID": "%s", "osExists": %v, "cacheExists": %v, "osError": %v}`,
					rscID, os.IsExist(err), exists, err,
				)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			} else if !exists {
				rscData, err := s.db.Fetch(rscID)
				if err != nil {
					s.errLog.Println(err)
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				rscPath = filepath.Join(s.rootDir, rscID)
				err = ioutil.WriteFile(rscPath, rscData.([]byte), os.ModePerm)
				if err != nil {
					s.errLog.Println(err)
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				rscs.r[rscID] = rscPath
				rscs.Unlock()
			}
			err = os.Symlink(rscPath, filepath.Join(workDir, rscID))
			if err != nil {
				s.errLog.Println(err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
		// Load and parse details json from local disk, downloading it from the db if not found on local disk
		if dtID := q.Get("dtls"); j.details == nil && dtID != "" {
			dtlsPath := filepath.Join(workDir, dtID)
			var dtlsBytes []byte
			_, err = os.Stat(dtlsPath)
			if os.IsNotExist(err) {
				dtlsData, err := s.db.Fetch(dtID)
				if err != nil {
					s.errLog.Println(err)
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				dtlsBytes = dtlsData.([]byte)
				ioutil.WriteFile(dtlsPath, dtlsBytes, os.ModePerm)
			} else if err != nil {
				s.errLog.Println(err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			if dtlsBytes == nil {
				dtlsBytes, err = ioutil.ReadFile(dtlsPath)
				if err != nil {
					s.errLog.Println(err)
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
			}
			err = json.Unmarshal(dtlsBytes, j.details)
			if err != nil {
				s.errLog.Println(err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
		// Compile pdf
		pdf, err := compile.Compile(j.tmpl, j.details, j.dir)
		if err != nil {
			log.Println(err)
			http.Error(w, err.Error(), 500)
			return
		}
		w.Header().Set("Content-Type", "application/pdf")
		io.Copy(w, pdf)
		pdf.Close()
		w.WriteHeader(200)
		//go func() {
		//	err = compile.CleanUp(workDir)
		//	if err != nil {
		//		log.Println(err)
		//	}
		//}()
	}
}
