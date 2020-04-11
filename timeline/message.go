package timeline

type MessagePart struct {
	Seq      int    `json:"seq,omitempty"`
	Name     string `json:"name,omitempty"`
	MimeType string `json:"mime_type,omitempty"`
	Encoding string `json:"encoding,omitempty"`
	Data     string `json:"data,omitempty"`
}

type Link struct {
}
type Message struct {
	Seq         int           `json:"seq,omitempty"`
	Id          string        `json:"id,omitempty"`
	Address     string        `json:"address,omitempty"`
	Timestamp   string        `json:"timestamp,omitempty"`
	Body        MessagePart   `json:"body,omitempty"`
	Links       []MessagePart `json:"links,omitempty"`
	Attachments []MessagePart `json:"attachments,omitempty"`
}
