package frontend

import (
	"context"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
	"github.com/ethereum/go-ethereum/common"
)

type AdminBackend interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	AddL2RPC(ctx context.Context, rpc string) error
}

type QueryBackend interface {
	CheckMessage(identifier types.Identifier, payloadHash common.Hash) (types.SafetyLevel, error)
	CheckMessages(messages []types.Message, minSafety types.SafetyLevel) error
	DerivedFrom(ctx context.Context, chainID types.ChainID, derived eth.BlockID) (derivedFrom eth.BlockID, err error)
	UnsafeView(ctx context.Context, chainID types.ChainID, unsafe types.ReferenceView) (types.ReferenceView, error)
	SafeView(ctx context.Context, chainID types.ChainID, safe types.ReferenceView) (types.ReferenceView, error)
	Finalized(ctx context.Context, chainID types.ChainID) (eth.BlockID, error)
}

type UpdatesBackend interface {
	UpdateLocalUnsafe(chainID types.ChainID, head eth.BlockRef) error
	UpdateLocalSafe(chainID types.ChainID, derivedFrom eth.BlockRef, lastDerived eth.BlockRef) error
	UpdateFinalizedL1(chainID types.ChainID, finalized eth.BlockRef) error
}

type Backend interface {
	AdminBackend
	QueryBackend
	UpdatesBackend
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
	return q.Supervisor.UnsafeView(ctx, chainID, unsafe)
}

func (q *QueryFrontend) SafeView(ctx context.Context, chainID types.ChainID, safe types.ReferenceView) (types.ReferenceView, error) {
	return q.Supervisor.SafeView(ctx, chainID, safe)
}

func (q *QueryFrontend) Finalized(ctx context.Context, chainID types.ChainID) (eth.BlockID, error) {
	return q.Supervisor.Finalized(ctx, chainID)
}

func (q *QueryFrontend) DerivedFrom(ctx context.Context, chainID types.ChainID, derived eth.BlockID) (derivedFrom eth.BlockID, err error) {
	return q.Supervisor.DerivedFrom(ctx, chainID, derived)
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

func (u *UpdatesFrontend) UpdateLocalUnsafe(chainID types.ChainID, head eth.BlockRef) error {
	return u.Supervisor.UpdateLocalUnsafe(chainID, head)
}

func (u *UpdatesFrontend) UpdateLocalSafe(chainID types.ChainID, derivedFrom eth.BlockRef, lastDerived eth.BlockRef) error {
	return u.Supervisor.UpdateLocalSafe(chainID, derivedFrom, lastDerived)
}

func (u *UpdatesFrontend) UpdateFinalizedL1(chainID types.ChainID, finalized eth.BlockRef) error {
	return u.Supervisor.UpdateFinalizedL1(chainID, finalized)
}
