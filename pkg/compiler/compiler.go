package compiler

type Compiler interface {
	Name() string
	Args(outDir string, arg ...string) []string
}
