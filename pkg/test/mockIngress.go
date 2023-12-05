package test

import (
	"context"
	"time"

	"github.com/raphaelreyna/latte/pkg/frontend"
)

type MockIngress struct {
	StartedAt     *time.Time
	StoppedAt     *time.Time
	RequestsQueue []*frontend.Request

	rc chan *frontend.Request
}

func (t *MockIngress) Start(ctx context.Context) error {
	n := time.Now()
	t.StartedAt = &n

	t.rc = make(chan *frontend.Request, len(t.RequestsQueue))
	for _, req := range t.RequestsQueue {
		t.rc <- req
	}
	return nil
}

func (t *MockIngress) Stop(ctx context.Context) error {
	n := time.Now()
	t.StoppedAt = &n
	close(t.rc)
	return nil
}

func (t *MockIngress) RequestsChan() <-chan *frontend.Request {
	return t.rc
}
