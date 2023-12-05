package render

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/raphaelreyna/latte/pkg/core"
	"github.com/raphaelreyna/latte/pkg/frontend"
	"github.com/spf13/cobra"
)

func (c *Cmd) run(cmd *cobra.Command, args []string) (err error) {
	ctx := cmd.Context()
	err = c.validate()
	if err != nil {
		return fmt.Errorf("error validating render command: %w", err)
	}

	// start the core using passed in args and flags
	coreConfig := core.Config{
		SourceDir: "",
		SharedDir: "",

		Ingresses:  []frontend.Ingress{c},
		Storage:    nil,
		RenderFunc: RenderFunc,
	}
	stopFunc, err := core.Start(ctx, &coreConfig)
	defer func() {
		if e := stopFunc(ctx); e != nil {
			err = errors.Join(err, fmt.Errorf("error stopping core: %w", e))
		}
	}()
	if err != nil {
		return fmt.Errorf("error creating core: %w", err)
	}

	// create the job request for the render
	req := frontend.NewRequest(ctx)
	doneChan := make(chan *frontend.JobDone, 1)
	req.Done = func(jd *frontend.JobDone) {
		doneChan <- jd
		close(doneChan)
	}

	// send the request to the core via reqChan
	c.reqChan <- req
	close(c.reqChan)

	// wait for the job to finish
	jd := <-doneChan
	output, err := json.Marshal(jd)
	if err != nil {
		return fmt.Errorf("error marshaling job done: %w", err)
	}

	// print the output
	fmt.Println(string(output))

	return nil
}
