package root

import (
	"context"

	"github.com/raphaelreyna/latte/cmd/latte/pkg/commands/render"
	"github.com/spf13/cobra"
)

type rootCommand struct {
	cobra.Command
}

func ExecuteContext(ctx context.Context) error {
	var (
		root rootCommand
		cmd  = &root.Command
	)

	root.Use = "latte"
	root.setSubCommands()

	err := cmd.ExecuteContext(ctx)
	return err
}

func (root *rootCommand) setSubCommands() {
	for _, cmd := range subCommands() {
		root.AddCommand(cmd)
	}
}

func subCommands() []*cobra.Command {
	return []*cobra.Command{
		render.New().CobraCommand(),
	}
}
