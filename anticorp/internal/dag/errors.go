package dag

type ErrDagAlreadyInitialized struct {
	msg string
}

func (e *ErrDagAlreadyInitialized) Error() string {
	return e.msg
}

func (e *ErrDagAlreadyInitialized) Is(err error) (is bool) {
	_, is = err.(*ErrDagAlreadyInitialized)
	return
}

func NewErrDagAlreadyInitialized() *ErrDagAlreadyInitialized {
	return &ErrDagAlreadyInitialized{
		msg: "dag already initialized",
	}
}

type ErrInvalidNodeHash struct {
	msg string
}

func (e *ErrInvalidNodeHash) Error() string {
	return e.msg
}

func (e *ErrInvalidNodeHash) Is(err error) (is bool) {
	_, is = err.(*ErrInvalidNodeHash)
	return
}

func NewErrInvalidNodeHash() *ErrInvalidNodeHash {
	return &ErrInvalidNodeHash{
		msg: "invalid node hash",
	}
}

type ErrInvalidNodeTimestamp struct {
	msg string
}

func (e *ErrInvalidNodeTimestamp) Error() string {
	return e.msg
}

func (e *ErrInvalidNodeTimestamp) Is(err error) (is bool) {
	_, is = err.(*ErrInvalidNodeTimestamp)
	return
}

func NewErrInvalidNodeTimestamp() *ErrInvalidNodeTimestamp {
	return &ErrInvalidNodeTimestamp{
		msg: "invalid node timestamp",
	}
}

type ErrNodeAlreadyInDag struct {
	msg string
}

func (e *ErrNodeAlreadyInDag) Error() string {
	return e.msg
}

func (e *ErrNodeAlreadyInDag) Is(err error) (is bool) {
	_, is = err.(*ErrNodeAlreadyInDag)
	return
}

func NewErrNodeAlreadyInDag() *ErrNodeAlreadyInDag {
	return &ErrNodeAlreadyInDag{
		msg: "node already in dag",
	}
}

type ErrNodeNotFound struct {
	msg string
}

func (e *ErrNodeNotFound) Error() string {
	return e.msg
}

func (e *ErrNodeNotFound) Is(err error) (is bool) {
	_, is = err.(*ErrNodeNotFound)
	return
}

func NewErrNodeNotFound() *ErrNodeNotFound {
	return &ErrNodeNotFound{
		msg: "node not found",
	}
}

type ErrPreviousNodeNotFound struct {
	msg string
}

func (e *ErrPreviousNodeNotFound) Error() string {
	return e.msg
}

func (e *ErrPreviousNodeNotFound) Is(err error) (is bool) {
	_, is = err.(*ErrNodeNotFound)
	return
}

func NewErrPreviousNodeNotFound() *ErrPreviousNodeNotFound {
	return &ErrPreviousNodeNotFound{
		msg: "previous node not found",
	}
}

type ErrHeadNodeNotFound struct {
	msg string
}

func (e *ErrHeadNodeNotFound) Error() string {
	return e.msg
}

func (e *ErrHeadNodeNotFound) Is(err error) (is bool) {
	_, is = err.(*ErrNodeNotFound)
	return
}

func NewErrHeadNodeNotFound() *ErrHeadNodeNotFound {
	return &ErrHeadNodeNotFound{
		msg: "head node not found",
	}
}

type ErrPreviousNodeIsNotHead struct {
	msg string
}

func (e *ErrPreviousNodeIsNotHead) Error() string {
	return e.msg
}

func (e *ErrPreviousNodeIsNotHead) Is(err error) (is bool) {
	_, is = err.(*ErrPreviousNodeIsNotHead)
	return
}

func NewErrPreviousNodeIsNotHead() *ErrPreviousNodeIsNotHead {
	return &ErrPreviousNodeIsNotHead{
		msg: "previous node is not the chain head",
	}
}

type ErrAddressDoesNotMatchPubKey struct {
	msg string
}

func (e *ErrAddressDoesNotMatchPubKey) Error() string {
	return e.msg
}

func (e *ErrAddressDoesNotMatchPubKey) Is(err error) (is bool) {
	_, is = err.(*ErrAddressDoesNotMatchPubKey)
	return
}

func NewErrAddressDoesNotMatchPubKey() *ErrAddressDoesNotMatchPubKey {
	return &ErrAddressDoesNotMatchPubKey{
		msg: "address does not match public key",
	}
}

type ErrInvalidBranchSeq struct {
	msg string
}

func (e *ErrInvalidBranchSeq) Error() string {
	return e.msg
}

func (e *ErrInvalidBranchSeq) Is(err error) (is bool) {
	_, is = err.(*ErrInvalidBranchSeq)
	return
}

func NewErrInvalidBranchSeq() *ErrInvalidBranchSeq {
	return &ErrInvalidBranchSeq{
		msg: "invalid node sequence",
	}
}

type ErrInvalidBranch struct {
	msg string
}

func (e *ErrInvalidBranch) Error() string {
	return e.msg
}

func (e *ErrInvalidBranch) Is(err error) (is bool) {
	_, is = err.(*ErrInvalidBranch)
	return
}

func NewErrInvalidBranch() *ErrInvalidBranch {
	return &ErrInvalidBranch{
		msg: "invalid branch",
	}
}

type ErrBranchRootNotFound struct {
	msg string
}

func (e *ErrBranchRootNotFound) Error() string {
	return e.msg
}

func (e *ErrBranchRootNotFound) Is(err error) (is bool) {
	_, is = err.(*ErrBranchRootNotFound)
	return
}

func NewErrBranchRootNotFound() *ErrBranchRootNotFound {
	return &ErrBranchRootNotFound{
		msg: "branch root not found",
	}
}

type ErrDefaultBranchNotSpecified struct {
	msg string
}

func (e *ErrDefaultBranchNotSpecified) Error() string {
	return e.msg
}

func (e *ErrDefaultBranchNotSpecified) Is(err error) (is bool) {
	_, is = err.(*ErrDefaultBranchNotSpecified)
	return
}

func NewErrDefaultBranchNotSpecified() *ErrDefaultBranchNotSpecified {
	return &ErrDefaultBranchNotSpecified{
		msg: "default branch not specified",
	}
}

type ErrUnableToDecodeNodeSignature struct {
	msg string
}

func (e *ErrUnableToDecodeNodeSignature) Error() string {
	return e.msg
}

func (e *ErrUnableToDecodeNodeSignature) Is(err error) (is bool) {
	_, is = err.(*ErrUnableToDecodeNodeSignature)
	return
}

func NewErrUnableToDecodeNodeSignature() *ErrUnableToDecodeNodeSignature {
	return &ErrUnableToDecodeNodeSignature{
		msg: "unable to decode node signature",
	}
}

type ErrUnableToDecodeNodePubKey struct {
	msg string
}

func (e *ErrUnableToDecodeNodePubKey) Error() string {
	return e.msg
}

func (e *ErrUnableToDecodeNodePubKey) Is(err error) (is bool) {
	_, is = err.(*ErrUnableToDecodeNodePubKey)
	return
}

func NewErrUnableToDecodeNodePubKey() *ErrUnableToDecodeNodePubKey {
	return &ErrUnableToDecodeNodePubKey{
		msg: "unable to decode node pubkey",
	}
}

type ErrUnableToDecodeNodeHash struct {
	msg string
}

func (e *ErrUnableToDecodeNodeHash) Error() string {
	return e.msg
}

func (e *ErrUnableToDecodeNodeHash) Is(err error) (is bool) {
	_, is = err.(*ErrUnableToDecodeNodeHash)
	return
}

func NewErrUnableToDecodeNodeHash() *ErrUnableToDecodeNodeHash {
	return &ErrUnableToDecodeNodeHash{
		msg: "unable to decode node hash",
	}
}

type ErrNodeSignatureDoesNotMatch struct {
	msg string
}

func (e *ErrNodeSignatureDoesNotMatch) Error() string {
	return e.msg
}

func (e *ErrNodeSignatureDoesNotMatch) Is(err error) (is bool) {
	_, is = err.(*ErrNodeSignatureDoesNotMatch)
	return
}

func NewErrNodeSignatureDoesNotMatch() *ErrNodeSignatureDoesNotMatch {
	return &ErrNodeSignatureDoesNotMatch{
		msg: "node signature does not match",
	}
}
