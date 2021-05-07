package message

type Payload interface {
	Bytes() []byte
}

type StringPayload string

func (s StringPayload) Bytes() []byte {
	return []byte(s)
}
