package server

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
)

type DB interface {
	// Store should be capable of storing a given []byte or contents of an io.ReadCloser
	Store(ctx context.Context, uid string, i interface{}) error
	// Fetch should return either a []byte, or io.ReadCloser.
	// If the requested resource could not be found, error should be of type NotFoundError
	Fetch(ctx context.Context, uid string) (interface{}, error)
	// Ping should check if the databases is reachable, if return error should be nil and non-nil otherwise.
	Ping(ctx context.Context) error
}

type NotFoundError struct{}

func (nfe *NotFoundError) Error() string {
	return "blob not found in database"
}

// toDisk only accepts argument i of types []byte or io.ReadCloser
func toDisk(i interface{}, path string) error {
	switch t := i.(type) {
	case []byte:
		bytes := i.([]byte)
		if bytes == nil {
			return fmt.Errorf("received nil pointer to []byte")
		}
		return ioutil.WriteFile(path, i.([]byte), os.ModePerm)
	case io.ReadCloser:
		f, err := os.Create(path)
		if err != nil {
			return err
		}
		rc := i.(io.ReadCloser)
		if rc == nil {
			return fmt.Errorf("received nil pointer to io.ReadCloser")
		}
		if _, err = io.Copy(f, rc); err != nil {
			return err
		}

	default:
		return fmt.Errorf("received interface of unexpected type: %v", t)
	}
	return nil
}
