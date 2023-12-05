package frontend

import (
	"encoding/json"
	"fmt"
	"net/url"
	"time"
)

type Job struct {
	ID string `json:"id"`

	TargetURI string `json:"targetURI"`
	SourceURI string `json:"sourceURI"`

	RequestedAt time.Time `json:"requestedAt"`

	// The following fields are optional

	Contexts        []json.RawMessage      `json:"contexts"`
	OnMissingKey    OnMissingKey           `json:"onMissingKey"`
	TemplatingFuncs map[string]interface{} `json:"-"`

	RenderCount int    `json:"renderCount"`
	IncludeLogs bool   `json:"includeLogs"`
	Image       string `json:"image"`
	Compiler    string `json:"compiler"`

	Timeout time.Duration `json:"timeout"`
}

func (j *Job) GetTargetURL() (*url.URL, error) {
	return url.Parse(j.TargetURI)
}

func (j *Job) GetSourceURI() (*url.URL, error) {
	return url.Parse(j.SourceURI)
}

func (j *Job) Validate() error {
	if j.ID == "" {
		return fmt.Errorf("job id cannot be empty")
	}

	if j.TargetURI == "" {
		return fmt.Errorf("job target uri cannot be empty")
	}

	if j.SourceURI == "" {
		return fmt.Errorf("job source uri cannot be empty")
	}

	return nil
}
