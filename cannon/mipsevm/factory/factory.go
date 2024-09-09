package factory

import (
	"errors"
	"fmt"
	"io"

	"github.com/ethereum-optimism/optimism/cannon/mipsevm"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm/multithreaded"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm/singlethreaded"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm/versions"
	"github.com/ethereum-optimism/optimism/cannon/serialize"
	"github.com/ethereum-optimism/optimism/op-service/jsonutil"
	"github.com/ethereum/go-ethereum/log"
)

var (
	ErrUnknownVersion = errors.New("unknown version")
	ErrWrongStateType = errors.New("wrong state type")
)

type VMFactory interface {
	CreateVM(logger log.Logger, po mipsevm.PreimageOracle, stdOut, stdErr io.Writer) mipsevm.FPVM
	State() mipsevm.FPVMState
}

func NewVMFactoryFromStateFile(path string) (VMFactory, error) {
	if !serialize.IsBinaryState(path) {
		// Always use singlethreaded for JSON states
		state, err := jsonutil.LoadJSON[singlethreaded.State](path)
		if err != nil {
			return nil, err
		}
		return &SingleThreadedFactory{state: state}, nil
	}
	versionedState, err := serialize.LoadSerializedBinary[VersionedState](path)
	if err != nil {
		return nil, err
	}

	switch versionedState.Version {
	case versions.VersionSingleThreaded:
		state, err := versionedState.AsSingleThreaded()
		if err != nil {
			return nil, err
		}
		return &SingleThreadedFactory{state: state}, nil
	case versions.VersionMultiThreaded:
		state, err := versionedState.AsMultiThreaded()
		if err != nil {
			return nil, err
		}
		return &MultiThreadedFactory{state: state}, nil
	default:
		return nil, fmt.Errorf("%w: %d", ErrUnknownVersion, versionedState.Version)
	}
}

type VersionedState struct {
	Version             versions.StateVersion
	singlethreadedState *singlethreaded.State
	multithreadedState  *multithreaded.State
}

func (s *VersionedState) Deserialize(in io.Reader) error {
	bin := serialize.NewBinaryReader(in)
	if err := bin.ReadUInt(&s.Version); err != nil {
		return err
	}
	switch s.Version {
	case versions.VersionSingleThreaded:
		s.singlethreadedState = &singlethreaded.State{}
		if err := s.singlethreadedState.Deserialize(in); err != nil {
			return err
		}
		return nil
	case versions.VersionMultiThreaded:
		s.multithreadedState = &multithreaded.State{}
		if err := s.multithreadedState.Deserialize(in); err != nil {
			return err
		}
		return nil
	default:
		return fmt.Errorf("%w: %d", ErrUnknownVersion, s.Version)
	}
}

func (s *VersionedState) AsSingleThreaded() (*singlethreaded.State, error) {
	if s.singlethreadedState == nil {
		return nil, ErrWrongStateType
	}
	return s.singlethreadedState, nil
}

func (s *VersionedState) AsMultiThreaded() (*multithreaded.State, error) {
	if s.multithreadedState == nil {
		return nil, ErrWrongStateType
	}
	return s.multithreadedState, nil
}
