package pipeline

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func NewRenderFunc(shellPath, outDir string, includeEnv bool) RenderFunc {
	return RenderFunc(func(p *Pipeline, j *RenderJob) error {
		var (
			cc    = j.Compiler
			shCmd = fmt.Sprintf("%s %s",
				cc.Name(),
				strings.Join(cc.Args(outDir, j.Args...), " "),
			)
			cmd = exec.Cmd{
				Path:   shellPath,
				Args:   []string{"-c", shCmd},
				Stdout: os.Stdout,
				Stderr: os.Stderr,
			}
		)

		if includeEnv {
			cmd.Env = os.Environ()
		}

		err := cmd.Run()
		if err != nil {
			return fmt.Errorf("failed to run command: %w", err)
		}

		return nil
	})
}
