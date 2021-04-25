package timeline

import (
	"encoding/json"
	"github.com/msaldanha/setinstone/anticorp/graph"
)

const (
	TypePost      = "Post"
	TypeReference = "Reference"
)

type Base struct {
	Type       string   `json:"type,omitempty"`
	Connectors []string `json:"connectors,omitempty"`
}

type Part struct {
	MimeType string `json:"mimeType,omitempty"`
	Encoding string `json:"encoding,omitempty"`
	Data     string `json:"data,omitempty"`
}

type PostPart struct {
	Seq  int    `json:"seq,omitempty"`
	Name string `json:"name,omitempty"`
	Part
}

type Post struct {
	Part
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
	Post
}

type ReferenceItem struct {
	Base
	Reference
}

type Item struct {
	graph.GraphNode
	Data interface{} `json:"data,omitempty"`
}

func NewItemFromGraphNode(v graph.GraphNode) (Item, error) {
	base := Base{}
	er := json.Unmarshal(v.Data, &base)
	if er != nil {
		return Item{}, er
	}

	item := Item{
		GraphNode: v,
	}

	var data interface{}
	switch base.Type {
	case TypeReference:
		ri := ReferenceItem{}
		er = json.Unmarshal(v.Data, &ri)
		data = ri
	case TypePost:
		p := PostItem{}
		er = json.Unmarshal(v.Data, &p)
		data = p
	default:
		er = ErrUnknownType
	}

	if er != nil {
		return Item{}, er
	}

	item.Data = data

	return item, nil
}
