package core

import (
	"context"
	"errors"
	"fmt"

	"github.com/raphaelreyna/latte/pkg/frontend"
	"github.com/raphaelreyna/latte/pkg/pipeline"
	"github.com/raphaelreyna/latte/pkg/storage"
)

// Core ties together the frontend, storage, and runtime.
// It is responsible for handling requests from the frontend and
// dispatching them to the runtime for rendering and storage.
type Core struct {
	storage          *storage.Storage
	renderFunc       pipeline.RenderFunc
	postPipelineHook func(ctx context.Context, sharedDir string, jd *frontend.JobDone) error

	ingresses []frontend.Ingress

	workerCount int

	sourceDir string
	sharedDir string

	stop func(context.Context) error
}

type Config struct {
	// SourceDir is the directory where the source files are stored.
	SourceDir string
	// SharedDir is the directory shared with the runtime.
	// Files from SourceDir will be either rendered (if they're templates) or
	// symlinked (if they're not templates) to this directory.
	// This is also the directory where the runtime will store its output.
	SharedDir string

	// PostPipelineHook is a function that will be called after the pipeline has
	// finished with the job and the *JobDone has been created and set, and before
	// the archive is stored.
	// It can be used to perform cleanup, modify the *JobDone, modify the archive, etc.
	// If it returns an error, the job will be marked as failed.
	PostPipelineHook func(ctx context.Context, sharedDir string, jd *frontend.JobDone) error

	Ingresses  []frontend.Ingress
	Storage    *storage.Storage
	RenderFunc pipeline.RenderFunc

	// WorkerCount is the number of workers that will be used in the pipeline.
	// Anything less than 1 will be treated as 1.
	WorkerCount int
}

func (c *Config) validate() error {
	if c.SourceDir == "" {
		return errors.New("source dir is required")
	}

	if c.SharedDir == "" {
		return errors.New("shared dir is required")
	}

	if len(c.Ingresses) == 0 {
		return errors.New("at least one ingress is required")
	}

	if c.Storage == nil {
		return errors.New("storage is required")
	}

	if c.RenderFunc == nil {
		return errors.New("renderFunc is required")
	}

	if c.WorkerCount < 0 {
		c.WorkerCount = 1
	}

	return nil
}

func NewCore(c *Config) (*Core, error) {
	err := c.validate()
	if err != nil {
		return nil, fmt.Errorf("invalid core config: %w", err)
	}

	return &Core{
		storage:          c.Storage,
		renderFunc:       c.RenderFunc,
		sourceDir:        c.SourceDir,
		sharedDir:        c.SharedDir,
		ingresses:        c.Ingresses,
		workerCount:      c.WorkerCount,
		postPipelineHook: c.PostPipelineHook,
	}, nil
}

// Start starts the frontend using the given ingress(es) after registering itself as the handler.
// It returns a function that can be used to stop the frontend.
func (c *Core) Start(ctx context.Context) error {
	stop, err := frontend.Start(ctx, c.handleRequest, c.ingresses...)
	c.stop = stop

	return err
}

func (c *Core) Stop(ctx context.Context) error {
	if c.stop == nil {
		return nil
	}

	return c.stop(ctx)
}
