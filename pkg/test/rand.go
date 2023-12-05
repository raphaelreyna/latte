package test

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"strconv"
	"sync"

	"github.com/raphaelreyna/latte/pkg/frontend"
)

type RandReader struct{}

func (r *RandReader) Read(p []byte) (n int, err error) {
	return rand.Read(p)
}

func RandInt(min, max int) int {
	delta := big.NewInt(int64(max - min))
	randBigInt, _ := rand.Int(&RandReader{}, delta)
	return min + int(randBigInt.Int64())
}

func RandomDirData(dirSizeLimit, fileSizeLimit int) (map[string][]byte, error) {
	fileCount := RandInt(1, dirSizeLimit)
	m := make(map[string][]byte, fileCount)

	for i := 0; i < fileCount; i++ {
		fileSize := RandInt(1, fileSizeLimit)
		buf := make([]byte, fileSize)
		n, err := rand.Read(buf)
		if err != nil {
			return nil, err
		}
		if n != fileSize {
			return nil, fmt.Errorf("unable to read enough random bytes")
		}
		m[fmt.Sprintf("%d", i)] = buf
	}

	return m, nil
}

func RandomJob(source, target string) *frontend.Job {
	idx := RandInt(0, 9999)
	if source == "" {
		source = fmt.Sprintf("https://source/%d", idx)
	}
	if target == "" {
		target = fmt.Sprintf("https://target/%d", idx)
	}

	j := frontend.Job{
		ID:        strconv.Itoa(idx),
		SourceURI: source,
		TargetURI: target,
	}

	return &j
}

func RandomRequest(ctx context.Context, wg *sync.WaitGroup, source, target string) *frontend.Request {
	req := frontend.NewRequest(ctx)

	req.Job = RandomJob(source, target)

	if wg != nil {
		req.Done = func(jd *frontend.JobDone) {
			wg.Done()
		}
	} else {
		req.Done = func(jd *frontend.JobDone) {}
	}

	return req
}
