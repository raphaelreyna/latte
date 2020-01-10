package compile

import (
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"text/template"
)

func Compile(tmpl *template.Template, deets *map[string]interface{}, jobname string) (*os.File, error) {
	os.Chdir(jobname)
	// Prepare pdflatex and grab a pipe to its stdin
	jn := "-jobname=" + jobname
	cmd := exec.Command("pdflatex", "-halt-on-error", jn)
	cmdStdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	// Write filled in template to pdflatex stdin
	err = tmpl.Execute(cmdStdin, deets)
	if err != nil {
		return nil, err
	}
	cmdStdin.Close()

	// Run command and grab its output and log it
	result, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	log.Println(string(result))

	// Grab the file and write it to the response
	pdf, err := os.Open(jobname + ".pdf")
	if err != nil {
		return nil, err
	}
	os.Chdir("..")
	return pdf, nil
}

func CleanUp(dir string) error {
	d, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer d.Close()
	names, err := d.Readdirnames(-1)
	if err != nil {
		return err
	}
	for _, name := range names {
		err = os.RemoveAll(filepath.Join(dir, name))
		if err != nil {
			return err
		}
	}
	err = os.Remove(dir)
	if err != nil {
		return err
	}
	return nil
}
