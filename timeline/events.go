package timeline

import "encoding/json"

type EventTypesEnum struct {
	EventReferenceAdded string
	EventPostAdded      string
}

var EventTypes = EventTypesEnum{
	EventReferenceAdded: "TIMELINE.EVENT.REFERENCE.ADDED",
	EventPostAdded:      "TIMELINE.EVENT.POST.ADDED",
}

type Event struct {
	Type string `json:"type,omitempty"`
	Id   string `json:"id,omitempty"`
}

func (e Event) Bytes() []byte {
	return []byte(e.Type + e.Id)
}

func (e Event) ToJson() []byte {
	b, _ := json.Marshal(e)
	return b
}
