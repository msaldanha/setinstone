package pulpit

import "github.com/msaldanha/setinstone/timeline"

type AddItemRequest struct {
	Type          string        `json:"type,omitempty"`
	PostItem      PostItem      `json:"postItem,omitempty"`
	ReferenceItem ReferenceItem `json:"referenceItem,omitempty"`
}

type PostItem struct {
	timeline.Part
	Links       []timeline.PostPart `json:"links,omitempty"`
	Attachments []string            `json:"attachments,omitempty"`
	Connectors  []string            `json:"connectors,omitempty"`
}

type ReferenceItem struct {
	Target    string `json:"target,omitempty"`
	Connector string `json:"connector,omitempty"`
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
