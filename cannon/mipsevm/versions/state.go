package versions

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/ethereum-optimism/optimism/cannon/mipsevm"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm/arch"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm/multithreaded"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm/singlethreaded"
	"github.com/ethereum-optimism/optimism/op-service/jsonutil"
	"github.com/ethereum-optimism/optimism/op-service/serialize"
)

type StateVersion uint8

const (
	// VersionSingleThreaded is the version of the Cannon STF found in op-contracts/v1.6.0 - https://github.com/ethereum-optimism/optimism/blob/op-contracts/v1.6.0/packages/contracts-bedrock/src/cannon/MIPS.sol
	VersionSingleThreaded StateVersion = iota
	VersionMultiThreaded
	// VersionSingleThreaded2 is based on VersionSingleThreaded with the addition of support for fcntl(F_GETFD) syscall
	VersionSingleThreaded2
	VersionMultiThreaded64
)

var (
	ErrUnknownVersion      = errors.New("unknown version")
	ErrJsonNotSupported    = errors.New("json not supported")
	ErrUnsupportedMipsArch = errors.New("mips architecture is not supported")
)

var StateVersionTypes = []StateVersion{VersionSingleThreaded, VersionMultiThreaded, VersionSingleThreaded2, VersionMultiThreaded64}

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
		if !arch.IsMips32 {
			return nil, ErrUnsupportedMipsArch
		}
		return &VersionedState{
			Version:   VersionSingleThreaded2,
			FPVMState: state,
		}, nil
	case *multithreaded.State:
		if arch.IsMips32 {
			return &VersionedState{
				Version:   VersionMultiThreaded,
				FPVMState: state,
			}, nil
		} else {
			return &VersionedState{
				Version:   VersionMultiThreaded64,
				FPVMState: state,
			}, nil
		}
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
	case VersionSingleThreaded2:
		if !arch.IsMips32 {
			return ErrUnsupportedMipsArch
		}
		state := &singlethreaded.State{}
		if err := state.Deserialize(in); err != nil {
			return err
		}
		s.FPVMState = state
		return nil
	case VersionMultiThreaded:
		if !arch.IsMips32 {
			return ErrUnsupportedMipsArch
		}
		state := &multithreaded.State{}
		if err := state.Deserialize(in); err != nil {
			return err
		}
		s.FPVMState = state
		return nil
	case VersionMultiThreaded64:
		if arch.IsMips32 {
			return ErrUnsupportedMipsArch
		}
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
	if !arch.IsMips32 {
		return nil, ErrUnsupportedMipsArch
	}
	return json.Marshal(s.FPVMState)
}

func (s StateVersion) String() string {
	switch s {
	case VersionSingleThreaded:
		return "singlethreaded"
	case VersionMultiThreaded:
		return "multithreaded"
	case VersionSingleThreaded2:
		return "singlethreaded-2"
	case VersionMultiThreaded64:
		return "multithreaded64"
	default:
		return "unknown"
	}
}

func ParseStateVersion(ver string) (StateVersion, error) {
	switch ver {
	case "singlethreaded":
		return VersionSingleThreaded, nil
	case "multithreaded":
		return VersionMultiThreaded, nil
	case "singlethreaded-2":
		return VersionSingleThreaded2, nil
	case "multithreaded64":
		return VersionMultiThreaded64, nil
	default:
		return StateVersion(0), errors.New("unknown state version")
	}
}
