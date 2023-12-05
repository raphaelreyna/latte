package storage_test

import (
	"bytes"
	"context"
	"io/fs"
	"net/url"
	"os"
	"testing"

	"github.com/raphaelreyna/latte/pkg/storage"
	"github.com/raphaelreyna/latte/pkg/test"
)

func TestExtractPitchArchive(t *testing.T) {
	ctx := context.Background()
	provider1 := test.MockProvider{
		Data: make(map[string]*test.MockWriteCloser),
	}
	storage := storage.Storage{}
	storage.RegisterStorageProvider(&provider1, "mock1")

	dirData, err := test.RandomDirData(10, 100)
	if err != nil {
		t.Fatal(err)
	}
	archiveData, err := test.ArchiveAsPitch(dirData)
	if err != nil {
		t.Fatal(err)
	}

	err = storage.StoreBytes(ctx, archiveData, &url.URL{
		Scheme: "mock1",
		Host:   "testing",
		Path:   "/test1",
	})
	if err != nil {
		t.Fatal(err)
	}

	outDir, err := os.MkdirTemp("", "latte-test")
	if err != nil {
		t.Fatal(err)
	}

	err = storage.ExtractPitchArchive(ctx, &url.URL{
		Scheme: "mock1",
		Host:   "testing",
		Path:   "/test1",
	}, outDir)
	if err != nil {
		t.Fatal(err)
	}

	contents, err := os.ReadDir(outDir)
	if err != nil {
		t.Fatal(err)
	}

	if len(contents) != len(dirData) {
		t.Fatalf("expected %d files, got %d", len(dirData), len(contents))
	}

	infoMap := make(map[string]fs.DirEntry, len(contents))
	for _, info := range contents {
		infoMap[info.Name()] = info
	}

	for name, content := range dirData {
		info, found := infoMap[name]
		if !found {
			t.Fatalf("expected to find file %q", name)
		}

		if info.IsDir() {
			t.Fatalf("expected %q to be a file", name)
		}

		filePath := outDir + "/" + info.Name()
		fileContent, err := os.ReadFile(filePath)
		if err != nil {
			t.Fatalf("error reading file %q: %v", info.Name(), err)
		}

		if !bytes.Equal(content, fileContent) {
			t.Fatalf("expected file %q to contain %q, got %q", info.Name(), string(content), string(fileContent))
		}
	}
}
