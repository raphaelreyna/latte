package compile

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"text/template"
	"io/ioutil"
)

func Compile(ctx context.Context, tmpl *template.Template, dtls map[string]interface{}, opts Options) (string, error) {
	// Move into the working directory
	currDir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	os.Chdir(opts.Dir)
	defer os.Chdir(currDir)

	// Grab a valid compiler from the options
	compiler := string(PDFLatex)
	if cc := opts.CC; cc.IsValid() {
		compiler = string(opts.CC)
	}
	// Create the jobname from the options
	jn := filepath.Base(opts.Dir)
	if opts.N < 1 {
		opts.N = 1
	}

	// Compile however many times the user asked for
	for count := uint(0) ; count < opts.N ; count ++ {
		// Make sure the context hasn't been canceled
		if err := ctx.Err(); err != nil {
			return "", err
		}

		texFile, err := ioutil.TempFile(opts.Dir, "*.tex")
		if err != nil {
			return "", err
		}
		defer func() {
			texFile.Close()
			//os.Remove(texFile.Name())
		}()

		err = tmpl.Execute(texFile, dtls)
		if err != nil {
			return "", err
		}

		args := []string{"-halt-on-error", "-jobname="+jn}
		if opts.CC == Latexmk {
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
