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
	"github.com/ethereum/go-ethereum/log"
)

type StateVersion uint8

const (
	VersionSingleThreaded StateVersion = iota
	VersionMultiThreaded
)

var (
	ErrUnknownVersion = errors.New("unknown version")
)

type VersionedState interface {
	mipsevm.FPVMState
	serialize.Serializable
	CreateVM(logger log.Logger, po mipsevm.PreimageOracle, stdOut, stdErr io.Writer) mipsevm.FPVM
}

func LoadStateFromFile(path string) (VersionedState, error) {
	if !serialize.IsBinaryState(path) {
		// Always use singlethreaded for JSON states
		state, err := jsonutil.LoadJSON[singlethreaded.State](path)
		if err != nil {
			return nil, err
		}
		return NewFromState(state)
	}
	return serialize.LoadSerializedBinary[versionedState](path)
}

func NewFromState(state mipsevm.FPVMState) (VersionedState, error) {
	switch state := state.(type) {
	case *singlethreaded.State:
		return &versionedState{
			version:        VersionSingleThreaded,
			VersionedState: &SingleThreadedState{state},
		}, nil
	case *multithreaded.State:
		return &versionedState{
			version:        VersionMultiThreaded,
			VersionedState: &MultiThreadedState{state},
		}, nil
	default:
		return nil, fmt.Errorf("%w: %T", ErrUnknownVersion, state)
	}
}

// versionedState deserializes a FPVMState and implements VersionedState based on the version of that state.
// It does this based on the version byte read in Deserialize
type versionedState struct {
	version StateVersion
	VersionedState
}

func (s *versionedState) Deserialize(in io.Reader) error {
	// Read the version byte and then create a multireader to allow the actual implementation to also read it
	// Allows the Serialize and Deserialize methods of the states to be exact inverses of each other.
	bin := serialize.NewBinaryReader(in)
	if err := bin.ReadUInt(&s.version); err != nil {
		return err
	}

	switch s.version {
	case VersionSingleThreaded:
		state := &singlethreaded.State{}
		if err := state.Deserialize(in); err != nil {
			return err
		}
		s.VersionedState = &SingleThreadedState{State: state}
		return nil
	case VersionMultiThreaded:
		state := &multithreaded.State{}
		if err := state.Deserialize(in); err != nil {
			return err
		}
		s.VersionedState = &MultiThreadedState{State: state}
		return nil
	default:
		return fmt.Errorf("%w: %d", ErrUnknownVersion, s.version)
	}
}

func (s *versionedState) Serialize(w io.Writer) error {
	bout := serialize.NewBinaryWriter(w)
	if err := bout.WriteUInt(s.version); err != nil {
		return err
	}
	return s.VersionedState.Serialize(w)
}

// MarshalJSON marshalls the underlying state without adding version prefix.
// JSON states are always assumed to be single threaded
func (s *versionedState) MarshalJSON() ([]byte, error) {
	if s.version != VersionSingleThreaded {
		return nil, fmt.Errorf("attempting to JSON marshal state of type %T, only single threaded states support JSON", s.VersionedState)
	}
	return json.Marshal(s.VersionedState)
}
