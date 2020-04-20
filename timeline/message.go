package timeline

type ItemPart struct {
	Seq      int    `json:"seq,omitempty"`
	Name     string `json:"name,omitempty"`
	MimeType string `json:"mime_type,omitempty"`
	Encoding string `json:"encoding,omitempty"`
	Data     string `json:"data,omitempty"`
}

type Item struct {
	Seq         int        `json:"seq,omitempty"`
	Id          string     `json:"id,omitempty"`
	Address     string     `json:"address,omitempty"`
	Timestamp   string     `json:"timestamp,omitempty"`
	Body        ItemPart   `json:"body,omitempty"`
	Links       []ItemPart `json:"links,omitempty"`
	Attachments []ItemPart `json:"attachments,omitempty"`
}
