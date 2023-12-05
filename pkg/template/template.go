package template

import (
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"text/template"

	templatingengine "github.com/raphaelreyna/latte/pkg/template/templating-engine"
)

var extRe = regexp.MustCompile(`^(.*\.tex).*$`)

func SetFileExtension(ext string) error {
	r, err := regexp.Compile("^(.*\\." + ext + ").*$")
	if err != nil {
		return err
	}

	extRe = r

	return nil
}

type FuncMap template.FuncMap

type JobConfig struct {
	Dir               string
	TemplatingEngine  templatingengine.TemplatingEngine
	MissingKeyHandler templatingengine.MissingKeyHandler
	FuncMap           FuncMap
}

func NewJob(conf *JobConfig) (*Job, error) {
	var (
		filesys   = os.DirFS(conf.Dir)
		fileGlobs = map[string]string{}
		symlinks  = map[string]string{}
	)

	// determine which files are templates and which are not
	err := fs.WalkDir(filesys, ".", walkInputDirFn(conf.Dir, fileGlobs, symlinks))
	if err != nil {
		return nil, fmt.Errorf("error walking input directory: %w", err)
	}

	// build up templates
	tmplts := map[string]templatingengine.Template{}
	for glob, name := range fileGlobs {
		tmplts[name], err = newTemplate(conf, glob, name)
		if err != nil {
			return nil, fmt.Errorf("error creating template: %w", err)
		}
	}

	return &Job{
		tmplts:   tmplts,
		symlinks: symlinks,
	}, nil
}

func walkInputDirFn(inDir string, fileGlobs, symlinks map[string]string) func(path string, d fs.DirEntry, err error) error {
	return func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("error while walking input directory: %w", err)
		}

		if d.IsDir() {
			return nil
		}

		var parts = extRe.FindStringSubmatch(d.Name())
		if len(parts) == 0 {
			// is non-template file, add it to the symlink pile
			var (
				parentDirName = filepath.Dir(path)
				inFileDir     = filepath.Join(inDir, parentDirName)
				inFilePath    = filepath.Join(inFileDir, d.Name())
			)

			symlinks[inFilePath] = path

			return nil
		}

		var (
			outFileDir  = filepath.Dir(path)
			outFilePath = filepath.Join(outFileDir, parts[1])
			outFileGlob = outFilePath + "*"
		)

		fileGlobs[outFileGlob] = outFilePath

		return nil
	}
}

func newTemplate(conf *JobConfig, glob, name string) (templatingengine.Template, error) {
	files, err := fs.Glob(os.DirFS(conf.Dir), glob)
	if err != nil {
		return nil, fmt.Errorf("error searching for files: %w", err)
	}
	if len(files) == 0 {
		return nil, errors.New("no files found")
	}

	for idx := range files {
		files[idx] = filepath.Join(conf.Dir, files[idx])
	}

	tmplt, err := conf.TemplatingEngine.NewTemplate(filepath.Base(files[0]), conf.MissingKeyHandler, template.FuncMap(conf.FuncMap))
	if err != nil {
		return nil, fmt.Errorf("error creating template: %w", err)
	}

	tmplt, err = tmplt.ParseFiles(files...)
	if err != nil {
		log.Printf(
			"error parsing template files: %s",
			err.Error(),
		)

		return nil, err
	}

	return tmplt, nil
}
