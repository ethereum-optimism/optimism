package frontend

import (
	"context"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

type AdminBackend interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	AddL2RPC(ctx context.Context, rpc string) error
}

type QueryBackend interface {
	CheckMessage(identifier types.Identifier, payloadHash common.Hash) (types.SafetyLevel, error)
	CheckMessages(messages []types.Message, minSafety types.SafetyLevel) error
	CheckBlock(chainID *hexutil.U256, blockHash common.Hash, blockNumber hexutil.Uint64) (types.SafetyLevel, error)
	DerivedFrom(ctx context.Context, chainID types.ChainID, blockHash common.Hash, blockNumber uint64) (eth.BlockRef, error)
}

type UpdatesBackend interface {
	UpdateLocalUnsafe(chainID types.ChainID, head eth.BlockRef)
	UpdateLocalSafe(chainID types.ChainID, derivedFrom eth.BlockRef, lastDerived eth.BlockRef)
	UpdateFinalizedL1(chainID types.ChainID, finalized eth.BlockRef)
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

// CheckMessage checks the safety-level of a collection of messages,
// and returns if the minimum safety-level is met for all messages.
func (q *QueryFrontend) CheckMessages(
	messages []types.Message,
	minSafety types.SafetyLevel) error {
	return q.Supervisor.CheckMessages(messages, minSafety)
}

func (q *QueryFrontend) UnsafeView(ctx context.Context, chainID types.ChainID, unsafe types.ReferenceView) (types.ReferenceView, error) {
	// TODO(#12358): attach to backend
	return types.ReferenceView{}, nil
}

func (q *QueryFrontend) SafeView(ctx context.Context, chainID types.ChainID, safe types.ReferenceView) (types.ReferenceView, error) {
	// TODO(#12358): attach to backend
	return types.ReferenceView{}, nil
}

func (q *QueryFrontend) Finalized(ctx context.Context, chainID types.ChainID) (eth.BlockID, error) {
	// TODO(#12358): attach to backend
	return eth.BlockID{}, nil
}

func (q *QueryFrontend) DerivedFrom(ctx context.Context, chainID types.ChainID, blockHash common.Hash, blockNumber uint64) (eth.BlockRef, error) {
	// TODO(#12358): attach to backend
	return eth.BlockRef{}, nil
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

// AddL2RPC adds a new L2 chain to the supervisor backend
func (a *AdminFrontend) AddL2RPC(ctx context.Context, rpc string) error {
	return a.Supervisor.AddL2RPC(ctx, rpc)
}

type UpdatesFrontend struct {
	Supervisor UpdatesBackend
}

func (u *UpdatesFrontend) UpdateLocalUnsafe(chainID types.ChainID, head eth.BlockRef) {
	u.Supervisor.UpdateLocalUnsafe(chainID, head)
}

func (u *UpdatesFrontend) UpdateLocalSafe(chainID types.ChainID, derivedFrom eth.BlockRef, lastDerived eth.BlockRef) {
	u.Supervisor.UpdateLocalSafe(chainID, derivedFrom, lastDerived)
}

func (u *UpdatesFrontend) UpdateFinalizedL1(chainID types.ChainID, finalized eth.BlockRef) {
	u.Supervisor.UpdateFinalizedL1(chainID, finalized)
}
