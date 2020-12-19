package job

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

type Test struct {
	Name string
	Env  *TestEnv

	// Name of the .pdf file in the testing pdf assets folder to test final product against
	ExpectedPDF    string
	ExpectedToPass bool

	// Needs to have keys "left" and "right", both of which have values which are two character strings
	Delimiters map[string]string

	// OnMissingKey valid values: 'error', 'zero', 'nothing'
	OnMissingKey string

	// Compiler valid values: "pdflatex", "latexmk"
	Compiler string
}

func (t *Test) Run(tt *testing.T, root string) {
	var err error

	j := Job{}
	j.AddFiles(t.Env.Resources...)
	j.Root = root
	t.Env.Root = root

	j.Opts = Options{
		CC:           Compiler(t.Compiler),
		OnMissingKey: MissingKeyOpt(t.OnMissingKey),
		Delims: Delimiters{
			Left:  t.Delimiters["left"],
			Right: t.Delimiters["right"],
		},
	}
	defer t.Env.Clean()

	j.SourceChain, err = t.Env.SourceChain()
	if err != nil {
		tt.Fatal(err)
	}

	if err = j.GetTemplate(t.Env.TexFile); err != nil {
		tt.Fatal(err)
	}

	if err = j.GetDetails(t.Env.DtlsFile); err != nil {
		tt.Fatal(err)
	}

	pdfLocation, err := j.Compile(context.Background())
	if err != nil {
		tt.Fatal(err)
	}

	if pdfLocation == "" {
		tt.Fatal("resulted in empty pdf location")
	}
}

func TestJob_Compile(t *testing.T) {
	// Create temp dir for testing
	currDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	tests := []Test{
		{
			Name:           "Basic",
			Delimiters:     map[string]string{"left": "#!", "right": "!#"},
			ExpectedPDF:    "hello-world_alice.pdf",
			ExpectedToPass: true,
			Env: &TestEnv{
				TexFile:   "hello-world.tex",
				DtlsFile:  "hello-world_alice.json",
				Resources: nil,
			},
		},
		{
			Name:           "Registered tex file",
			Delimiters:     map[string]string{"left": "#!", "right": "!#"},
			ExpectedPDF:    "hello-world_alice.pdf",
			ExpectedToPass: true,
			Env: &TestEnv{
				TexFile:    "hello-world.tex",
				TexFileLoc: 1,
				DtlsFile:   "hello-world_alice.json",
				Resources:  nil,
			},
		},
	}

	for _, test := range tests[1:2] {
		t.Run(test.Name, func(tt *testing.T) {
			testingDir, err := ioutil.TempDir(filepath.Join(currDir, "../../testing"), "testingTmp")
			if err != nil {
				t.Fatal(err)
			}
			err = os.Chdir(testingDir)
			if err != nil {
				t.Fatal(err)
			}
			test.Run(tt, testingDir)
			os.Chdir("../")
			os.RemoveAll(testingDir)
		})
	}
}
