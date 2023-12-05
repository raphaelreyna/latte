package frontend

import (
	"context"
	"errors"
	"fmt"

	"golang.org/x/exp/slog"
)

type Request struct {
	Job     *Job
	Context func() context.Context
	Done    func(*JobDone)
}

func (r *Request) Validate() error {
	if r.Job == nil {
		return errors.New("job cannot be nil")
	}

	if err := r.Job.Validate(); err != nil {
		return fmt.Errorf("invalid job: %w", err)
	}

	if r.Context == nil {
		return errors.New("context cannot be nil")
	}

	if r.Done == nil {
		return errors.New("done cannot be nil")
	}

	return nil
}

func NewRequest(ctx context.Context) *Request {
	var req = Request{
		Context: func() context.Context {
			return ctx
		},
	}

	return &req
}

type Ingress interface {
	Start(context.Context) error
	Stop(context.Context) error
	RequestsChan() <-chan *Request
}

type RequestHandler func(*Request)

func Start(ctx context.Context, handler RequestHandler, ingress ...Ingress) (func(context.Context) error, error) {
	// for each ingress, start a goroutine that will handle requests
	// from the ingress by dispatching the request to the handler in a
	// new goroutine.
	for _, ing := range ingress {
		go func(ingress Ingress) {
			rc := ingress.RequestsChan()
			for req := range rc {
				if err := req.Validate(); err != nil {
					slog.ErrorContext(ctx, "invalid request", err, "request", req)
					continue
				}
				go handler(req)
			}
		}(ing)
		if err := ing.Start(ctx); err != nil {
			return func(context.Context) error { return nil }, err
		}
	}

	return func(ctx context.Context) (err error) {
		for _, ingress := range ingress {
			err = errors.Join(err, ingress.Stop(ctx))
		}
		return err
	}, nil
}
