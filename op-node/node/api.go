package node

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum-optimism/optimism/op-bindings/predeploys"
	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum-optimism/optimism/op-node/l2"
	"github.com/ethereum-optimism/optimism/op-node/metrics"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/driver"
	"github.com/ethereum-optimism/optimism/op-node/version"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
)

type l2EthClient interface {
	GetBlockHeader(ctx context.Context, blockTag string) (*types.Header, error)
	// GetProof returns a proof of the account, it may return a nil result without error if the address was not found.
	GetProof(ctx context.Context, address common.Address, blockTag string) (*l2.AccountResult, error)

	BlockByNumber(ctx context.Context, number *big.Int) (*types.Block, error)
	L2BlockRefByNumber(ctx context.Context, l2Num *big.Int) (eth.L2BlockRef, error)
	L2BlockRefByHash(ctx context.Context, l2Hash common.Hash) (eth.L2BlockRef, error)
}

type driverClient interface {
	SyncStatus(ctx context.Context) (*driver.SyncStatus, error)
}

type nodeAPI struct {
	config *rollup.Config
	client l2EthClient
	dr     driverClient
	log    log.Logger
	m      *metrics.Metrics
}

func newNodeAPI(config *rollup.Config, l2Client l2EthClient, dr driverClient, log log.Logger, m *metrics.Metrics) *nodeAPI {
	return &nodeAPI{
		config: config,
		client: l2Client,
		dr:     dr,
		log:    log,
		m:      m,
	}
}

func (n *nodeAPI) OutputAtBlock(ctx context.Context, number rpc.BlockNumber) ([]eth.Bytes32, error) {
	recordDur := n.m.RecordRPCServerRequest("optimism_outputAtBlock")
	defer recordDur()
	// TODO: rpc.BlockNumber doesn't support the "safe" tag. Need a new type

	head, err := n.client.GetBlockHeader(ctx, toBlockNumArg(number))
	if err != nil {
		n.log.Error("failed to get block", "err", err)
		return nil, err
	}
	if head == nil {
		return nil, ethereum.NotFound
	}

	proof, err := n.client.GetProof(ctx, predeploys.L2ToL1MessagePasserAddr, toBlockNumArg(number))
	if err != nil {
		n.log.Error("failed to get contract proof", "err", err)
		return nil, err
	}
	if proof == nil {
		return nil, ethereum.NotFound
	}
	// make sure that the proof (including storage hash) that we retrieved is correct by verifying it against the state-root
	if err := proof.Verify(head.Root); err != nil {
		n.log.Error("invalid withdrawal root detected in block", "stateRoot", head.Root, "blocknum", number, "msg", err)
		return nil, fmt.Errorf("invalid withdrawal root hash")
	}

	var l2OutputRootVersion eth.Bytes32 // it's zero for now
	l2OutputRoot := l2.ComputeL2OutputRoot(l2OutputRootVersion, head.Hash(), head.Root, proof.StorageHash)

	return []eth.Bytes32{l2OutputRootVersion, l2OutputRoot}, nil
}

func (n *nodeAPI) SyncStatus(ctx context.Context) (*driver.SyncStatus, error) {
	recordDur := n.m.RecordRPCServerRequest("optimism_syncStatus")
	defer recordDur()
	return n.dr.SyncStatus(ctx)
}

func (n *nodeAPI) RollupConfig(_ context.Context) (*rollup.Config, error) {
	recordDur := n.m.RecordRPCServerRequest("optimism_rollupConfig")
	defer recordDur()
	return n.config, nil
}

func (n *nodeAPI) Version(ctx context.Context) (string, error) {
	recordDur := n.m.RecordRPCServerRequest("optimism_version")
	defer recordDur()
	return version.Version + "-" + version.Meta, nil
}

func toBlockNumArg(number rpc.BlockNumber) string {
	if number == rpc.LatestBlockNumber {
		return "latest"
	}
	if number == rpc.PendingBlockNumber {
		return "pending"
	}
	return hexutil.EncodeUint64(uint64(number.Int64()))
}
