package pulpit

import "github.com/msaldanha/setinstone/timeline"

type AddItemRequest struct {
	Body        timeline.PostPart   `json:"body,omitempty"`
	Links       []timeline.PostPart `json:"links,omitempty"`
	Attachments []string            `json:"attachments,omitempty"`
	RefTypes    []string            `json:"ref_types,omitempty"`
}

type AddMediaRequest struct {
	Files []string `json:"files,omitempty"`
}

type AddMediaResult struct {
	File  string `json:"file,omitempty"`
	Id    string `json:"id,omitempty"`
	Error string `json:"error,omitempty"`
}

type AddReferenceRequest struct {
	Target string `json:"target,omitempty"`
	Type   string `json:"type,omitempty"`
}

type LoginRequest struct {
	Address  string `json:"address,omitempty"`
	Password string `json:"password,omitempty"`
}
