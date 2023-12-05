package pipeline

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/raphaelreyna/pitch"
)

func ArchiveDir(wc io.WriteCloser, name string) error {
	err := pitch.ArchiveDir(wc, name)
	return err
}

func Cat(wc io.Writer, names ...string) (err error) {
	readers := make([]pitch.Reader, len(names))
	for i, name := range names {
		file, err := os.Open(name)
		if err != nil {
			return fmt.Errorf("error opening file %s: %w", name, err)
		}
		readers[i] = pitch.NewReader(file)
	}

	catReader := pitch.Cat(readers...)
	defer func() {
		if e := catReader.Close(); e != nil {
			err = errors.Join(err, e)
		}
	}()

	if _, err = io.Copy(wc, catReader); err != nil {
		return fmt.Errorf("error copying from cat reader: %w", err)
	}

	return nil
}
