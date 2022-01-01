package timeline

type ErrReadOnly struct {
	msg string
}

func (e *ErrReadOnly) Error() string {
	return e.msg
}

func (e *ErrReadOnly) Is(err error) (is bool) {
	_, is = err.(*ErrReadOnly)
	return
}

func NewErrReadOnly() *ErrReadOnly {
	return &ErrReadOnly{
		msg: "read only",
	}
}

type ErrInvalidMessage struct {
	msg string
}

func (e *ErrInvalidMessage) Error() string {
	return e.msg
}

func (e *ErrInvalidMessage) Is(err error) (is bool) {
	_, is = err.(*ErrInvalidMessage)
	return
}

func NewErrInvalidMessage() *ErrInvalidMessage {
	return &ErrInvalidMessage{
		msg: "invalid message",
	}
}

type ErrUnknownType struct {
	msg string
}

func (e *ErrUnknownType) Error() string {
	return e.msg
}

func (e *ErrUnknownType) Is(err error) (is bool) {
	_, is = err.(*ErrUnknownType)
	return
}

func NewErrUnknownType() *ErrUnknownType {
	return &ErrUnknownType{
		msg: "unknown type",
	}
}

type ErrNotFound struct {
	msg string
}

func (e *ErrNotFound) Error() string {
	return e.msg
}

func (e *ErrNotFound) Is(err error) (is bool) {
	_, is = err.(*ErrNotFound)
	return
}

func NewErrNotFound() *ErrNotFound {
	return &ErrNotFound{
		msg: "not found",
	}
}

type ErrCannotRefOwnItem struct {
	msg string
}

func (e *ErrCannotRefOwnItem) Error() string {
	return e.msg
}

func (e *ErrCannotRefOwnItem) Is(err error) (is bool) {
	_, is = err.(*ErrCannotRefOwnItem)
	return
}

func NewErrCannotRefOwnItem() *ErrCannotRefOwnItem {
	return &ErrCannotRefOwnItem{
		msg: "cannot reference own item",
	}
}

type ErrCannotRefARef struct {
	msg string
}

func (e *ErrCannotRefARef) Error() string {
	return e.msg
}

func (e *ErrCannotRefARef) Is(err error) (is bool) {
	_, is = err.(*ErrCannotRefARef)
	return
}

func NewErrCannotRefARef() *ErrCannotRefARef {
	return &ErrCannotRefARef{
		msg: "cannot reference a reference",
	}
}

type ErrCannotAddReference struct {
	msg string
}

func (e *ErrCannotAddReference) Error() string {
	return e.msg
}

func (e *ErrCannotAddReference) Is(err error) (is bool) {
	_, is = err.(*ErrCannotAddReference)
	return
}

func NewErrCannotAddReference() *ErrCannotAddReference {
	return &ErrCannotAddReference{
		msg: "cannot add reference in this item",
	}
}

type ErrNotAReference struct {
	msg string
}

func (e *ErrNotAReference) Error() string {
	return e.msg
}

func (e *ErrNotAReference) Is(err error) (is bool) {
	_, is = err.(*ErrNotAReference)
	return
}

func NewErrNotAReference() *ErrNotAReference {
	return &ErrNotAReference{
		msg: "this item is not a reference",
	}
}

type ErrCannotAddRefToNotOwnedItem struct {
	msg string
}

func (e *ErrCannotAddRefToNotOwnedItem) Error() string {
	return e.msg
}

func (e *ErrCannotAddRefToNotOwnedItem) Is(err error) (is bool) {
	_, is = err.(*ErrCannotAddRefToNotOwnedItem)
	return
}

func NewErrCannotAddRefToNotOwnedItem() *ErrCannotAddRefToNotOwnedItem {
	return &ErrCannotAddRefToNotOwnedItem{
		msg: "cannot add reference to not owned item",
	}
}

type ErrInvalidParameterAddress struct {
	msg string
}

func (e *ErrInvalidParameterAddress) Error() string {
	return e.msg
}

func (e *ErrInvalidParameterAddress) Is(err error) (is bool) {
	_, is = err.(*ErrInvalidParameterAddress)
	return
}

func NewErrInvalidParameterAddress() *ErrInvalidParameterAddress {
	return &ErrInvalidParameterAddress{
		msg: "invalid parameter graph",
	}
}

type ErrInvalidParameterGraph struct {
	msg string
}

func (e *ErrInvalidParameterGraph) Error() string {
	return e.msg
}

func (e *ErrInvalidParameterGraph) Is(err error) (is bool) {
	_, is = err.(*ErrInvalidParameterGraph)
	return
}

func NewErrInvalidParameterGraph() *ErrInvalidParameterGraph {
	return &ErrInvalidParameterGraph{
		msg: "invalid parameter graph",
	}
}

type ErrInvalidParameterEventManager struct {
	msg string
}

func (e *ErrInvalidParameterEventManager) Error() string {
	return e.msg
}

func (e *ErrInvalidParameterEventManager) Is(err error) (is bool) {
	_, is = err.(*ErrInvalidParameterEventManager)
	return
}

func NewErrInvalidParameterEventManager() *ErrInvalidParameterEventManager {
	return &ErrInvalidParameterEventManager{
		msg: "invalid parameter event manager",
	}
}
