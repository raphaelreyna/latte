package latexmk

import "fmt"

var Compiler compiler

type compiler struct{}

func (compiler) Name() string {
	return "latexmk"
}

func (compiler) Args(outDir string, arg ...string) []string {
	var a = []string{
		"-pdf",
		fmt.Sprintf("-outdir=%s", outDir),
	}

	if len(arg) != 0 {
		a = append(a, arg...)
	}

	return a
}
