package templatingengine

import (
	"errors"
	"io"

	"text/template"
)

var Default TemplatingEngine

var (
	ErrDelimsCollideWithTex = errors.New("delimiters collide with tex syntax")
	ErrDelimsEmpty          = errors.New("delimiters cannot be empty strings")
	ErrDelimsSymmetric      = errors.New("left and right delimiters cannot be the same")
)

type Template interface {
	Execute(io.Writer, any) error
	ParseFiles(filenames ...string) (Template, error)
}

type TemplatingEngine interface {
	NewTemplate(string, MissingKeyHandler, template.FuncMap) (Template, error)
}
