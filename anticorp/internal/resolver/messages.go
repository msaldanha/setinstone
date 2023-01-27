package resolver

import (
	"github.com/msaldanha/setinstone/anticorp/message"
)

type QueryTypesEnum struct {
	QueryNameRequest  string
	QueryNameResponse string
}

var QueryTypes = QueryTypesEnum{
	QueryNameRequest:  "QUERY.NAME.REQUEST",
	QueryNameResponse: "QUERY.NAME.RESPONSE",
}

type Query struct {
	Data      string `json:"data,omitempty"`
	Reference string `json:"reference,omitempty"`
}

func (m Query) Bytes() []byte {
	var data []byte
	data = append(data, []byte(m.Data)...)
	data = append(data, []byte(m.Reference)...)
	return data
}

func ExtractQuery(m message.Message) Query {
	var r Query
	r, _ = m.Payload.(Query)
	return r
}
