package backend

import (
	"context"
	"errors"
	"io"
	"sync/atomic"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"

	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/frontend"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

type SupervisorBackend struct {
	started atomic.Bool

	// TODO(protocol-quest#287): collection of logdbs per chain
	// TODO(protocol-quest#288): collection of logdb updating services per chain
}

var _ frontend.Backend = (*SupervisorBackend)(nil)

var _ io.Closer = (*SupervisorBackend)(nil)

func NewSupervisorBackend() *SupervisorBackend {
	return &SupervisorBackend{}
}

func (su *SupervisorBackend) Start(ctx context.Context) error {
	if !su.started.CompareAndSwap(false, true) {
		return errors.New("already started")
	}
	// TODO(protocol-quest#288): start logdb updating services of all chains
	return nil
}

func (su *SupervisorBackend) Stop(ctx context.Context) error {
	if !su.started.CompareAndSwap(true, false) {
		return errors.New("already stopped")
	}
	// TODO(protocol-quest#288): stop logdb updating services of all chains
	return nil
}

func (su *SupervisorBackend) Close() error {
	// TODO(protocol-quest#288): close logdb of all chains
	return nil
}

func (su *SupervisorBackend) CheckMessage(identifier types.Identifier, payloadHash common.Hash) (types.SafetyLevel, error) {
	// TODO(protocol-quest#288): hook up to logdb lookup
	return types.CrossUnsafe, nil
}

func (su *SupervisorBackend) CheckBlock(chainID *hexutil.U256, blockHash common.Hash, blockNumber hexutil.Uint64) (types.SafetyLevel, error) {
	// TODO(protocol-quest#288): hook up to logdb lookup
	return types.CrossUnsafe, nil
}
