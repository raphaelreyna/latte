package server

import (
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"text/template"

	lru "github.com/hashicorp/golang-lru"
	"github.com/raphaelreyna/go-recon/sources"
	"github.com/raphaelreyna/latte/internal/job"
)

func (s *Server) handleGenerate() (http.HandlerFunc, error) {
	type request struct {
		// Template is base64 encoded .tex file
		Template string `json:"template"`
		// Details must be a json object
		Details map[string]interface{} `json:"details"`
		// Resources must be a json object whose keys are the resources file names and value is the base64 encoded string of the file
		Resources  map[string]string `json:"resources"`
		Delimiters job.Delimiters    `json:"delimiters, omitempty"`
		// OnMissingKey valid values: 'error', 'zero', 'nothing'
		OnMissingKey string `json:"onMissingKey"`
		// Compiler valid values: 'pdflatex', 'latexmk'
		Compiler job.Compiler `json:"compiler"`
		// Count valid values: > 0
		Count uint `json:"count"`
	}

	type errorResponse struct {
		Error string `json:"error"`
		Data  string `json:"data,omitempty"`
	}

	// Cache parsed templates
	tmplsCache, err := lru.New(s.tCacheSize)
	if err != nil {
		return nil, err
	}
	tmplsCacheMu := sync.Mutex{}
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
				/*if err = os.RemoveAll(workDir); err != nil {
					s.errLog.Println(err)
				}*/
			}()
		}()

		sc := sources.NewDirSourceChain(sources.SoftLink, s.rootDir)
		if s.db != nil {
			sc = append(sc, s.db)
		}

		// Create a new job for this request
		j := job.Job{Opts: job.DefaultOptions}
		j.Root = workDir
		var tmplOption string
		cOpts := j.Opts

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
			if req.Delimiters.Left == "" || req.Delimiters.Right == "" {
				s.respond(w, "only received one delimiter; need none or both", http.StatusBadRequest)
				return
			} else if req.Delimiters.Left != "" && req.Delimiters.Right != "" {
				cOpts.Delims = req.Delimiters
			}
			if req.Template != "" {
				// Check if we've already parsed this template; if not, parse it and cache the results
				tHash := md5.Sum([]byte(req.Template))
				// We append template delimiters to account for the same file being uploaded with different delimiters.
				// This would really only happen on accident but not taking it into account leads to unexpected caching behavior.
				cid := hex.EncodeToString(tHash[:]) + cOpts.Delims.Left + cOpts.Delims.Right
				tmplsCacheMu.Lock()
				ti, exists := tmplsCache.Get(cid)
				var t *template.Template
				if !exists {
					tBytes, err := base64.StdEncoding.DecodeString(req.Template)
					if err != nil {
						tmplsCacheMu.Unlock()
						s.errLog.Println(err)
						http.Error(w, err.Error(), http.StatusInternalServerError)
						return
					}
					t = template.New(cid).Delims(cOpts.Delims.Left, cOpts.Delims.Right)
					t, err = t.Parse(string(tBytes))
					if err != nil {
						tmplsCacheMu.Unlock()
						s.errLog.Println(err)
						http.Error(w, err.Error(), http.StatusInternalServerError)
						return
					}

					tmplsCache.Add(cid, t)
				} else {
					t = ti.(*template.Template)
				}
				j.Template = t
				tmplsCacheMu.Unlock()
			}
			// Grab details if they were provided
			if len(req.Details) > 0 {
				j.Details = req.Details
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

			// Grab options if they were provided
			switch omk := req.OnMissingKey; omk {
			case "error":
				fallthrough
			case "zero":
				tmplOption = "missingkey=" + omk
			case "nothing":
				tmplOption = "missingkey=default"
			case "":
				break
			default:
				s.infoLog.Printf("received invalid onMissingKey field found in JSON body: %s\n", omk)
				http.Error(w, "invalid onMissingKey field found in JSON body", http.StatusBadRequest)
				return
			}

			cOpts.CC = req.Compiler
			cOpts.N = req.Count
		}

		// *************************************************************
		// Check the URL for template, details or resource IDs.
		// These are used to symlink any previoulsly registered files
		// into the working directory for this render/generate request.
		// *************************************************************
		q := r.URL.Query()
		// Check if a registered template is being requested in the URL, if so make sure its available on the local disk
		if tmplID := q.Get("tmpl"); j.Template == nil && tmplID != "" {
			tmplID = tmplID + cOpts.Delims.Left + cOpts.Delims.Right
			tmplsCacheMu.Lock()

			ti, exists := tmplsCache.Get(tmplID)
			if !exists {
				// Look for the requested template in the source chain and parse it
				if err := j.GetTemplate(tmplID); err != nil {
					tmplsCacheMu.Unlock()
					s.errLog.Println(err)
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}

				tmplsCache.Add(tmplID, j.Template)
			} else {
				j.Template = ti.(*template.Template)
			}
			tmplsCacheMu.Unlock()
		} else if j.Template == nil {
			err = errors.New("no template provided")
			s.errLog.Println(err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		// Finish setting up the template
		if tmplOption == "" {
			switch omk := q.Get("onMissingKey"); omk {
			case "":
				fallthrough
			case "error":
			case "zero":
				tmplOption = "missingkey=" + omk
				j.Template = j.Template.Option(tmplOption)
			case "nothing":
				tmplOption = "missingkey=default"
				j.Template = j.Template.Option(tmplOption)
			default:
				s.infoLog.Printf("received invalid onMissingKey field found in JSON body: %s\n", omk)
				http.Error(w, "invalid onMissingKey field found in JSON body", http.StatusBadRequest)
				return
			}
		}

		// handle linking resources into the working directory, downloading those that aren't in the root directory
		rscsIDs := q["rsc"]
		j.AddFiles(rscsIDs...)

		// Load and parse details json from local disk, downloading it from the db if not found on local disk
		if dtID := q.Get("dtls"); len(j.Details) == 0 && dtID != "" {
			if err := j.GetDetails(dtID); err != nil {
				s.infoLog.Printf("error getting details for template: %s\n", err.Error())
				http.Error(w, "error getting saved details: "+err.Error(), http.StatusBadRequest)
				return
			}
		}

		// finish configuring compilation options
		if cOpts.CC == "" {
			cOpts.CC = job.Compiler(q.Get("compiler"))
		}
		if cOpts.N < 2 {
			if n, err := strconv.Atoi(q.Get("count")); err == nil {
				cOpts.N = uint(n)
			}
		}
		// Apply all configured options
		j.Opts = cOpts

		// Compile pdf
		pdfPath, err := j.Compile(r.Context())
		if err != nil {
			er := &errorResponse{Error: err.Error(), Data: string(pdfPath)}
			w.Header().Set("Content-Type", "application/json")
			payload := s.respond(w, er, http.StatusInternalServerError)
			s.errLog.Printf("%s", payload)
			return
		}

		// Open the newly generated PDF and send it to the client
		pdf, err := os.Open(filepath.Join(workDir, pdfPath))
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			_ = s.respond(w, &errorResponse{Error: "encountered an error"}, http.StatusInternalServerError)
			s.errLog.Printf("error opening pdf: %s", err.Error())
			return
		}
		w.Header().Set("Content-Type", "application/pdf")
		io.Copy(w, pdf)
		pdf.Close()
	}, nil
}
