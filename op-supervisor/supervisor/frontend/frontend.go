package frontend

import (
	"context"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"

	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

type AdminBackend interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
}

type QueryBackend interface {
	CheckMessage(identifier types.Identifier, payloadHash common.Hash) (types.SafetyLevel, error)
	CheckBlock(chainID *hexutil.U256, blockHash common.Hash, blockNumber hexutil.Uint64) (types.SafetyLevel, error)
}

type Backend interface {
	AdminBackend
	QueryBackend
}

type QueryFrontend struct {
	Supervisor QueryBackend
}

// CheckMessage checks the safety-level of an individual message.
// The payloadHash references the hash of the message-payload of the message.
func (q *QueryFrontend) CheckMessage(identifier types.Identifier, payloadHash common.Hash) (types.SafetyLevel, error) {
	return q.Supervisor.CheckMessage(identifier, payloadHash)
}

// CheckBlock checks the safety-level of an L2 block as a whole.
func (q *QueryFrontend) CheckBlock(chainID *hexutil.U256, blockHash common.Hash, blockNumber hexutil.Uint64) (types.SafetyLevel, error) {
	return q.Supervisor.CheckBlock(chainID, blockHash, blockNumber)
}

type AdminFrontend struct {
	Supervisor Backend
}

// Start starts the service, if it was previously stopped.
func (a *AdminFrontend) Start(ctx context.Context) error {
	return a.Supervisor.Start(ctx)
}

// Stop stops the service, if it was previously started.
func (a *AdminFrontend) Stop(ctx context.Context) error {
	return a.Supervisor.Stop(ctx)
}
