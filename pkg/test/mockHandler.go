package test

import (
	"sync"

	"github.com/raphaelreyna/latte/pkg/frontend"
)

type MockHandler struct {
	ReceivedRequests []*frontend.Request
	sync.WaitGroup
}

func (h *MockHandler) Handle(req *frontend.Request) {
	h.ReceivedRequests = append(h.ReceivedRequests, req)
	h.Done()
}
