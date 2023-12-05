package frontend

import (
	"time"

	"github.com/raphaelreyna/pitch"
)

type TableOfContents pitch.TableOfContents

type JobDone struct {
	JobID  string `json:"jobID,omitempty"`
	Status string `json:"status,omitempty"`

	ArtifactURL string `json:"artifactURL,omitempty"`

	TableOfContents TableOfContents `json:"resultTableOfContents,omitempty"`
	HasLogs         bool            `json:"hasLogs,omitempty"`
	RerenderCount   int             `json:"rerenderCount,omitempty"`

	Error string `json:"error,omitempty"`

	Renders []ContextRender `json:"renders,omitempty"`

	RequestedAt time.Time     `json:"requestedAt,omitempty"`
	StartedAt   time.Time     `json:"startedAt,omitempty"`
	Duration    time.Duration `json:"duration,omitempty"`
}

type ContextRender struct {
	Status   string        `json:"status,omitempty"`
	Errors   []error       `json:"errors,omitempty"`
	Duration time.Duration `json:"duration,omitempty"`
	Extra    any           `json:"extra,omitempty"`
}
