package job

import (
	"github.com/raphaelreyna/go-recon"
	"text/template"
	"encoding/hex"
	"crypto/md5"
	"encoding/base64"
	"path/filepath"
	"io/ioutil"
	"os"
)

type Request struct {
	Template string `json:"template"`

	Details map[string]interface{} `json:"details"`

	Resources map[string]string `json:"resources"`

	Delimiters Delimiters `json:"delimiters"`
	OnMissingKey MissingKeyOpt `json:"onMissingKey"`
	Compiler Compiler `json:"compiler"`
	Count uint `json:"count"`
}

func (r *Request) NewJob(root string, sc recon.SourceChain, cache *TemplateCache) (*Job, error) {
	var err error
	j := &Job{Opts: DefaultOptions}
	j.Root = root
	j.SourceChain = sc

	opts := DefaultOptions

	if r.Delimiters != EmptyDelimiters && r.Delimiters != BadDefaultDelimiters {
		opts.Delims = r.Delimiters
	} else {
		opts.Delims = DefaultDelimiters
		r.Delimiters = DefaultDelimiters
	}

	if x := r.OnMissingKey; x != "" {
		opts.OnMissingKey = x
	}
	if x := r.Compiler; x != "" {
		opts.CC = x
	}
	if x := r.Count; x > 0 {
		opts.N = x
	}

	j.Opts = opts
	j.Details = r.Details

	if j.Template, err = r.parseTemplate(cache); err != nil {
		return nil, err
	}


	// Write resources files into working directory
	for name, data := range r.Resources {
		fname := filepath.Join(root, name)
		bytes, err := base64.StdEncoding.DecodeString(data)
		if err != nil {
			return nil, err
		}
		err = ioutil.WriteFile(fname, bytes, os.ModePerm)
		if err != nil {
			return nil, err
		}
	}

	return j, nil
}

func (r *Request) parseTemplate(cache *TemplateCache) (*template.Template, error) {
	if r.Template == "" {
		return nil, nil
	}

	// Check if we've already parsed this template; if not, parse it and cache the results
	tHash := md5.Sum([]byte(r.Template))
	// We append template delimiters to account for the same file being uploaded with different delimiters.
	// This would really only happen on accident but not taking it into account leads to unexpected caching behavior.
	cid := hex.EncodeToString(tHash[:]) + r.Delimiters.Left + r.Delimiters.Right
	cache.Lock()
	defer cache.Unlock()
	ti, exists := cache.Get(cid)
	var t *template.Template
	if !exists {
		tBytes, err := base64.StdEncoding.DecodeString(r.Template)
		if err != nil {
			cache.Unlock()
			return nil, err
		}
		t = template.New(cid).Delims(r.Delimiters.Left, r.Delimiters.Right)
		t, err = t.Parse(string(tBytes))
		if err != nil {
			cache.Unlock()
			return nil, err
		}

		cache.Add(cid, t)
	} else {
		t = ti.(*template.Template)
	}

	return t.Option("missingkey=" + r.OnMissingKey.Val()), nil
}
