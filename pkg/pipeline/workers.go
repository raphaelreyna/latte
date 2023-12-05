package pipeline

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/raphaelreyna/latte/pkg/compiler/latexmk"
	"github.com/raphaelreyna/pitch"
)

func (p *Pipeline) templateWorker(ctx context.Context) {
	var (
		tjobs = p.tjobs.C()
		rjobs = p.rjobs.C()

		tg = p.tj
	)

	defer p.rjobs.Close()

	for {
		select {
		case <-ctx.Done():
			return
		case tj, ok := <-tjobs:
			if !ok {
				return
			}

			start := time.Now()

			var contextDir = filepath.Join(p.outDir, strconv.Itoa(tj.templatingContextIndex))
			var outdir = filepath.Join(contextDir, "template-out")
			if err := tg.Execute(outdir, tj.templatingContext); err != nil {
				p.error(tj.templatingContextIndex, "template-error", err)
				continue
			}

			rj := RenderJob{
				Start:        start,
				ContextIndex: tj.templatingContextIndex,
				InDir:        outdir,
				OutDir:       filepath.Join(contextDir, "render-out"),
				Compiler:     latexmk.Compiler,
				RunCount:     p.renderCount,
				context:      tj.context,
			}

			if err := os.MkdirAll(rj.OutDir, 0777); err != nil {
				p.error(tj.templatingContextIndex, "template-error", err)
				return
			}

			rjobs <- &rj
		}
	}
}

func (p *Pipeline) renderWorker(ctx context.Context) {
	var (
		rjobs = p.rjobs.C()
		ajobs = p.ajobs.C()
	)

	defer p.ajobs.Close()

	for {
		select {
		case <-ctx.Done():
			return
		case rj, ok := <-rjobs:
			if !ok {
				return
			}

			if hook := p.preRenderHook; hook != nil {
				if err := hook(rj.Ctx(), rj.InDir); err != nil {
					p.error(rj.ContextIndex, "pre-render-hook-error", err)
					continue
				}
			}

			if err := p.renderFunc(p, rj); err != nil {
				p.error(rj.ContextIndex, "render-error", err)
				continue
			}

			var prefix = filepath.Base(filepath.Dir(rj.OutDir)) // the parent directory of the render-out directory (the index of the context)
			ajobs <- &ajob{
				ctx:          rj.Ctx(),
				dir:          rj.OutDir,
				prefix:       prefix,
				contextIndex: rj.ContextIndex,
				duration:     time.Since(rj.Start),
			}
		}
	}
}

type ajob struct {
	ctx          context.Context
	prefix       string
	dir          string
	contextIndex int
	duration     time.Duration
}

func (p *Pipeline) archiveWorker(ctx context.Context) {
	var ajobs = p.ajobs.C()

	defer p.errChan.Close()

	for {
		select {
		case <-ctx.Done():
			return
		case aj, ok := <-ajobs:
			if !ok {
				return
			}

			if hook := p.preArchiveHook; hook != nil {
				if err := hook(aj.ctx, aj.dir); err != nil {
					p.error(aj.contextIndex, "pre-archive-hook-error", err)
					continue
				}
			}

			p.durations[aj.contextIndex] = aj.duration
			if err := p.archiveDir(aj.dir); err != nil {
				p.error(aj.contextIndex, "archive-error", err)
				continue
			}
		}
	}
}

// errorWorker listens for errors and stores them in a MultiError
// which is returned by Wait() if any errors were encountered.
//
// WARNING: errorWorker is not intended to be used with multiple goroutines.
func (p *Pipeline) errorWorker(ctx context.Context) {
	var (
		err     error
		errChan = p.errChan.C()
	)

	defer func() {
		if err != nil {
			p.doneChan <- err
		}
		close(p.doneChan)
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case e, ok := <-errChan:
			if !ok {
				return
			}
			err = errors.Join(err, e)
		}
	}
}

func (p *Pipeline) archiveDir(dir string) error {
	if p.noArchiving {
		return nil
	}

	parent, this := filepath.Split(dir)
	archiveOutFilePath := filepath.Join(parent, this+".pch")
	archiveFile, err := os.Create(archiveOutFilePath)
	if err != nil {
		return fmt.Errorf("error creating archive file: %w", err)
	}
	defer archiveFile.Close()

	return ArchiveDir(archiveFile, dir)
}

// MergeArchives merges all the archives from each context into a single archive.
// The merged archive is stored in the out directory with the name MergedArchiveName + ".pch".
func (p *Pipeline) MergeArchives(failOnFileNotFound bool) error {
	if p.noArchiving {
		return nil
	}

	outFilePath := filepath.Join(p.outDir, GetMergedArchiveName())
	outFile, err := os.Create(outFilePath)
	if err != nil {
		return fmt.Errorf("error creating archive file: %w", err)
	}
	defer outFile.Close()

	outFilePW := pitch.NewWriter(outFile)
	defer outFilePW.Close()

	tc := int(p.templateCount.Load())
	for i := 0; i < tc; i++ {
		index := strconv.Itoa(i)
		archiveFilePath := filepath.Join(p.outDir, index, "render-out.pch")

		err := func() error {
			archiveFile, err := os.Open(archiveFilePath)
			if err != nil {
				if errors.Is(err, os.ErrNotExist) {
					if failOnFileNotFound {
						return fmt.Errorf("error opening archive file: %w", err)
					}
					return nil
				}
				return fmt.Errorf("error opening archive file: %w", err)
			}
			defer func() {
				if e := archiveFile.Close(); e != nil {
					err = errors.Join(err, fmt.Errorf("error closing archive file: %w", e))
				}
				if e := os.Remove(archiveFilePath); e != nil {
					err = errors.Join(err, fmt.Errorf("error removing archive file: %w", e))
				}
			}()
			archiveFilePw := pitch.NewReader(archiveFile)
			defer archiveFilePw.Close()

			for {
				hdr, err := archiveFilePw.Next()
				if errors.Is(err, io.EOF) {
					break
				}
				if err != nil {
					return fmt.Errorf("error reading next header: %w", err)
				}

				_, err = outFilePW.WriteHeader(
					filepath.Join(index, hdr.Name),
					int64(hdr.Size),
					nil,
				)
				if err != nil {
					return fmt.Errorf("error writing header: %w", err)
				}

				if _, err := io.Copy(outFile, archiveFilePw); err != nil {
					return fmt.Errorf("error copying from archive file: %w", err)
				}
			}

			return nil
		}()
		if err != nil {
			return fmt.Errorf("error moving file: %w", err)
		}

	}

	return nil
}
