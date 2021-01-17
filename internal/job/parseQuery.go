package job

import (
	"net/url"
	"errors"
	"text/template"
	"strconv"
)

// ParseQuery takes url.Values and loads the template and resources referenced in q into the root directory.
func (j *Job) ParseQuery(q url.Values, cache *TemplateCache) error {
	cOpts := j.Opts

	// Check if a registered template is being requested in the URL, if so make sure its available on the local disk
	if tmplID := q.Get("tmpl"); j.Template == nil && tmplID != "" {
		tmplID = tmplID + cOpts.Delims.Left + cOpts.Delims.Right
		cache.Lock()

		ti, exists := cache.Get(tmplID)
		if !exists {
			// Look for the requested template in the source chain and parse it
			if err := j.GetTemplate(q.Get("tmpl")); err != nil {
				cache.Unlock()
				return err
			}

			cache.Add(tmplID, j.Template)
		} else {
			j.Template = ti.(*template.Template)
		}
		cache.Unlock()
	} else if j.Template == nil {
		return errors.New("no template provided")
	}
	// Finish setting up the template
	if omk := q.Get("onMissingKey"); omk != "" && cOpts.OnMissingKey == "" {
		cOpts.OnMissingKey = MissingKeyOpt(omk)
		if omk := cOpts.OnMissingKey; !omk.IsValid() {
			return errors.New("invalid onMissingKey field found in JSON body")
		}
	}

	// handle linking resources into the working directory, downloading those that aren't in the root directory
	rscsIDs := q["rsc"]
	j.AddResource(rscsIDs...)

	// Load and parse details json from local disk, downloading it from the db if not found on local disk
	if dtID := q.Get("dtls"); len(j.Details) == 0 && dtID != "" {
		if err := j.GetDetails(dtID); err != nil {
			return err
		}
	}

	// finish configuring compilation options
	if cOpts.CC == "" {
		cOpts.CC = Compiler(q.Get("compiler"))
	}
	if cOpts.N < 2 {
		if n, err := strconv.Atoi(q.Get("count")); err == nil {
			cOpts.N = uint(n)
		}
	}

	// Set the job options
	j.Opts = cOpts

	return nil
}
