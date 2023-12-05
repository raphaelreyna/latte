package core_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/url"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/raphaelreyna/latte/pkg/core"
	"github.com/raphaelreyna/latte/pkg/frontend"
	"github.com/raphaelreyna/latte/pkg/pipeline"
	"github.com/raphaelreyna/latte/pkg/storage"
	"github.com/raphaelreyna/latte/pkg/test"
)

func TestStart(t *testing.T) {
	ctx := context.Background()
	provider1 := test.MockProvider{
		Data: make(map[string]*test.MockWriteCloser),
	}
	storage := storage.Storage{}
	storage.RegisterStorageProvider(&provider1, "mock1")

	dirData := map[string][]byte{
		"test1":    []byte("test1"),
		"main.tex": []byte(`|@ .Name @|`),
	}
	archiveData, err := test.ArchiveAsPitch(dirData)
	if err != nil {
		t.Fatal(err)
	}

	err = storage.StoreBytes(ctx, archiveData, &url.URL{
		Scheme: "mock1",
		Host:   "testing",
		Path:   "/test",
	})
	if err != nil {
		t.Fatal(err)
	}

	sourceDir, err := os.MkdirTemp("", "latte-test")
	if err != nil {
		t.Fatal(err)
	}
	sharedDir, err := os.MkdirTemp("", "latte-test")
	if err != nil {
		t.Fatal(err)
	}

	wg := sync.WaitGroup{}
	jds := make([]*frontend.JobDone, 0)

	requests := []*frontend.Request{
		{
			Job: &frontend.Job{
				ID:        "test1",
				TargetURI: "mock1://testing/test1-output",
				SourceURI: "mock1://testing/test",
				Contexts: []json.RawMessage{
					[]byte(`{"Name":"0"}`),
				},
			},
			Context: func() context.Context {
				return ctx
			},
			Done: func(jd *frontend.JobDone) {
				jds = append(jds, jd)
				wg.Done()
			},
		},
	}
	wg.Add(len(requests))

	ti := test.MockIngress{
		RequestsQueue: requests,
	}

	ppHookCallCount := 0
	config := core.Config{
		SourceDir:  sourceDir,
		SharedDir:  sharedDir,
		Ingresses:  []frontend.Ingress{&ti},
		Storage:    &storage,
		RenderFunc: pipeline.RenderFunc(render),
		PostPipelineHook: func(ctx context.Context, sharedDir string, jd *frontend.JobDone) error {
			ppHookCallCount++
			return nil
		},
	}

	stop, err := core.Start(ctx, &config)
	if err != nil {
		t.Fatal(err)
	}

	doneChan := make(chan struct{})
	go func() {
		wg.Wait()
		close(doneChan)
	}()

	timer := time.NewTimer(10 * time.Second)
	defer timer.Stop()
	select {
	case <-timer.C:
		t.Fatal("timed out waiting for requests to be handled")
	case <-doneChan:
	}

	if err = stop(ctx); err != nil {
		t.Fatal(err)
	}

	if len(jds) != len(requests) {
		t.Fatalf("expected %d job done(s), got %d", len(requests), len(jds))
	}

	for i, jd := range jds {
		artifactData, ok := provider1.Data[jd.ArtifactURL]
		if !ok {
			t.Fatalf("expected job done %d to have artifact data", i)
		}
		toc := jd.TableOfContents
		br := toc["0/test.pdf"]
		fileData := artifactData.Bytes()[br.Start:br.End]

		if !bytes.Equal(fileData, []byte("PASS")) {
			t.Fatalf("expected job done %d to have artifact data equal to 'PASS', got '%s'", i, artifactData.String())
		}
	}

	if ppHookCallCount != len(jds) {
		t.Fatalf("expected post pipeline hook to be called %d times, got %d", len(jds), ppHookCallCount)
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
