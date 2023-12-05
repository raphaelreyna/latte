package render

import (
	"errors"

	"github.com/raphaelreyna/latte/pkg/frontend"
	"github.com/raphaelreyna/latte/pkg/pipeline"
	"github.com/spf13/cobra"
)

var RenderFunc = pipeline.RenderFunc(nil)

type Cmd struct {
	cobraCommand *cobra.Command
	reqChan      chan *frontend.Request
}

func New() *Cmd {
	cmd := Cmd{
		reqChan: make(chan *frontend.Request),
	}
	return &cmd
}

func (cmd *Cmd) CobraCommand() *cobra.Command {
	if cmd.cobraCommand != nil {
		return cmd.cobraCommand
	}

	cmd.cobraCommand = &cobra.Command{
		Use:   "render [file|dir]",
		Short: "Render a file or directory",
	}
	cmd.cobraCommand.RunE = cmd.run

	return cmd.cobraCommand
}

func (cmd *Cmd) validate() error {
	if cmd.cobraCommand == nil {
		return errors.New("cobraCommand is required")
	}
	if cmd.reqChan == nil {
		return errors.New("reqChan is required")
	}
	return nil
}
