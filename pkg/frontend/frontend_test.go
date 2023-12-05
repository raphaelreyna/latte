package frontend_test

import (
	"context"
	"testing"
	"time"

	"github.com/raphaelreyna/latte/pkg/frontend"
	"github.com/raphaelreyna/latte/pkg/test"
)

func TestStart(t *testing.T) {
	ctx := context.Background()
	h := test.MockHandler{}
	ti := test.MockIngress{
		RequestsQueue: []*frontend.Request{
			test.RandomRequest(ctx, nil, "", ""),
			test.RandomRequest(ctx, nil, "", ""),
		},
	}
	h.Add(len(ti.RequestsQueue))

	stopFunc, err := frontend.Start(ctx, h.Handle, &ti)
	if err != nil {
		t.Fatalf("error starting frontend: %v", err)
	}

	doneChan := make(chan struct{})
	go func() {
		h.Wait()
		close(doneChan)
	}()
	timeout := time.After(1 * time.Second)
	select {
	case <-timeout:
		t.Fatalf("timed out waiting for requests to be handled")
	case <-doneChan:
	}

	if err = stopFunc(ctx); err != nil {
		t.Fatalf("error stopping frontend: %v", err)
	}

	receivedRequests := make(map[*frontend.Request]struct{})
	for _, req := range h.ReceivedRequests {
		receivedRequests[req] = struct{}{}
	}

	for _, req := range ti.RequestsQueue {
		if _, ok := receivedRequests[req]; !ok {
			t.Fatalf("expected request %v to be handled, but it was not", req)
		}
	}
}
