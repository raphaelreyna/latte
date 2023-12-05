package pipeline

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/raphaelreyna/latte/pkg/compiler"
	"github.com/raphaelreyna/latte/pkg/frontend"
	"github.com/raphaelreyna/latte/pkg/template"
	templatingengine "github.com/raphaelreyna/latte/pkg/template/templating-engine"
	texttemplate "github.com/raphaelreyna/latte/pkg/template/templating-engine/text/template"
	"github.com/raphaelreyna/pitch"
)

// mergedArchiveName is the name (+".pch") that will be
// given to the merged archive files.
var mergedArchiveName = "archive.pch"

func SetMergedArchiveName(name string) {
	if name == "" {
		return
	}

	if !strings.HasSuffix(name, ".pch") {
		mergedArchiveName = name + ".pch"
	} else {
		mergedArchiveName = name
	}
}

func GetMergedArchiveName() string {
	return mergedArchiveName
}

type MultiError interface {
	Unwrap() []error
}

func Unwrap(err error) []error {
	if err == nil {
		return nil
	}
	m, ok := err.(MultiError)
	if !ok {
		return []error{err}
	}
	return m.Unwrap()
}

type RenderFunc func(*Pipeline, *RenderJob) error

type Job struct {
	context                context.Context
	templatingContext      json.RawMessage
	templatingContextIndex int
}

func NewJob(tmpltCtxIdx int, tmpltCtxData json.RawMessage) *Job {
	return &Job{
		templatingContextIndex: tmpltCtxIdx,
		templatingContext:      tmpltCtxData,
	}
}

type RenderJob struct {
	ContextIndex int
	Start        time.Time
	InDir        string
	OutDir       string
	Compiler     compiler.Compiler
	RunCount     int
	Args         []string

	context context.Context
}

func (rj *RenderJob) Ctx() context.Context {
	return rj.context
}

type Configuration struct {
	// WorkerCount is the number of workers that are used for the template and render jobs.
	WorkerCount int
	// RenderCount is the number of times the render job is executed.
	RenderCount int
	// SourceDir is the directory that contains the template files (and non-template files).
	// This directory can be read-only.
	SourceDir string
	// OutDir is the directory that contains the rendered files (and non-template files).
	// Both output from the template and render jobs are stored in this directory.
	// This directory must be writable.
	// This directory will have a subdirectory for each template context: "0", "1", "2", ...
	// If no template contexts are provided, this directory will have a single subdirectory, "0".
	// If OutDir does not exist, it will be created.
	OutDir string
	// OnMissingKey is the handler that is called when a key is missing in the template context.
	OnMissingKey string
	// RenderFunc is the function that is called to render the files once the template job has finished.
	RenderFunc RenderFunc

	NoArchiving bool

	// PreArchiveHook is a function that is called before the results archive is created.
	// It is passed the path to the shared directory.
	// WARNING: This function may be called concurrently.
	PreArchiveHook func(context.Context, string) error
	// PreRenderHook is a function that is called before the render job is executed.
	// It is passed the path to the shared directory.
	// WARNING: This function may be called concurrently.
	PreRenderHook func(context.Context, string) error
	// ContextContext is a function that is called with a base context.Context and the index of the job context.
	// It returns a context.Context that is used for the render job context context.Context.
	ContextContext func(context.Context, int) context.Context
	// FuncMap is a map of functions that are made available to the templates.
	// WARNING: This map and its functions may be called concurrently.
	FuncMap template.FuncMap
}

type Pipeline struct {
	baseCtx context.Context

	tjobs   safeChan[*Job, chan *Job]
	rjobs   safeChan[*RenderJob, chan *RenderJob]
	ajobs   safeChan[*ajob, chan *ajob]
	errChan safeChan[error, chan error]

	doneChan chan error

	tj *template.Job

	renderCount int
	renderFunc  RenderFunc

	outDir string

	noArchiving bool
	toc         frontend.TableOfContents

	templateCount atomic.Int32

	cancelWorkers func()

	durations map[int]time.Duration

	preArchiveHook func(context.Context, string) error
	preRenderHook  func(context.Context, string) error
	contextContext func(context.Context, int) context.Context
}

func NewPipeline(ctx context.Context, c Configuration) (*Pipeline, error) {
	var (
		p = Pipeline{
			baseCtx:        ctx,
			tjobs:          newSafeChan(make(chan *Job, c.WorkerCount), nil),
			rjobs:          newSafeChan(make(chan *RenderJob, c.WorkerCount), nil),
			ajobs:          newSafeChan(make(chan *ajob, c.WorkerCount), nil),
			errChan:        newSafeChan(make(chan error, c.WorkerCount), nil),
			doneChan:       make(chan error, 1),
			durations:      make(map[int]time.Duration),
			outDir:         c.OutDir,
			noArchiving:    c.NoArchiving,
			renderCount:    c.RenderCount,
			renderFunc:     c.RenderFunc,
			preArchiveHook: c.PreArchiveHook,
			preRenderHook:  c.PreRenderHook,
			contextContext: c.ContextContext,
		}
		err error
	)

	if c.ContextContext == nil {
		p.contextContext = func(ctx context.Context, idx int) context.Context {
			return ctx
		}
	}

	if c.SourceDir == "" {
		return nil, errors.New("invalid source dir")
	}

	p.tj, err = template.NewJob(&template.JobConfig{
		Dir:               c.SourceDir,
		MissingKeyHandler: templatingengine.MissingKeyHandler(c.OnMissingKey),
		TemplatingEngine:  texttemplate.NewTemplatingEngine(),
		FuncMap:           c.FuncMap,
	})
	if err != nil {
		return nil, fmt.Errorf("error creating new template group: %w", err)
	}

	_, err = os.Stat(p.outDir)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("error checking if out dir exists: %w", err)
		}
		return nil, fmt.Errorf("out dir does not exist: %w", err)
	}

	cctx, cancel := context.WithCancel(ctx)
	p.cancelWorkers = cancel

	go p.errorWorker(cctx)
	for i := 0; i < c.WorkerCount; i++ {
		go p.templateWorker(cctx)
		go p.renderWorker(cctx)
		go p.archiveWorker(cctx)
	}

	return &p, nil
}

func (p *Pipeline) MergedArchiveName() string {
	return filepath.Join(p.outDir, mergedArchiveName)
}

func (p *Pipeline) Add(tj *Job) {
	p.templateCount.Add(1)
	tj.context = p.contextContext(p.baseCtx, tj.templatingContextIndex)
	p.tjobs.C() <- tj
}

func (p *Pipeline) Close() {
	p.tjobs.Close()
}

func (p *Pipeline) Wait(ctx context.Context) error {
	var err error
	select {
	case <-ctx.Done():
		err = ctx.Err()
	case err = <-p.doneChan:
	}

	p.cancelWorkers()

	return err
}

func (p *Pipeline) Durations() map[int]time.Duration {
	return p.durations
}

func (p *Pipeline) TableOfContents() (frontend.TableOfContents, error) {
	if p.toc != nil {
		return p.toc, nil
	}

	filePath := filepath.Join(p.outDir, "archive.pch")
	file, err := os.Open(filePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("error opening archive file: %w", err)
	}
	defer file.Close()

	toc, err := pitch.BuildTableOfContents(file)
	if err != nil {
		return nil, fmt.Errorf("error building table of contents: %w", err)
	}

	p.toc = frontend.TableOfContents(toc)

	return p.toc, nil
}

func (p *Pipeline) error(idx int, msg string, err error) {
	p.errChan.C() <- fmt.Errorf("[%d]%s: %w", idx, msg, err)
}

func ParsePipelineError(err error) (int, string, error, error) {
	if err == nil {
		return -1, "", nil, nil
	}
	s := err.Error()
	parts := strings.SplitN(s, ": ", 2)
	if len(parts) != 2 {
		return -1, "", nil, fmt.Errorf("error parsing pipeline error: %w", err)
	}

	headParts := strings.SplitN(parts[0], "]", 2)
	if len(headParts) != 2 {
		return -1, "", nil, fmt.Errorf("error parsing pipeline error: %w", err)
	}

	idxString := headParts[0][1:]
	idx, err := strconv.Atoi(idxString)
	if err != nil {
		return -1, "", nil, fmt.Errorf("error parsing pipeline error: %w", err)
	}

	return idx, headParts[1], errors.New(parts[1]), nil

}

type txChan[T any] interface {
	chan T | chan<- T
}

type safeChan[T any, C txChan[T]] struct {
	c      C
	closed bool
	mu     *sync.Mutex
}

func newSafeChan[T any, C txChan[T]](c C, mu *sync.Mutex) safeChan[T, C] {
	sc := safeChan[T, C]{
		c:  c,
		mu: mu,
	}
	if sc.mu == nil {
		sc.mu = &sync.Mutex{}
	}

	return sc
}

func (sc *safeChan[T, C]) C() C {
	return sc.c
}

func (sc *safeChan[T, C]) Close() {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	if sc.closed {
		return
	}

	close(sc.c)
	sc.closed = true
}
