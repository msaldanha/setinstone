package event

type Event interface {
	Name() string
	Data() []byte
}

type event struct {
	name string
	data []byte
}

func (e event) Data() []byte {
	data := make([]byte, len(e.data))
	copy(data, e.data)
	return data
}

func (e event) Name() string {
	return e.name
}
