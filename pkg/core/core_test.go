package core_test

import (
	"context"
	"encoding/json"
	"net/url"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/matryer/is"
	"github.com/raphaelreyna/latte/pkg/core"
	"github.com/raphaelreyna/latte/pkg/frontend"
	"github.com/raphaelreyna/latte/pkg/pipeline"
	"github.com/raphaelreyna/latte/pkg/storage"
	"github.com/raphaelreyna/latte/pkg/test"
)

func TestStart(t *testing.T) {
	ctx := context.Background()
	is := is.New(t)

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
	is.NoErr(err)

	err = storage.StoreBytes(ctx, archiveData, &url.URL{
		Scheme: "mock1",
		Host:   "testing",
		Path:   "/test",
	})
	is.NoErr(err)

	sourceDir, err := os.MkdirTemp("", "latte-test")
	is.NoErr(err)

	sharedDir, err := os.MkdirTemp("", "latte-test")
	is.NoErr(err)

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

	cr, err := core.NewCore(&config)
	is.NoErr(err)

	err = cr.Start(ctx)
	is.NoErr(err)

	doneChan := make(chan struct{})
	go func() {
		wg.Wait()
		close(doneChan)
	}()

	timer := time.NewTimer(5 * time.Second)
	defer timer.Stop()
	select {
	case <-timer.C:
		t.Fatal("timed out waiting for requests to be handled")
	case <-doneChan:
	}

	err = cr.Stop(ctx)
	is.NoErr(err)

	if len(jds) != len(requests) {
		t.Fatalf("expected %d job done(s), got %d", len(requests), len(jds))
	}

	for _, jd := range jds {
		artifactData, ok := provider1.Data[jd.ArtifactURL]
		is.True(ok)

		toc := jd.TableOfContents
		br, ok := toc["0/render-out/test.pdf"]
		is.True(ok)
		fileData := artifactData.Bytes()[br.Start:br.End]

		is.Equal(fileData, []byte("PASS"))
	}

	is.Equal(ppHookCallCount, len(jds))
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
