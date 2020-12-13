package compile

type Compiler string

var (
	PDFLatex Compiler = "pdflatex"
	Latexmk Compiler = "latexmk"
)

func (c Compiler) IsValid() bool {
	return c == PDFLatex || c == Latexmk
}

type Options struct {
	// CC is the LaTeX compiler to use
	CC Compiler
	// Dir is the working
	Dir string
	// N controls the number of compilations/passes.
	N uint
}

func DefaultOptions() Options {
	return Options{PDFLatex, "", 1}
}
