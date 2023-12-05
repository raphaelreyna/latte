package test

import (
	"bytes"

	"github.com/raphaelreyna/pitch"
)

func ArchiveAsPitch(data map[string][]byte) ([]byte, error) {
	buf := bytes.NewBuffer(nil)
	pw := pitch.NewWriter(buf)

	for name, content := range data {
		_, err := pw.WriteHeader(name, int64(len(content)), nil)
		if err != nil {
			return nil, err
		}

		_, err = pw.Write(content)
		if err != nil {
			return nil, err
		}
	}

	pw.Close()

	return buf.Bytes(), nil
}
