package pulpit

type MimeTypeData struct {
	MimeType string `json:"mime_type,omitempty"`
	Encoding string `json:"encoding,omitempty"`
	Data     string `json:"data,omitempty"`
}

type Link struct {
}
type Message struct {
	Timestamp   string         `json:"timestamp,omitempty"`
	Body        MimeTypeData   `json:"body,omitempty"`
	Links       []MimeTypeData `json:"links,omitempty"`
	Attachments []MimeTypeData `json:"attachments,omitempty"`
}
