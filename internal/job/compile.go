package job

import (
	"context"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
)

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

	tmpl := j.Template.Option("missingkey=" + j.Opts.OnMissingKey.Val())
	err = tmpl.Execute(texFile, j.Details)
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
