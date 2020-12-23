package job

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"text/template"

	"github.com/raphaelreyna/go-recon"
	"errors"
)

// Job represents the request for constructing a tex file from the template and details and then compiling that into a PDF.
type Job struct {
	recon.Dir
	Template *template.Template
	Details  map[string]interface{}
	Opts     Options
}

func NewJob(root string, sc recon.SourceChain) *Job {
	j := &Job{Opts: DefaultOptions}
	j.Root = root
	j.SourceChain = sc

	return j
}

// AddResource adds resource files to the Job that should be available in the root/working directory.
func (j *Job) AddResource(files ...string) {
	if j.Files == nil {
		j.Files = []*recon.File{}
	}

	for _, resource := range files {
		j.Files = append(j.Files, &recon.File{
			Name: resource,
		})
	}
}

// GetTemplate looks for a template named id in the SourceChain and parses it, storing the results for later use.
func (j *Job) GetTemplate(id string) error {
	// Make sure the delimiters aren't empty
	if j.Opts.Delims == BadDefaultDelimiters || j.Opts.Delims == EmptyDelimiters {
		return errors.New("invalid delimiters, cannot parse template")
	}
	f := recon.File{Name: id}
	_, err := f.AddTo(j.Root, 0644, j.SourceChain)
	if err != nil {
		return err
	}

	name := filepath.Join(j.Root, f.Name)

	data, err := ioutil.ReadFile(name)
	if err != nil {
		return err
	}

	t := template.New(id)
	t = t.Delims(j.Opts.Delims.Left, j.Opts.Delims.Right)
	t, err = t.Parse(string(data))
	if err != nil {
		return err
	}

	j.Template = t
	return nil
}

// GetDetails looks for a details file named id in the SourceChain and stores the results for later.
func (j *Job) GetDetails(id string) error {
	f := recon.File{Name: id}
	_, err := f.AddTo(j.Root, 0644, j.SourceChain)
	if err != nil {
		return err
	}

	name := filepath.Join(j.Root, f.Name)
	defer os.Remove(name)

	data, err := ioutil.ReadFile(name)
	if err != nil {
		return err
	}

	dtls := map[string]interface{}{}
	if err = json.Unmarshal(data, &dtls); err != nil {
		return err
	}

	j.Details = dtls
	return nil
}
