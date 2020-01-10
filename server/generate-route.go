package server

import (
	"encoding/base64"
	"encoding/json"
	"github.com/google/uuid"
	"github.com/raphaelreyna/latte/compile"
	"io"
	"log"
	"net/http"
	"text/template"
)

func HandleGenerate(w http.ResponseWriter, r *http.Request) {
	// Define request struct
	reqStruct := struct {
		// Template is base64 encoded .tex file
		Template string `json:"template"`
		// Details must be a json object
		Details map[string]interface{} `json:"details"`
		// Resources must be a json object whose keys are the resources file names and value is the base64 encoded string of the file
		Resources map[string]string `json:"resources"`
	}{}
	// Grab JSON and parse it
	defer r.Body.Close()
	err := json.NewDecoder(r.Body).Decode(&reqStruct)
	if err != nil {
		log.Println(err)
		http.Error(w, err.Error(), 500)
		return
	}
	// Save resources
	resources := reqStruct.Resources
	uid := uuid.New().String()
	if len(resources) > 0 {
		err = SaveEnvToDisk(uid, resources)
		if err != nil {
			log.Println(err)
			http.Error(w, err.Error(), 500)
			return
		}
	}
	// Grab details map
	deets := reqStruct.Details
	// Grab template bytes
	tmplBytes, err := base64.StdEncoding.DecodeString(reqStruct.Template)
	if err != nil {
		log.Println(err)
		http.Error(w, err.Error(), 500)
		return
	}
	// Create template struct
	tmpl, err := template.New(uid).Delims("#!", "!#").Parse(string(tmplBytes))
	if err != nil {
		log.Println(err)
		http.Error(w, err.Error(), 500)
		return
	}

	pdf, err := compile.Compile(tmpl, &deets, uid)
	if err != nil {
		log.Println(err)
		http.Error(w, err.Error(), 500)
		return
	}
	defer pdf.Close()
	err = compile.CleanUp(uid)
	if err != nil {
		log.Println(err)
	}
	w.Header().Set("Content-Type", "application/pdf")
	io.Copy(w, pdf)
	w.WriteHeader(200)
}
