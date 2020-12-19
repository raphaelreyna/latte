package job

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"text/template"

	"github.com/raphaelreyna/go-recon"
)

type Job struct {
	recon.Dir
	Template *template.Template
	Details  map[string]interface{}
	Opts     Options
}

func (j *Job) AddFiles(files ...string) {
	if j.Files == nil {
		j.Files = []*recon.File{}
	}

	for _, resource := range files {
		j.Files = append(j.Files, &recon.File{
			Name: resource,
		})
	}
}

func (j *Job) GetTemplate(id string) error {
	f := recon.File{Name: id}
	_, err := f.AddTo(j.Root, 0644, j.SourceChain)
	if err != nil {
		return err
	}

	name := filepath.Join(j.Root, f.Name)
	//defer os.Remove(name)

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

func (j *Job) CreateWorkDir(root string) (clean func() error, err error) {
	if j.Root, err = ioutil.TempDir(root, ""); err != nil {
		return nil, err
	}

	return func() error {
		return os.RemoveAll(j.Root)
	}, nil
}

// Compile creates a tex file by filling in the template with the details and then compiles
// the results and returns the location of the resulting PDF.
func (j *Job) Compile(ctx context.Context) (string, error) {
	opts := j.Opts
	// Move into the working directory
	currDir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	os.Chdir(j.Root)
	defer os.Chdir(currDir)

	// Grab a valid compiler from the options
	compiler := string(CC_Default)
	if cc := opts.CC; cc.IsValid() {
		compiler = string(opts.CC)
	}
	// Create the jobname from the options
	jn := filepath.Base(j.Root)
	if opts.N < 1 {
		opts.N = 1
	}

	// Create the tex file
	texFile, err := ioutil.TempFile(j.Root, "*_filled-in.tex")
	if err != nil {
		return "", err
	}
	defer func() {
		texFile.Close()
		//os.Remove(texFile.Name())
	}()

	err = j.Template.Execute(texFile, j.Details)
	if err != nil {
		return "", err
	}

	// Compile however many times the user asked for
	for count := uint(0); count < opts.N; count++ {
		// Make sure the context hasn't been canceled
		if err := ctx.Err(); err != nil {
			return "", err
		}

		args := []string{"-halt-on-error", "-jobname=" + jn}
		if opts.CC == CC_Latexmk {
			args = append(args, "-pdf")
		}
		args = append(args, texFile.Name())
		// Create a handle for the compiler command
		cmd := exec.CommandContext(ctx, compiler, args...)

		switch count {
		case opts.N - 1: // capture the error on the last run
			// Run command and grab its output and log it
			result, err := cmd.Output()
			if err != nil {
				return string(result), err
			}
		default:
			if err = cmd.Start(); err != nil {
				return "", err
			}
			if err = cmd.Wait(); err != nil {
				return "", err
			}
		}
	}
	return jn + ".pdf", nil
}
