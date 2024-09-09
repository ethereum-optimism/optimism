package versions

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/ethereum-optimism/optimism/cannon/mipsevm"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm/multithreaded"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm/singlethreaded"
	"github.com/ethereum-optimism/optimism/cannon/serialize"
	"github.com/ethereum-optimism/optimism/op-service/jsonutil"
)

type StateVersion uint8

const (
	VersionSingleThreaded StateVersion = iota
	VersionMultiThreaded
)

var (
	ErrUnknownVersion   = errors.New("unknown version")
	ErrJsonNotSupported = errors.New("json not supported")
)

func LoadStateFromFile(path string) (*VersionedState, error) {
	if !serialize.IsBinaryFile(path) {
		// Always use singlethreaded for JSON states
		state, err := jsonutil.LoadJSON[singlethreaded.State](path)
		if err != nil {
			return nil, err
		}
		return NewFromState(state)
	}
	return serialize.LoadSerializedBinary[VersionedState](path)
}

func NewFromState(state mipsevm.FPVMState) (*VersionedState, error) {
	switch state := state.(type) {
	case *singlethreaded.State:
		return &VersionedState{
			Version:   VersionSingleThreaded,
			FPVMState: state,
		}, nil
	case *multithreaded.State:
		return &VersionedState{
			Version:   VersionMultiThreaded,
			FPVMState: state,
		}, nil
	default:
		return nil, fmt.Errorf("%w: %T", ErrUnknownVersion, state)
	}
}

// VersionedState deserializes a FPVMState and implements VersionedState based on the version of that state.
// It does this based on the version byte read in Deserialize
type VersionedState struct {
	Version StateVersion
	mipsevm.FPVMState
}

func (s *VersionedState) Serialize(w io.Writer) error {
	bout := serialize.NewBinaryWriter(w)
	if err := bout.WriteUInt(s.Version); err != nil {
		return err
	}
	return s.FPVMState.Serialize(w)
}

func (s *VersionedState) Deserialize(in io.Reader) error {
	bin := serialize.NewBinaryReader(in)
	if err := bin.ReadUInt(&s.Version); err != nil {
		return err
	}

	switch s.Version {
	case VersionSingleThreaded:
		state := &singlethreaded.State{}
		if err := state.Deserialize(in); err != nil {
			return err
		}
		s.FPVMState = state
		return nil
	case VersionMultiThreaded:
		state := &multithreaded.State{}
		if err := state.Deserialize(in); err != nil {
			return err
		}
		s.FPVMState = state
		return nil
	default:
		return fmt.Errorf("%w: %d", ErrUnknownVersion, s.Version)
	}
}

// MarshalJSON marshals the underlying state without adding version prefix.
// JSON states are always assumed to be single threaded
func (s *VersionedState) MarshalJSON() ([]byte, error) {
	if s.Version != VersionSingleThreaded {
		return nil, fmt.Errorf("%w for type %T", ErrJsonNotSupported, s.FPVMState)
	}
	return json.Marshal(s.FPVMState)
}
