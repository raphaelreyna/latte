package core

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/raphaelreyna/latte/pkg/frontend"
	"github.com/raphaelreyna/latte/pkg/log"
	"github.com/raphaelreyna/latte/pkg/pipeline"
)

func (c *core) handleRequest(req *frontend.Request) {
	// TODO: validate job in the request
	var (
		ctx  = req.Context()
		strg = c.storage

		j  = req.Job
		jd = frontend.JobDone{
			JobID:       j.ID,
			HasLogs:     j.IncludeLogs,
			RequestedAt: j.RequestedAt,
			Renders:     make([]frontend.ContextRender, len(j.Contexts)),
		}
	)

	// The source URI
	sourceURI, err := j.GetSourceURI()
	if err != nil {
		log.Error(ctx, "error parsing source uri", err)
		fail(req, &jd, err)
		return
	}

	targetURL, err := j.GetTargetURL()
	if err != nil {
		log.Error(ctx, "error parsing target url", err)
		fail(req, &jd, err)
		return
	}

	// Make sure the source and workspace directories are created.
	// The sourceDir is always created; if the source is a local directory,
	// it will be symlinked to the sourceDir, otherwise its assumed to be an archive
	// and will be unarchived to the sourceDir.
	sourceDir, sharedDir, err := c.prepareDirs(ctx, j.ID, sourceURI)
	if err != nil {
		log.Error(ctx, "error preparing directories", err)
		fail(req, &jd, err)
		return
	}

	// make sure to remove the source shared directories as
	// part of the cleanup process
	defer func() {
		if err := removeAll(sourceDir, sharedDir); err != nil {
			log.Error(ctx, "error removing source and shared directories", err)
		}
	}()

	var (
		pplnConf = pipeline.Configuration{
			SourceDir:    sourceDir,
			OutDir:       sharedDir,
			WorkerCount:  3, // TODO: make this configurable
			RenderCount:  j.RenderCount,
			OnMissingKey: string(j.OnMissingKey),
			RenderFunc:   c.renderFunc,
		}
	)

	ppln, err := pipeline.NewPipeline(ctx, pplnConf)
	if err != nil {
		log.Error(ctx, "error creating pipeline", err)
		fail(req, &jd, err)
		return
	}

	// for each context, create a job and add it to the pipeline
	// also create a context render for each context.
	jd.StartedAt = time.Now()
	for idx, tctx := range j.Contexts {
		pj := pipeline.NewJob(idx, tctx)
		ppln.Add(pj)
		jd.Renders[idx] = frontend.ContextRender{}
	}
	ppln.Close()

	// Create a context with a timeout and render the job
	if j.Timeout != 0 {
		var cancel func()
		ctx, cancel = context.WithTimeout(ctx, j.Timeout)
		defer cancel()
	}
	err = ppln.Wait(ctx)
	jd.Duration = time.Since(jd.StartedAt)

	// execute the post pipeline hook
	if c.postPipelineHook != nil {
		if err := c.postPipelineHook(ctx, sharedDir, &jd); err != nil {
			log.Error(ctx, "error executing post job done hook", err)
			fail(req, &jd, err)
			return
		}
	}

	// start merging archives in the background
	mergeErrChan := make(chan error, 1)
	go func(failOnFileNotFound bool) {
		defer close(mergeErrChan)
		err := ppln.MergeArchives(failOnFileNotFound)
		if err != nil {
			mergeErrChan <- err
		}
	}(true)

	// Unwrap the errors and parse them and add them to the context renders
	errs := pipeline.Unwrap(err)
	for _, e := range errs {
		contextIdx, msg, contextErr, err := pipeline.ParsePipelineError(e)
		if err != nil {
			log.Error(ctx, "error parsing pipeline error", err)
			fail(req, &jd, err)
			return
		}
		cr := jd.Renders[contextIdx]
		errString := fmt.Sprintf("%s: %s", msg, contextErr)
		cr.Errors = append(cr.Errors, errors.New(errString))
	}

	// flag failed renders and add durations
	renderFailedCount := 0
	durations := ppln.Durations()
	for idx, cr := range jd.Renders {
		duration, ok := durations[idx]
		if !ok {
			log.Error(ctx,
				"unable to get duration for context", nil,
				"contextIndex", idx,
			)
			fail(req, &jd, err)
			return
		}
		cr.Duration = duration

		if len(cr.Errors) == 0 {
			cr.Status = "success"
		} else {
			cr.Status = "failed"
			renderFailedCount++
		}

		jd.Renders[idx] = cr
	}
	if renderFailedCount == len(jd.Renders) {
		jd.Status = "failed"
	} else if renderFailedCount > 0 {
		jd.Status = "partial"
	} else {
		jd.Status = "success"
	}

	// wait for the archives to be merge then get the table of contents
	mergeErr := <-mergeErrChan
	if mergeErr != nil {
		log.Error(ctx, "error merging archives", err)
		fail(req, &jd, err)
		return
	}
	jd.TableOfContents, err = ppln.TableOfContents()
	if err != nil {
		log.Error(ctx, "error getting table of contents", err)
		fail(req, &jd, err)
		return
	}

	// store the final archive in the storage
	archivePath := filepath.Join(sharedDir, pipeline.GetMergedArchiveName())
	if err := strg.StoreFile(ctx, archivePath, targetURL); err != nil {
		log.Error(ctx, "error storing results", err)
		fail(req, &jd, err)
		return
	}
	jd.ArtifactURL = targetURL.String()

	req.Done(&jd)
}

func fail(req *frontend.Request, jd *frontend.JobDone, err error) {
	jd.Status = "failed"
	if err != nil {
		jd.Error = err.Error()
	}
	req.Done(jd)
}

func mkdir(paths ...string) error {
	for _, p := range paths {
		if err := os.MkdirAll(p, os.ModePerm); err != nil {
			return err
		}
	}
	return nil
}

func removeAll(paths ...string) (err error) {
	for _, p := range paths {
		if e := os.RemoveAll(p); e != nil {
			err = errors.Join(err, e)
		}
	}
	return err
}

// prepareDirs prepares the source and shared directories.
// If the source is a local directory, it will be symlinked to the source directory.
// If the source is an archive, it will be fetched and unarchived to the source directory.
func (c *core) prepareDirs(ctx context.Context, id string, src *url.URL) (sourceDir, sharedDir string, err error) {
	isLocalDir, err := isLocalDir(src)
	if err != nil {
		return "", "", fmt.Errorf("error checking if source is local directory: %w", err)
	}

	// sharedDir is the directory shared with the runtime.
	// For each context in the job, the pipeline will create a directory
	// (which will have its index as its name) and render the templates from source
	// into that directory while symlinking the non-templates.
	sharedDir = filepath.Join(c.sharedDir, id)
	// Create new shared directories
	if err = mkdir(sharedDir); err != nil {
		return "", "", fmt.Errorf("unable to create shared directory: %w", err)
	}

	// sourceDir is the directory where the source files will be stored.
	// This includes templates and non-templates.
	sourceDir = filepath.Join(c.sourceDir, id)
	if isLocalDir {
		// If the source is a local directory, symlink it to the source directory
		if err = os.Symlink(src.Path, sourceDir); err != nil {
			return "", "", fmt.Errorf("unable to symlink source directory: %w", err)
		}
	} else {
		// If the source is not a local directory, then it's an archive.
		// Fetch and unarchive the source directory
		if err = mkdir(sourceDir); err != nil {
			return "", "", fmt.Errorf("unable to create source directory: %w", err)
		}

		err = c.storage.ExtractPitchArchive(ctx, src, sourceDir)
		if err != nil {
			return "", "", fmt.Errorf("unable to extract pitch archive: %w", err)
		}
	}

	return
}

func isLocalDir(u *url.URL) (bool, error) {
	if u.Scheme != "local" {
		return false, nil
	}

	stat, err := os.Stat(u.Path)
	if err != nil {
		return false, fmt.Errorf("error stating path: %w", err)
	}

	return stat.IsDir(), nil
}
