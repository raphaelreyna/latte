package job

import "os/exec"

// Compiler represents the various compilers that are available
type Compiler string

func (c Compiler) IsValid() bool {
	return c == CC_PDFLatex || c == CC_Latexmk
}

var (
	CC_PDFLatex Compiler = "pdflatex"
	CC_Latexmk  Compiler = "latexmk"
	CC_Default  Compiler = CC_Latexmk
)

// Check if the system has Latemk installed, if so then make it the default
func init() {
	cmd := exec.Command("which", "latexmk")
	_, err := cmd.CombinedOutput()
	if err != nil {
		CC_Default = CC_PDFLatex
	} else {
		CC_Default = CC_Latexmk
	}
}

// MissingKeyOpt controls how missing keys are handled when filling in a template
type MissingKeyOpt string

var (
	// MK_Error will cause an error if details is missing a key used in the template
	MK_Error MissingKeyOpt = "error"
	// MK_Zero will cause values whose keys are missing from details to be replace with a zero value.
	MK_Zero MissingKeyOpt = "zero"
	// MK_Nothing will cause missing keys to be ignored.
	MK_Nothing MissingKeyOpt = "nothing"
)

func (mko MissingKeyOpt) IsValid() bool {
	return mko == MK_Error || mko == MK_Zero || mko == MK_Nothing
}

func (mko MissingKeyOpt) Val() string {
	switch mko {
	case "":
		fallthrough
	case MK_Nothing:
		return "default"
	default:
		return string(mko)
	}
}

// Delimiters holds the left and right delimiters for a template
type Delimiters struct {
	Left  string
	Right string
}

var DefaultDelimiters Delimiters = Delimiters{
	Left:  "#!",
	Right: "!#",
}

var BadDefaultDelimiters Delimiters = Delimiters{
	Left:  "{{",
	Right: "}}",
}

var EmptyDelimiters Delimiters = Delimiters{
	Left:  "",
	Right: "",
}

// Options holds the user settable options for a compilation job
type Options struct {
	// CC is the LaTeX compiler to use
	CC Compiler
	// N controls the number of compilations/passes.
	N uint
	// OnMissingKey controls how missing keys are handled when filling in the template
	OnMissingKey MissingKeyOpt
	// Delims holds the left and right delimiters to use for the template
	Delims Delimiters
}

var DefaultOptions Options = Options{
	CC:           CC_Default,
	N:            1,
	OnMissingKey: MK_Error,
	Delims:       DefaultDelimiters,
}
