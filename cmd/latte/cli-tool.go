package main

import (
	"context"
	"encoding/json"
	"flag"
	"github.com/raphaelreyna/latte/internal/job"
	"log"
	"os"
	"path/filepath"
	"text/template"
)

func cli(errLog, infoLog *log.Logger) {
	t := flag.String("t", "", "path to template/tex file")
	d := flag.String("d", "", "path to details json file")
	flag.Parse()
	p := os.Args[len(os.Args)-1]
	if *t == "" {
		errLog.Fatal("no template/tex file provided")
	}
	if *d == "" {
		errLog.Fatal("no details json file provided")
	}

	switch p {
	case *t:
		fallthrough
	case *d:
		fallthrough
	case os.Args[0]:
		fallthrough
	case "server":
		p = ""
	}

	if p != "" {
		statInfo, err := os.Stat(p)
		if err != nil {
			errLog.Fatalf("error while reading info for %s: %v", p, err)
		}
		if !statInfo.IsDir() {
			p = ""
		}
	}

	if filepath.Ext(*t) != ".tex" {
		errLog.Fatalf("%s must be a valid .tex file", *t)
	}
	_, err := os.Stat(*t)
	if err != nil {
		errLog.Fatalf("error while reading info for %s: %v", *t, err)
	}

	if filepath.Ext(*d) != ".json" {
		errLog.Fatalf("%s must be a valid .json file", *d)
	}
	_, err = os.Stat(*d)
	if err != nil {
		errLog.Fatalf("error while reading info for %s: %v", *d, err)
	}

	if p == "" {
		p, err = os.Getwd()
		if err != nil {
			errLog.Fatalf("error while obtaining working directory: %v", err)
		}
	}
	tmpl, err := template.New(filepath.Base(*t)).Delims("#!", "!#").ParseFiles(*t)
	if err != nil {
		errLog.Fatalf("error while parsing template %s: %v", *t, err)
	}

	var dtls map[string]interface{}
	dFile, err := os.Open(*d)
	if err != nil {
		errLog.Fatalf("error while opening details json file %s: %v", *t, err)
	}
	err = json.NewDecoder(dFile).Decode(&dtls)
	if err != nil {
		errLog.Fatalf("error while decoding json file %s: %v", *t, err)
	}

	j := job.NewJob(p, nil)
	j.Template = tmpl
	j.Details = dtls

	pdfPath, err := j.Compile(context.Background())
	if err != nil {
		errLog.Fatalf("error while compiling pdf: %v", err)
	}
	infoLog.Printf("Successfully created PDF at location: %s", filepath.Join(p, pdfPath))
}
