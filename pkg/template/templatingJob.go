package template

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	templatingengine "github.com/raphaelreyna/latte/pkg/template/templating-engine"
)

var perm = os.FileMode(0777)

// Job represents a directory of template and non-template files.
// Template files are parsed and can be executed with a given data structure multiple times.
// Non-template files are symlinked to the output directory.
type Job struct {
	tmplts   map[string]templatingengine.Template
	symlinks map[string]string
}

// Execute executes the templates and symlinks the non-template files to the output directory.
func (j *Job) Execute(outDir string, v interface{}) error {
	if err := j.symlink(outDir); err != nil {
		return fmt.Errorf("error creating symlinks of non-template files: %w", err)
	}

	if err := j.render(outDir, v); err != nil {
		return fmt.Errorf("error rendering templates: %w", err)
	}

	return nil
}

// symlink creates symlinks for non-template files
func (j *Job) symlink(outDir string) error {
	for in, relout := range j.symlinks {
		var (
			out        = filepath.Join(outDir, relout)
			parentPath = filepath.Dir(out)
		)

		if err := os.MkdirAll(parentPath, perm); err != nil {
			return fmt.Errorf("error creating parent directory for symlink: %w", err)
		}

		if err := os.Symlink(in, out); err != nil {
			return fmt.Errorf("error creating symlink: %w", err)
		}
	}

	return nil
}

// render renders the templates with the given data structure into the output directory.
// If v is a json.RawMessage, []byte or string, it is unmarshalled into a map[string]interface{}.
func (tg *Job) render(outDir string, v interface{}) (err error) {
	var vv = v

	switch x := vv.(type) {
	case json.RawMessage:
		var m = make(map[string]interface{})
		if err := json.Unmarshal(x, &m); err != nil {
			return err
		}
		vv = m
	case []byte:
		var m = make(map[string]interface{})
		if err := json.Unmarshal(x, &m); err != nil {
			return err
		}
		vv = m
	case string:
		var m = make(map[string]interface{})
		if err := json.Unmarshal([]byte(x), &m); err != nil {
			return err
		}
		vv = m
	}

	for outPath, tmplt := range tg.tmplts {
		err := execute(filepath.Join(outDir, outPath), tmplt, vv)
		if err != nil {
			return fmt.Errorf("error creating output file for template execution: %w", err)
		}
	}

	return nil
}

func newFile(path string) (*os.File, error) {
	var (
		parentPath = filepath.Dir(path)
	)

	if err := os.MkdirAll(parentPath, perm); err != nil {
		return nil, fmt.Errorf("error creating parent directory for output file: %w", err)
	}

	return os.Create(path)
}

// execute executes a template and writes the output to a file
func execute(path string, tmplt templatingengine.Template, v any) (err error) {
	file, err := newFile(path)
	if err != nil {
		return fmt.Errorf("error creating output file for template execution: %w", err)
	}
	defer func() {
		if e := file.Close(); e != nil {
			err = errors.Join(err, fmt.Errorf("unable to close output file for template execution: %w", e))
		}
	}()

	if err = tmplt.Execute(file, v); err != nil {
		err = fmt.Errorf("error executing template: %w", err)
		return
	}

	return err
}
