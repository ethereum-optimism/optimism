package node

import (
	"context"
	"errors"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/depset"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-node/node/safedb"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/engine"
	"github.com/ethereum-optimism/optimism/op-node/version"
	"github.com/ethereum-optimism/optimism/op-service/apis"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/rpc"
	opsigner "github.com/ethereum-optimism/optimism/op-service/signer"
)

type l2EthClient interface {
	InfoByHash(ctx context.Context, hash common.Hash) (eth.BlockInfo, error)
	// GetProof returns a proof of the account, it may return a nil result without error if the address was not found.
	// Optionally keys of the account storage trie can be specified to include with corresponding values in the proof.
	GetProof(ctx context.Context, address common.Address, storage []common.Hash, blockTag string) (*eth.AccountResult, error)
	OutputV0AtBlock(ctx context.Context, blockHash common.Hash) (*eth.OutputV0, error)
}

type driverClient interface {
	SyncStatus(ctx context.Context) (*eth.SyncStatus, error)
	BlockRefWithStatus(ctx context.Context, num uint64) (eth.L2BlockRef, *eth.SyncStatus, error)
	ResetDerivationPipeline(context.Context) error
	StartSequencer(ctx context.Context, blockHash common.Hash) error
	StopSequencer(context.Context) (common.Hash, error)
	SequencerActive(context.Context) (bool, error)
	OnUnsafeL2Payload(ctx context.Context, payload *eth.ExecutionPayloadEnvelope)
	OverrideLeader(ctx context.Context) error
	ConductorEnabled(ctx context.Context) (bool, error)
	SetRecoverMode(ctx context.Context, mode bool) error
}

type SafeDBReader interface {
	SafeHeadAtL1(ctx context.Context, l1BlockNum uint64) (l1 eth.BlockID, l2 eth.BlockID, err error)
}

type adminAPI struct {
	*rpc.CommonAdminAPI
	dr driverClient
}

var _ apis.OpnodeAdminServer = (*adminAPI)(nil)

func NewAdminAPI(dr driverClient, log log.Logger) *adminAPI {
	return &adminAPI{
		CommonAdminAPI: rpc.NewCommonAdminAPI(log),
		dr:             dr,
	}
}

func (n *adminAPI) ResetDerivationPipeline(ctx context.Context) error {
	return n.dr.ResetDerivationPipeline(ctx)
}

func (n *adminAPI) StartSequencer(ctx context.Context, blockHash common.Hash) error {
	return n.dr.StartSequencer(ctx, blockHash)
}

func (n *adminAPI) StopSequencer(ctx context.Context) (common.Hash, error) {
	return n.dr.StopSequencer(ctx)
}

func (n *adminAPI) SequencerActive(ctx context.Context) (bool, error) {
	return n.dr.SequencerActive(ctx)
}

// PostUnsafePayload is a special API that allows posting an unsafe payload to the L2 derivation pipeline.
// It should only be used by op-conductor for sequencer failover scenarios.
func (n *adminAPI) PostUnsafePayload(ctx context.Context, envelope *eth.ExecutionPayloadEnvelope) error {
	payload := envelope.ExecutionPayload
	if actual, ok := envelope.CheckBlockHash(); !ok {
		log.Error("payload has bad block hash", "bad_hash", payload.BlockHash.String(), "actual", actual.String())
		return fmt.Errorf("payload has bad block hash: %s, actual block hash is: %s", payload.BlockHash.String(), actual.String())
	}
	n.dr.OnUnsafeL2Payload(ctx, envelope)
	return nil
}

// OverrideLeader disables sequencer conductor interactions and allow sequencer to run in non-HA mode during disaster recovery scenarios.
func (n *adminAPI) OverrideLeader(ctx context.Context) error {
	return n.dr.OverrideLeader(ctx)
}

// ConductorEnabled returns true if the sequencer conductor is enabled.
func (n *adminAPI) ConductorEnabled(ctx context.Context) (bool, error) {
	return n.dr.ConductorEnabled(ctx)
}

func (n *adminAPI) SetRecoverMode(ctx context.Context, mode bool) error {
	return n.dr.SetRecoverMode(ctx, mode)
}

type nodeAPI struct {
	config *rollup.Config
	depSet depset.DependencySet
	client l2EthClient
	dr     driverClient
	safeDB SafeDBReader
	log    log.Logger
}

var _ apis.RollupNodeServer = (*nodeAPI)(nil)

func NewNodeAPI(config *rollup.Config, depSet depset.DependencySet, l2Client l2EthClient, dr driverClient, safeDB SafeDBReader, log log.Logger) *nodeAPI {
	return &nodeAPI{
		config: config,
		depSet: depSet,
		client: l2Client,
		dr:     dr,
		safeDB: safeDB,
		log:    log,
	}
}

func (n *nodeAPI) OutputAtBlock(ctx context.Context, number hexutil.Uint64) (*eth.OutputResponse, error) {
	ref, status, err := n.dr.BlockRefWithStatus(ctx, uint64(number))
	if err != nil {
		return nil, fmt.Errorf("failed to get L2 block ref with sync status: %w", err)
	}

	// OutputV0AtBlock uses the WithdrawalsRoot in the block header as the value for the
	// output MessagePasserStorageRoot, if Isthmus hard fork has activated.
	output, err := n.client.OutputV0AtBlock(ctx, ref.Hash)
	if err != nil {
		return nil, fmt.Errorf("failed to get L2 output at block %s: %w", ref, err)
	}
	return &eth.OutputResponse{
		Version:               output.Version(),
		OutputRoot:            eth.OutputRoot(output),
		BlockRef:              ref,
		WithdrawalStorageRoot: common.Hash(output.MessagePasserStorageRoot),
		StateRoot:             common.Hash(output.StateRoot),
		Status:                status,
	}, nil
}

func (n *nodeAPI) SafeHeadAtL1Block(ctx context.Context, number hexutil.Uint64) (*eth.SafeHeadResponse, error) {
	l1Block, safeHead, err := n.safeDB.SafeHeadAtL1(ctx, uint64(number))
	if errors.Is(err, safedb.ErrNotFound) {
		return nil, err
	} else if err != nil {
		return nil, fmt.Errorf("failed to get safe head at l1 block %s: %w", number, err)
	}
	return &eth.SafeHeadResponse{
		L1Block:  l1Block,
		SafeHead: safeHead,
	}, nil
}

func (n *nodeAPI) SyncStatus(ctx context.Context) (*eth.SyncStatus, error) {
	return n.dr.SyncStatus(ctx)
}

func (n *nodeAPI) RollupConfig(_ context.Context) (*rollup.Config, error) {
	return n.config, nil
}

func (n *nodeAPI) DependencySet(_ context.Context) (depset.DependencySet, error) {
	if n.depSet != nil {
		return n.depSet, nil
	}
	return nil, ethereum.NotFound
}

func (n *nodeAPI) Version(ctx context.Context) (string, error) {
	return version.Version + "-" + version.Meta, nil
}

type opstackAPI struct {
	engine    engine.RollupAPI
	publisher apis.PublishAPI
}

func NewOpstackAPI(eng engine.RollupAPI, publisher apis.PublishAPI) *opstackAPI {
	return &opstackAPI{
		engine:    eng,
		publisher: publisher,
	}
}

func (a *opstackAPI) OpenBlockV1(ctx context.Context, parent eth.BlockID, attrs *eth.PayloadAttributes) (eth.PayloadInfo, error) {
	return a.engine.OpenBlock(ctx, parent, attrs)
}

func (a *opstackAPI) CancelBlockV1(ctx context.Context, id eth.PayloadInfo) error {
	return a.engine.CancelBlock(ctx, id)
}

func (a *opstackAPI) SealBlockV1(ctx context.Context, id eth.PayloadInfo) (*eth.ExecutionPayloadEnvelope, error) {
	return a.engine.SealBlock(ctx, id)
}

func (a *opstackAPI) CommitBlockV1(ctx context.Context, envelope *opsigner.SignedExecutionPayloadEnvelope) error {
	return a.engine.CommitBlock(ctx, envelope)
}

func (a *opstackAPI) PublishBlockV1(ctx context.Context, signed *opsigner.SignedExecutionPayloadEnvelope) error {
	return a.publisher.PublishBlock(ctx, signed)
}
