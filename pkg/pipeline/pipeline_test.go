package pipeline_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/raphaelreyna/latte/pkg/pipeline"
	"github.com/raphaelreyna/pitch"
)

func TestPipeline(t *testing.T) {
	ctx := context.Background()
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

	outDir, err := os.MkdirTemp("", "test")
	if err != nil {
		t.Fatalf("error creating temporary directory: %v", err)
	}

	config := pipeline.Configuration{
		WorkerCount:    1,
		RenderCount:    1,
		SourceDir:      sourceDir,
		OutDir:         outDir,
		OnMissingKey:   "error",
		RenderFunc:     pipeline.RenderFunc(render),
		PreArchiveHook: nil,
		PreRenderHook:  nil,
		ContextContext: nil,
		FuncMap:        nil,
	}
	p, err := pipeline.NewPipeline(ctx, config)
	if err != nil {
		t.Fatalf("error creating pipeline: %v", err)
	}
	if p == nil {
		t.Fatalf("pipeline is nil")
	}

	job := *pipeline.NewJob(0, []byte(`{"Name": "LaTTe", "Age": 42}`))
	p.Add(&job)
	p.Close()

	err = p.Wait(ctx)
	errs := pipeline.Unwrap(err)
	if len(errs) != 0 {
		t.Fatalf("unexpected error: %v", err)
	}

	err = p.MergeArchives(true)
	if err != nil {
		t.Fatalf("error merging archives: %v", err)
	}

	outFileName := p.MergedArchiveName()
	outFile, err := os.Open(outFileName)
	if err != nil {
		t.Fatalf("error opening output file: %v", err)
	}
	defer outFile.Close()

	stat, err := outFile.Stat()
	if err != nil {
		t.Fatalf("error getting file info: %v", err)
	}
	if stat.Size() == 0 {
		t.Fatalf("output file is empty")
	}

	toc, err := pitch.BuildTableOfContents(outFile)
	if err != nil {
		t.Fatalf("error building table of contents: %v", err)
	}

	if len(toc) != 1 {
		t.Fatalf("unexpected table of contents: %v", toc)
	}
	br, ok := toc["0/render-out/test.pdf"]
	if !ok {
		t.Fatalf("unexpected table of contents: %v", toc)
	}

	if _, err := outFile.Seek(br.Start, 0); err != nil {
		t.Fatalf("error seeking to start of file: %v", err)
	}

	buf := make([]byte, int(br.End-br.Start))
	if _, err := outFile.Read(buf); err != nil {
		t.Fatalf("error reading from output file: %v", err)
	}

	if string(buf) != "PASS" {
		t.Fatalf("unexpected output file contents: %v", string(buf))
	}
}

func render(p *pipeline.Pipeline, job *pipeline.RenderJob) error {
	file, err := os.Create(filepath.Join(job.OutDir, "test.pdf"))
	if err != nil {
		return err
	}
	if _, err := file.WriteString("PASS"); err != nil {
		return err
	}
	return file.Close()
}
