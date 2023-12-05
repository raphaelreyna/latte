package template_test

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/raphaelreyna/latte/pkg/template"
	templatingengine "github.com/raphaelreyna/latte/pkg/template/templating-engine"
	texttemplate "github.com/raphaelreyna/latte/pkg/template/templating-engine/text/template"
)

func TestTemplatingJob(t *testing.T) {
	// create a temporary directory to hold test files
	sourceDir, err := os.MkdirTemp("", "test")
	if err != nil {
		t.Fatalf("error creating temporary directory: %v", err)
	}
	defer os.RemoveAll(sourceDir)

	// create some test template files
	err = os.WriteFile(filepath.Join(sourceDir, "test1.tex"), []byte("|@ .Name @|"), 0644)
	if err != nil {
		t.Fatalf("error creating test file: %v", err)
	}
	err = os.WriteFile(filepath.Join(sourceDir, "test2.tex"), []byte("|@ .Age @|"), 0644)
	if err != nil {
		t.Fatalf("error creating test file: %v", err)
	}

	// create some test non-template files
	err = os.WriteFile(filepath.Join(sourceDir, "test1.txt"), []byte("test1"), 0644)
	if err != nil {
		t.Fatalf("error creating test file: %v", err)
	}

	// create a new template group config
	conf := &template.JobConfig{
		Dir:               sourceDir,
		TemplatingEngine:  texttemplate.NewTemplatingEngine(),
		MissingKeyHandler: templatingengine.MissingKeyHandler_Error,
	}

	// create a new templatingJob
	tj, err := template.NewJob(conf)
	if err != nil {
		t.Fatalf("error creating template group: %v", err)
	}

	outDir, err := os.MkdirTemp("", "test")
	if err != nil {
		t.Fatalf("error creating temporary directory: %v", err)
	}

	err = tj.Execute(outDir, map[string]any{
		"Name": "LaTTe",
		"Age":  42,
	})
	if err != nil {
		t.Fatalf("error executing template group: %v", err)
	}

	outDirContents, err := os.ReadDir(outDir)
	if err != nil {
		t.Fatalf("error reading output directory: %v", err)
	}

	contentsMap := map[string]fs.DirEntry{}
	for _, f := range outDirContents {
		contentsMap[f.Name()] = f
	}

	if info, ok := contentsMap["test1.tex"]; !ok {
		t.Fatalf("expected output directory to contain file %q", "test1")
	} else {
		content, err := os.ReadFile(filepath.Join(outDir, info.Name()))
		if err != nil {
			t.Fatalf("error reading file %q: %v", info.Name(), err)
		}
		if string(content) != "LaTTe" {
			t.Fatalf("expected file %q to contain %q, got %q", info.Name(), "LaTTe", string(content))
		}
	}

	if _, ok := contentsMap["test2.tex"]; !ok {
		t.Fatalf("expected output directory to contain file %q", "test2")
	} else {
		content, err := os.ReadFile(filepath.Join(outDir, "test2.tex"))
		if err != nil {
			t.Fatalf("error reading file %q: %v", "test2.tex", err)
		}
		if string(content) != "42" {
			t.Fatalf("expected file %q to contain %q, got %q", "test2.tex", "42", string(content))
		}
	}

	if _, ok := contentsMap["test1.txt"]; !ok {
		t.Fatalf("expected output directory to contain file %q", "test1.txt")
		content, err := os.ReadFile(filepath.Join(outDir, "test1.txt"))
		if err != nil {
			t.Fatalf("error reading file %q: %v", "test1.txt", err)
		}
		if string(content) != "test1" {
			t.Fatalf("expected file %q to contain %q, got %q", "test1.txt", "test1", string(content))
		}
	}
}
