package job

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/raphaelreyna/go-recon"
	"github.com/raphaelreyna/go-recon/sources"
)

type mockDB struct {
	data map[string]interface{}
}

func (mdb *mockDB) Store(ctx context.Context, uid string, i interface{}) error {
	mdb.data[uid] = i
	return nil
}

func (mdb *mockDB) Fetch(ctx context.Context, uid string) (interface{}, error) {
	result, exists := mdb.data[uid]
	if !exists {
		return nil, errors.New("file not found")
	}
	return result, nil
}

func (mdb *mockDB) Ping(ctx context.Context) error {
	return nil
}

func (mdb *mockDB) AddFileAs(name, destination string, perm os.FileMode) error {
	file, err := os.OpenFile(destination, os.O_CREATE|os.O_WRONLY, perm)
	if err != nil {
		return err
	}
	defer file.Close()

	data, exists := mdb.data[name]
	if !exists {
		os.Remove(file.Name())
		return fmt.Errorf("could not find file")
	}

	dataString := string(data.([]uint8))

	_, err = file.Write([]byte(dataString))

	return err
}

type TestEnv struct {
	Root string

	// Name of .tex file in the testing tex assets folder
	TexFile    string
	TexFileLoc int // 1 - registered and on disk; 2 - registered and in db and not on disk

	// Name of .json file in the testing details assets folder
	DtlsFile    string
	DtlsFileLoc int // 0 - unregistered; 1 - registered and on disk; 2 - registered and in db and not on disk

	// List of resource file names in the testing resources assets folder
	Resources    []string
	ResourcesLoc int // 0 - unregistered; 1 - registered and on disk; 2 - registered and in db and not on disk

	rootDir string
}

func (te *TestEnv) SourceChain() (recon.SourceChain, error) {
	var err error
	te.rootDir, err = ioutil.TempDir(te.Root, "source_*")
	if err != nil {
		return nil, err
	}

	db := &mockDB{data: map[string]interface{}{}}
	sc := sources.NewDirSourceChain(sources.SoftLink, te.rootDir)

	switch te.TexFileLoc {
	case 0:
		loc := filepath.Join(te.rootDir, te.TexFile)
		file, err := os.OpenFile(loc, os.O_CREATE|os.O_WRONLY, 0600)
		if err != nil {
			return nil, err
		}
		defer file.Close()

		originalLoc := filepath.Join(te.rootDir, "../../assets/templates/", te.TexFile)
		originalFile, err := os.Open(originalLoc)
		if err != nil {
			return nil, err
		}
		defer originalFile.Close()

		if _, err := io.Copy(file, originalFile); err != nil {
			return nil, err
		}
	case 1:
		originalLoc := filepath.Join(te.rootDir, "../../assets/templates/", te.TexFile)

		data, err := ioutil.ReadFile(originalLoc)
		if err != nil {
			return nil, err
		}

		if err = db.Store(context.Background(), te.TexFile, data); err != nil {
			return nil, err
		}
	}

	switch te.DtlsFileLoc {
	case 0:
		loc := filepath.Join(te.rootDir, te.DtlsFile)
		file, err := os.OpenFile(loc, os.O_CREATE|os.O_WRONLY, 0600)
		if err != nil {
			return nil, err
		}
		defer file.Close()

		originalLoc := filepath.Join(te.rootDir, "../../assets/details/"+te.DtlsFile)
		originalFile, err := os.Open(originalLoc)
		if err != nil {
			return nil, err
		}
		defer originalFile.Close()

		if _, err := io.Copy(file, originalFile); err != nil {
			return nil, err
		}
	case 1:
		originalLoc := filepath.Join(te.rootDir, "../../assets/details/"+te.DtlsFile)

		data, err := ioutil.ReadFile(originalLoc)
		if err != nil {
			return nil, err
		}

		if err = db.Store(context.Background(), te.DtlsFile, data); err != nil {
			return nil, err
		}
	}

	switch te.ResourcesLoc {
	case 0:
		for _, resource := range te.Resources {
			loc := filepath.Join(te.rootDir, resource)
			file, err := os.OpenFile(loc, os.O_CREATE|os.O_WRONLY, 0600)
			if err != nil {
				return nil, err
			}
			defer file.Close()

			originalLoc := filepath.Join(te.rootDir, "../../assets/templates/", resource)
			originalFile, err := os.Open(originalLoc)
			if err != nil {
				return nil, err
			}
			defer originalFile.Close()

			if _, err := io.Copy(file, originalFile); err != nil {
				return nil, err
			}
		}
	case 1:
		for _, resource := range te.Resources {
			originalLoc := filepath.Join(te.rootDir, "../../assets/templates/", resource)

			data, err := ioutil.ReadFile(originalLoc)
			if err != nil {
				return nil, err
			}

			if err = db.Store(context.Background(), resource, data); err != nil {
				return nil, err
			}
		}
	}

	sc = append(sc, db)

	return sc, nil
}

func (te *TestEnv) Clean() error {
	if te.rootDir == "" {
		return nil
	}

	return os.RemoveAll(te.rootDir)
}
