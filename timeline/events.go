package timeline

type EventTypesEnum struct {
	EventReferenceAdded string
	EventPostAdded      string
}

var EventTypes = EventTypesEnum{
	EventReferenceAdded: "TIMELINE.EVENT.REFERENCE.ADDED",
	EventPostAdded:      "TIMELINE.EVENT.POST.ADDED",
}

type Event struct {
	Type string
	Id   string
}

func (e Event) Bytes() []byte {
	return []byte(e.Type + e.Id)
}
