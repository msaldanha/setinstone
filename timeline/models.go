package timeline

import (
	"encoding/json"
	"github.com/msaldanha/setinstone/anticorp/graph"
)

const (
	TypeMessage = "Message"
	TypeLike    = "Like"
	TypeComment = "Comment"
)

type Base struct {
	Seq       int      `json:"seq,omitempty"`
	Id        string   `json:"id,omitempty"`
	Address   string   `json:"address,omitempty"`
	Timestamp string   `json:"timestamp,omitempty"`
	Type      string   `json:"type,omitempty"`
	Children  []string `json:"children,omitempty"`
}

type MessagePart struct {
	Seq      int    `json:"seq,omitempty"`
	Name     string `json:"name,omitempty"`
	MimeType string `json:"mime_type,omitempty"`
	Encoding string `json:"encoding,omitempty"`
	Data     string `json:"data,omitempty"`
}

type Message struct {
	Body        MessagePart   `json:"body,omitempty"`
	Links       []MessagePart `json:"links,omitempty"`
	Attachments []MessagePart `json:"attachments,omitempty"`
}

type MessageItem struct {
	Base
	Message
}

type Item interface {
	IsMessage() bool
	AsMessage() (MessageItem, bool)
}

type item struct {
	v    graph.GraphNode
	base Base
}

func NewItemFromGraphNode(v graph.GraphNode) (Item, error) {
	base := Base{}
	er := json.Unmarshal(v.Data, &base)
	if er != nil {
		return nil, er
	}
	base.Seq = v.Seq
	base.Id = v.Key
	base.Address = v.Address
	base.Timestamp = v.Timestamp
	return &item{v: v, base: base}, nil
}

func (i *item) IsMessage() bool {
	return i.base.Type == TypeMessage
}

func (i *item) AsMessage() (MessageItem, bool) {
	if !i.IsMessage() {
		return MessageItem{}, false
	}
	item := MessageItem{}
	er := json.Unmarshal(i.v.Data, &item)
	if er != nil {
		return MessageItem{}, false
	}
	i.updateBase(&item.Base)
	return item, true
}

func (i *item) updateBase(base *Base) {
	base.Seq = i.v.Seq
	base.Id = i.v.Key
	base.Address = i.v.Address
	base.Timestamp = i.v.Timestamp
}
