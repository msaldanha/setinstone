package pulpit

import "github.com/msaldanha/setinstone/timeline"

type AddMessageRequest struct {
	Body        timeline.MessagePart   `json:"body,omitempty"`
	Links       []timeline.MessagePart `json:"links,omitempty"`
	Attachments []string               `json:"attachments,omitempty"`
}
