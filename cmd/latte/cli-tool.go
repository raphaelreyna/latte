package main

import (
	"context"
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"text/template"

	"github.com/raphaelreyna/latte/internal/job"
	"github.com/rs/zerolog/log"
)

func cli() {
	t := flag.String("t", "", "path to template/tex file")
	d := flag.String("d", "", "path to details json file")
	flag.Parse()
	p := os.Args[len(os.Args)-1]
	if *t == "" {
		log.Fatal().
			Msg("no template/tex file provided")
	}
	if *d == "" {
		log.Fatal().
			Msg("no details json file provided")
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
			log.Fatal().Err(err).
				Str("arg", p).
				Msg("error parsing argument")
		}
		if !statInfo.IsDir() {
			p = ""
		}
	}

	if filepath.Ext(*t) != ".tex" {
		log.Fatal().
			Str("path", *t).
			Msg("invalid tex file")
	}
	_, err := os.Stat(*t)
	if err != nil {
		log.Fatal().
			Str("path", *t).
			Msg("error reading file stat")
	}

	if filepath.Ext(*d) != ".json" {
		log.Fatal().
			Str("path", *d).
			Msg("invalid json file")
	}
	_, err = os.Stat(*d)
	if err != nil {
		log.Fatal().
			Str("path", *d).
			Msg("error reading file stat")
	}

	if p == "" {
		p, err = os.Getwd()
		if err != nil {
			log.Fatal().Err(err).
				Msg("error while obtaining working directory")
		}
	}
	tmpl, err := template.New(filepath.Base(*t)).Delims("#!", "!#").ParseFiles(*t)
	if err != nil {
		log.Fatal().Err(err).
			Str("template", *t).
			Msg("error while parsing template")
	}

	var dtls map[string]interface{}
	dFile, err := os.Open(*d)
	if err != nil {
		log.Fatal().Err(err).
			Str("path", *t).
			Msg("error while opening details json file")
	}
	err = json.NewDecoder(dFile).Decode(&dtls)
	if err != nil {
		log.Fatal().Err(err).
			Str("path", *t).
			Msg("error while decoding json file")
	}

	j := job.NewJob(p, nil)
	j.Template = tmpl
	j.Details = dtls

	pdfPath, err := j.Compile(context.Background())
	if err != nil {
		log.Fatal().Err(err).
			Msg("error while compiling pdf")
	}

	log.Info().
		Str("path", filepath.Join(p, pdfPath)).
		Msg("successfully created PDF at location")
}
