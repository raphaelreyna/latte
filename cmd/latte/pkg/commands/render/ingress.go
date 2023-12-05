package render

import (
	"context"

	"github.com/raphaelreyna/latte/pkg/frontend"
)

func (c *Cmd) Start(context.Context) error {
	return nil
}

func (c *Cmd) Stop(context.Context) error {
	return nil
}

func (c *Cmd) RequestsChan() <-chan *frontend.Request {
	return c.reqChan
}
