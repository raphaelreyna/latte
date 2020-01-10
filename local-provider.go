package main

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"text/template"
)

type localProvider struct {
	templateString string
	templateFile   string
	detailsBytes   *[]byte
	detailsFile    string
}

func (p *localProvider) getTemplate(left, right string) (*template.Template, error) {
	if p.templateString == "" {
		if p.templateFile == "" {
			return nil, errors.New("no template source provided")
		}
		return template.New(filepath.Base(p.templateFile)).Delims(left, right).ParseFiles(p.templateFile)
	}
	return template.New("tmpl").Delims(left, right).Parse(p.templateString)
}

func (p *localProvider) getDetails() (map[string]interface{}, error) {
	if p.detailsBytes == nil {
		if p.detailsFile == "" {
			return nil, errors.New("no details source provided")
		}
		f, err := os.Open(p.detailsFile)
		if err != nil {
			return nil, err
		}
		var deets map[string]interface{}
		err = json.NewDecoder(f).Decode(&deets)
		f.Close()
		if err != nil {
			return nil, err
		}
		return deets, nil
	}
	var deets map[string]interface{}
	err := json.Unmarshal(*p.detailsBytes, &deets)
	return deets, err
}
