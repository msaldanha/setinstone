package timeline

import (
	"encoding/json"
	"github.com/msaldanha/setinstone/anticorp/graph"
)

const (
	TypePost      = "Post"
	TypeReference = "Reference"
	TypeComment   = "Comment"
)

type Base struct {
	Seq        int      `json:"seq,omitempty"`
	Id         string   `json:"id,omitempty"`
	Address    string   `json:"address,omitempty"`
	Timestamp  string   `json:"timestamp,omitempty"`
	Type       string   `json:"type,omitempty"`
	Connectors []string `json:"connectors,omitempty"`
}

type PostPart struct {
	Seq      int    `json:"seq,omitempty"`
	Name     string `json:"name,omitempty"`
	MimeType string `json:"mimeType,omitempty"`
	Encoding string `json:"encoding,omitempty"`
	Data     string `json:"data,omitempty"`
}

type Post struct {
	Body        PostPart   `json:"body,omitempty"`
	Links       []PostPart `json:"links,omitempty"`
	Attachments []PostPart `json:"attachments,omitempty"`
}

type PostItem struct {
	Base
	Post
}

type Reference struct {
	Connector string `json:"connector,omitempty"`
	Target    string `json:"target,omitempty"`
}

type ReferenceItem struct {
	Base
	Reference
}

type Item interface {
	IsPost() bool
	AsPost() (PostItem, bool)
	IsReference() bool
	AsReference() (ReferenceItem, bool)
	AsBase() (Base, bool)
	AsInterface() (interface{}, bool)
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
	base.Connectors = v.Branches
	return &item{v: v, base: base}, nil
}

func (i *item) IsPost() bool {
	return i.base.Type == TypePost
}

func (i *item) AsPost() (PostItem, bool) {
	if !i.IsPost() {
		return PostItem{}, false
	}
	item := PostItem{}
	er := json.Unmarshal(i.v.Data, &item)
	if er != nil {
		return PostItem{}, false
	}
	i.updateBase(&item.Base)
	return item, true
}

func (i *item) IsReference() bool {
	return i.base.Type == TypeReference
}

func (i *item) AsReference() (ReferenceItem, bool) {
	if !i.IsReference() {
		return ReferenceItem{}, false
	}
	item := ReferenceItem{}
	er := json.Unmarshal(i.v.Data, &item)
	if er != nil {
		return ReferenceItem{}, false
	}
	i.updateBase(&item.Base)
	return item, true
}

func (i *item) AsBase() (Base, bool) {
	return i.base, true
}

func (i *item) AsInterface() (interface{}, bool) {
	switch i.base.Type {
	case TypeReference:
		return i.AsReference()
	case TypePost:
		return i.AsPost()
	}
	return nil, false
}

func (i *item) updateBase(base *Base) {
	base.Seq = i.v.Seq
	base.Id = i.v.Key
	base.Address = i.v.Address
	base.Timestamp = i.v.Timestamp
	base.Connectors = i.v.Branches
}
