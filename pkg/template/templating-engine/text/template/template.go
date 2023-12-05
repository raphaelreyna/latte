package template

import (
	"io"
	"text/template"

	templatingengine "github.com/raphaelreyna/latte/pkg/template/templating-engine"
)

func init() {
	templatingengine.Default = &TemplatingEngine{}
}

type _template struct {
	t *template.Template
}

func (t *_template) Execute(out io.Writer, data interface{}) error {
	return t.t.Execute(out, data)
}

func (t *_template) Funcs(funcs map[string]interface{}) {
	t.t.Funcs(funcs)
}

func (t *_template) Options(opts ...string) {
	t.t.Option(opts...)
}

func (t *_template) SubTemplates() []templatingengine.Template {
	var (
		tmplts  = t.t.Templates()
		tmplts2 = make([]templatingengine.Template, len(tmplts))
	)

	for idx, tt := range tmplts {
		tmplts2[idx] = &_template{tt}
	}

	return tmplts2
}

func (t *_template) Name() string {
	return t.t.Name()
}

func (t *_template) ParseFiles(filenames ...string) (templatingengine.Template, error) {
	tt, err := t.t.ParseFiles(filenames...)
	if err != nil {
		return nil, err
	}

	return &_template{tt}, nil
}

type TemplatingEngine struct {
}

func NewTemplatingEngine() *TemplatingEngine {
	return &TemplatingEngine{}
}

func (te *TemplatingEngine) NewTemplate(name string, mkh templatingengine.MissingKeyHandler, fm template.FuncMap) (templatingengine.Template, error) {
	if mkh == "" {
		mkh = templatingengine.MissingKeyHandler_Error
	}

	var tmplt = template.New(name)

	tmplt = tmplt.Delims("|@", "@|")

	var mko = missingKeyOpt(mkh)
	if !mko.Valid() {
		mko = mk_error
	}

	tmplt = tmplt.Option(mko.Val())

	return &_template{tmplt}, nil
}

// missingKeyOpt controls how missing keys are handled when filling in a template
type missingKeyOpt string

var (
	// mk_error will cause an error if details is missing a key used in the template
	mk_error missingKeyOpt = "error"
	// mk_zero will cause values whose keys are missing from details to be replace with a zero value.
	mk_zero missingKeyOpt = "zero"
	// mk_nothing will cause missing keys to be ignored.
	mk_nothing missingKeyOpt = "nothing"
)

func (mko missingKeyOpt) Valid() bool {
	return mko == mk_error || mko == mk_zero || mko == mk_nothing
}

func (mko missingKeyOpt) Val() string {
	switch mko {
	case "":
		fallthrough
	case mk_nothing:
		return "missingkey=default"
	default:
		return "missingkey=" + string(mko)
	}
}
