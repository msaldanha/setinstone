package pulpit

import "github.com/msaldanha/setinstone/timeline"

type AddMessageRequest struct {
	Body        timeline.PostPart   `json:"body,omitempty"`
	Links       []timeline.PostPart `json:"links,omitempty"`
	Attachments []string            `json:"attachments,omitempty"`
}

type AddMediaRequest struct {
	Files []string `json:"files,omitempty"`
}

type AddMediaResult struct {
	File  string `json:"file,omitempty"`
	Id    string `json:"id,omitempty"`
	Error string `json:"error,omitempty"`
}
