package main

import (
	"text/template"
)

type templateProvider interface {
	getTemplate(string, string) (*template.Template, error)
}

type detailsProvider interface {
	getDetails() (map[string]interface{}, error)
}
