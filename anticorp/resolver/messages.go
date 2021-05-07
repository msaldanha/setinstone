package resolver

import "github.com/msaldanha/setinstone/anticorp/message"

type MessageTypesEnum struct {
	QueryNameRequest  string
	QueryNameResponse string
}

var MessageTypes = MessageTypesEnum{
	QueryNameRequest:  "QUERY.NAME.REQUEST",
	QueryNameResponse: "QUERY.NAME.RESPONSE",
}

type Message struct {
	Body      string `json:"payload,omitempty"`
	Reference string `json:"reference,omitempty"`
}

func (m Message) Bytes() []byte {
	var data []byte
	data = append(data, []byte(m.Body)...)
	data = append(data, []byte(m.Reference)...)
	return data
}

func ConvertToMessage(m message.Message) Message {
	var r Message
	r, _ = m.Payload.(Message)
	return r
}
