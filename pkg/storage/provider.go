package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"

	"github.com/raphaelreyna/pitch"
)

type Provider interface {
	Delete(context.Context, *url.URL) error
	WriteCloser(context.Context, *url.URL) (io.WriteCloser, error)
	ReadCloser(context.Context, *url.URL) (io.ReadCloser, error)
}

func ExtractPitchArchive(ctx context.Context, p Provider, src *url.URL, dstPath string) (err error) {
	if dirInfo, err := os.Stat(dstPath); err != nil {
		return fmt.Errorf("unable to stat destination path: %w", err)
	} else if !dirInfo.IsDir() {
		return fmt.Errorf("destination path is not a directory")
	}

	r, err := p.ReadCloser(ctx, src)
	if err != nil {
		return fmt.Errorf("unable to open read closer: %w", err)
	}

	defer func() {
		if e := r.Close(); e != nil {
			err = errors.Join(err, fmt.Errorf("unable to close read closer: %w", e))
		}
	}()

	pr := pitch.NewReader(r)

	for {
		hdr, err := pr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return fmt.Errorf("unable to read next header: %w", err)
		}

		var fullpath = dstPath + "/" + hdr.Name

		file, err := os.Create(fullpath)
		if err != nil {
			return fmt.Errorf("unable to create file: %w", err)
		}

		err = func() (err error) {
			defer func() {
				if e := file.Close(); e != nil {
					err = errors.Join(err, fmt.Errorf("unable to close file: %w", e))
				}
			}()

			if _, err = io.CopyN(file, pr, int64(hdr.Size)); err != nil {
				return fmt.Errorf("unable to copy file contents: %w", err)
			}

			return nil
		}()

		if err != nil {
			return err
		}
	}

	return nil
}

func StoreFile(ctx context.Context, p Provider, srcPath string, dst *url.URL) error {
	src, err := os.Open(srcPath)
	if err != nil {
		return fmt.Errorf("unable to open source file: %w", err)
	}

	defer func() {
		if e := src.Close(); e != nil {
			err = errors.Join(err, fmt.Errorf("unable to close source file: %w", e))
		}
	}()

	w, err := p.WriteCloser(ctx, dst)
	if err != nil {
		return fmt.Errorf("unable to open write closer: %w", err)
	}

	defer func() {
		if e := w.Close(); e != nil {
			err = errors.Join(err, fmt.Errorf("unable to close write closer: %w", e))
		}
	}()

	if _, err = io.Copy(w, src); err != nil {
		return fmt.Errorf("unable to copy contents: %w", err)
	}

	return nil
}

func StoreBytes(ctx context.Context, p Provider, src []byte, dst *url.URL) error {
	w, err := p.WriteCloser(ctx, dst)
	if err != nil {
		return fmt.Errorf("unable to open write closer: %w", err)
	}

	defer func() {
		if e := w.Close(); e != nil {
			err = errors.Join(err, fmt.Errorf("unable to close write closer: %w", e))
		}
	}()

	if _, err = w.Write(src); err != nil {
		return fmt.Errorf("unable to write contents: %w", err)
	}

	return nil
}
