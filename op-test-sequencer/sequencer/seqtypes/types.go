package seqtypes

import (
	"errors"

	"github.com/google/uuid"

	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum-optimism/optimism/op-service/jsonrpc"
)

const maxIDLength = 100

var ErrInvalidID = errors.New("invalid ID")

type genericID string

func (id genericID) String() string {
	return string(id)
}

func (id genericID) MarshalText() ([]byte, error) {
	if len(id) > maxIDLength {
		return nil, ErrInvalidID
	}
	return []byte(id), nil
}

func (id *genericID) UnmarshalText(data []byte) error {
	if len(data) > maxIDLength {
		return ErrInvalidID
	}
	*id = genericID(data)
	return nil
}

// BuildJobID identifies a block-building job.
// Multiple alternative blocks may be built in parallel.
type BuildJobID genericID

func (id BuildJobID) String() string {
	return genericID(id).String()
}

func (id BuildJobID) MarshalText() ([]byte, error) {
	return genericID(id).MarshalText()
}

func (id *BuildJobID) UnmarshalText(data []byte) error {
	return (*genericID)(id).UnmarshalText(data)
}

type BuilderID genericID

func (id BuilderID) String() string {
	return genericID(id).String()
}

func (id BuilderID) MarshalText() ([]byte, error) {
	return genericID(id).MarshalText()
}

func (id *BuilderID) UnmarshalText(data []byte) error {
	return (*genericID)(id).UnmarshalText(data)
}

type SignerID genericID

func (id SignerID) String() string {
	return genericID(id).String()
}

func (id SignerID) MarshalText() ([]byte, error) {
	return genericID(id).MarshalText()
}

func (id *SignerID) UnmarshalText(data []byte) error {
	return (*genericID)(id).UnmarshalText(data)
}

type CommitterID genericID

func (id CommitterID) String() string {
	return genericID(id).String()
}

func (id CommitterID) MarshalText() ([]byte, error) {
	return genericID(id).MarshalText()
}

func (id *CommitterID) UnmarshalText(data []byte) error {
	return (*genericID)(id).UnmarshalText(data)
}

type PublisherID genericID

func (id PublisherID) String() string {
	return genericID(id).String()
}

func (id PublisherID) MarshalText() ([]byte, error) {
	return genericID(id).MarshalText()
}

func (id *PublisherID) UnmarshalText(data []byte) error {
	return (*genericID)(id).UnmarshalText(data)
}

type SequencerID genericID

func (id SequencerID) String() string {
	return genericID(id).String()
}

func (id SequencerID) MarshalText() ([]byte, error) {
	return genericID(id).MarshalText()
}

func (id *SequencerID) UnmarshalText(data []byte) error {
	return (*genericID)(id).UnmarshalText(data)
}

var (
	ErrGeneric                = &jsonrpc.Error{Code: -38500, Message: "sequencer error"}
	ErrUnknownKind            = &jsonrpc.Error{Code: -38501, Message: "unknown kind"}
	ErrUnknownBuilder         = &jsonrpc.Error{Code: -38502, Message: "unknown builder"}
	ErrNotImplemented         = &jsonrpc.Error{Code: -38503, Message: "not implemented"}
	ErrUnknownJob             = &jsonrpc.Error{Code: -38510, Message: "unknown job"}
	ErrConflictingJob         = &jsonrpc.Error{Code: -38511, Message: "conflicting job"}
	ErrNotSealed              = &jsonrpc.Error{Code: -38520, Message: "block not yet sealed"}
	ErrAlreadySealed          = &jsonrpc.Error{Code: -38521, Message: "block already sealed"}
	ErrUnsigned               = &jsonrpc.Error{Code: -38530, Message: "block not yet signed"}
	ErrAlreadySigned          = &jsonrpc.Error{Code: -38531, Message: "block already signed"}
	ErrUncommitted            = &jsonrpc.Error{Code: -38540, Message: "block not yet committed"}
	ErrAlreadyCommitted       = &jsonrpc.Error{Code: -38541, Message: "block already committed"}
	ErrSequencerInactive      = &jsonrpc.Error{Code: -38550, Message: "sequencer inactive"}
	ErrSequencerAlreadyActive = &jsonrpc.Error{Code: -38551, Message: "sequencer already active"}
	ErrBackendInactive        = &jsonrpc.Error{Code: -38560, Message: "backend inactive"}
	ErrBackendAlreadyStarted  = &jsonrpc.Error{Code: -38561, Message: "backend already started"}
)

func RandomJobID() BuildJobID {
	return BuildJobID("job-" + uuid.New().String())
}

type BuildOpts struct {
	// Parent block to build on top of
	Parent common.Hash `json:"parent"`

	// L1Origin overrides the L1 origin of the block.
	// Optional, by default the L1 origin of the parent block
	// is progressed when first allowed (respecting time invariants).
	L1Origin *common.Hash `json:"l1Origin,omitempty"`
}
