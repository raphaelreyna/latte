package compile

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"text/template"
)

func Compile(ctx context.Context, tmpl *template.Template, dtls map[string]interface{}, dir, command string) (string, error) {
	currDir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	os.Chdir(dir)
	// Prepare pdflatex and grab a pipe to its stdin
	jn := filepath.Base(dir)
	cmd := exec.CommandContext(ctx, command, "-halt-on-error", "-jobname="+jn)
	cmdStdin, err := cmd.StdinPipe()
	if err != nil {
		return "", err
	}
	// Write filled in template to pdflatex stdin
	err = tmpl.Execute(cmdStdin, dtls)
	if err != nil {
		return "", err
	}
	cmdStdin.Close()

	// Run command and grab its output and log it
	result, err := cmd.Output()
	if err != nil {
		return string(result), err
	}
	err = os.Chdir(currDir)
	if err != nil {
		return "", err
	}
	return jn + ".pdf", nil
}
