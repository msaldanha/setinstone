package dag

import "errors"

var (
	ErrDagAlreadyInitialized       = errors.New("dag already initialized")
	ErrInvalidNodeHash             = errors.New("invalid node hash")
	ErrInvalidNodeTimestamp        = errors.New("invalid node timestamp")
	ErrNodeAlreadyInDag            = errors.New("node already in dag")
	ErrNodeNotFound                = errors.New("node not found")
	ErrPreviousNodeNotFound        = errors.New("previous node not found")
	ErrHeadNodeNotFound            = errors.New("head node not found")
	ErrPreviousNodeIsNotHead       = errors.New("previous node is not the chain head")
	ErrAddressDoesNotMatchPubKey   = errors.New("address does not match public key")
	ErrInvalidBranchSeq            = errors.New("invalid node sequence")
	ErrInvalidBranch               = errors.New("invalid branch")
	ErrBranchRootNotFound          = errors.New("branch root not found")
	ErrDefaultBranchNotSpecified   = errors.New("default branch not specified")
	ErrUnableToDecodeNodeSignature = errors.New("unable to decode node signature")
	ErrUnableToDecodeNodePubKey    = errors.New("unable to decode node pubkey")
	ErrUnableToDecodeNodeHash      = errors.New("unable to decode node hash")
	ErrNodeSignatureDoesNotMatch   = errors.New("node signature does not match")
)
