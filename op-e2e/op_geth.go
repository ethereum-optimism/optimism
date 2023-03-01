package op_e2e

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"reflect"
	"testing"

	"github.com/ethereum-optimism/optimism/op-chain-ops/genesis"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils"
	"github.com/ethereum-optimism/optimism/op-node/client"
	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-node/sources"
	"github.com/ethereum-optimism/optimism/op-node/testlog"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	gn "github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"
)

var (
	// ErrForkChoiceUpdatedNotValid is returned when a forkChoiceUpdated returns a status other than Valid
	ErrForkChoiceUpdatedNotValid = errors.New("forkChoiceUpdated status was not valid")
	// ErrNewPayloadNotValid is returned when a newPayload call returns a status other than Valid, indicating the new block is invalid
	ErrNewPayloadNotValid = errors.New("newPayload status was not valid")
)

// OpGeth is an actor that functions as a l2 op-geth node
// It provides useful functions for advancing and querying the chain
type OpGeth struct {
	node          *gn.Node
	l2Engine      *sources.EngineClient
	L2Client      *ethclient.Client
	SystemConfig  eth.SystemConfig
	L1ChainConfig *params.ChainConfig
	L2ChainConfig *params.ChainConfig
	L1Head        eth.BlockInfo
	L2Head        *eth.ExecutionPayload
	sequenceNum   uint64
}

func NewOpGeth(t *testing.T, ctx context.Context, cfg *SystemConfig) (*OpGeth, error) {
	logger := testlog.Logger(t, log.LvlCrit)
	l1Genesis, err := genesis.BuildL1DeveloperGenesis(cfg.DeployConfig)
	require.Nil(t, err)
	l1Block := l1Genesis.ToBlock()

	l2Genesis, err := genesis.BuildL2DeveloperGenesis(cfg.DeployConfig, l1Block)
	require.Nil(t, err)
	l2GenesisBlock := l2Genesis.ToBlock()

	rollupGenesis := rollup.Genesis{
		L1: eth.BlockID{
			Hash:   l1Block.Hash(),
			Number: l1Block.NumberU64(),
		},
		L2: eth.BlockID{
			Hash:   l2GenesisBlock.Hash(),
			Number: l2GenesisBlock.NumberU64(),
		},
		L2Time:       l2GenesisBlock.Time(),
		SystemConfig: e2eutils.SystemConfigFromDeployConfig(cfg.DeployConfig),
	}

	node, _, err := initL2Geth("l2", big.NewInt(int64(cfg.DeployConfig.L2ChainID)), l2Genesis, cfg.JWTFilePath)
	require.Nil(t, err)
	require.Nil(t, node.Start())

	auth := rpc.WithHTTPAuth(gn.NewJWTAuth(cfg.JWTSecret))
	l2Node, err := client.NewRPC(ctx, logger, node.WSAuthEndpoint(), auth)
	require.Nil(t, err)

	// Finally create the engine client
	l2Engine, err := sources.NewEngineClient(
		l2Node,
		logger,
		nil,
		sources.EngineClientDefaultConfig(&rollup.Config{Genesis: rollupGenesis}),
	)
	require.Nil(t, err)

	l2Client, err := ethclient.Dial(node.HTTPEndpoint())
	require.Nil(t, err)

	genesisPayload, err := eth.BlockAsPayload(l2GenesisBlock)

	require.Nil(t, err)
	return &OpGeth{
		node:          node,
		L2Client:      l2Client,
		l2Engine:      l2Engine,
		SystemConfig:  rollupGenesis.SystemConfig,
		L1ChainConfig: l1Genesis.Config,
		L2ChainConfig: l2Genesis.Config,
		L1Head:        l1Block,
		L2Head:        genesisPayload,
	}, nil
}

func (d *OpGeth) Close() {
	_ = d.node.Close()
	d.l2Engine.Close()
	d.L2Client.Close()
}

// AddL2Block Appends a new L2 block to the current chain including the specified transactions
// The L1Info transaction is automatically prepended to the created block
func (d *OpGeth) AddL2Block(ctx context.Context, txs ...*types.Transaction) (*eth.ExecutionPayload, error) {
	attrs, err := d.CreatePayloadAttributes(txs...)
	if err != nil {
		return nil, err
	}
	res, err := d.StartBlockBuilding(ctx, attrs)
	if err != nil {
		return nil, err
	}

	payload, err := d.l2Engine.GetPayload(ctx, *res.PayloadID)
	if err != nil {
		return nil, err
	}
	if !reflect.DeepEqual(payload.Transactions, attrs.Transactions) {
		return nil, errors.New("required transactions were not included")
	}

	status, err := d.l2Engine.NewPayload(ctx, payload)
	if err != nil {
		return nil, err
	}
	if status.Status != eth.ExecutionValid {
		return nil, fmt.Errorf("%w: %s", ErrNewPayloadNotValid, status.Status)
	}

	fc := eth.ForkchoiceState{
		HeadBlockHash: payload.BlockHash,
		SafeBlockHash: payload.BlockHash,
	}
	res, err = d.l2Engine.ForkchoiceUpdate(ctx, &fc, nil)
	if err != nil {
		return nil, err
	}
	if res.PayloadStatus.Status != eth.ExecutionValid {
		return nil, fmt.Errorf("%w: %s", ErrForkChoiceUpdatedNotValid, res.PayloadStatus.Status)
	}
	d.L2Head = payload
	d.sequenceNum = d.sequenceNum + 1
	return payload, nil
}

// StartBlockBuilding begins block building for the specified PayloadAttributes by sending a engine_forkChoiceUpdated call.
// The current L2Head is used as the parent of the new block.
// ErrForkChoiceUpdatedNotValid is returned if the forkChoiceUpdate call returns a status other than valid.
func (d *OpGeth) StartBlockBuilding(ctx context.Context, attrs *eth.PayloadAttributes) (*eth.ForkchoiceUpdatedResult, error) {
	fc := eth.ForkchoiceState{
		HeadBlockHash: d.L2Head.BlockHash,
		SafeBlockHash: d.L2Head.BlockHash,
	}
	res, err := d.l2Engine.ForkchoiceUpdate(ctx, &fc, attrs)
	if err != nil {
		return nil, err
	}
	if res.PayloadStatus.Status != eth.ExecutionValid {
		return nil, fmt.Errorf("%w: %s", ErrForkChoiceUpdatedNotValid, res.PayloadStatus.Status)
	}
	if res.PayloadID == nil {
		return nil, errors.New("forkChoiceUpdated returned nil PayloadID")
	}
	return res, nil
}

// CreatePayloadAttributes creates a valid PayloadAttributes containing a L1Info deposit transaction followed by the supplied transactions.
func (d *OpGeth) CreatePayloadAttributes(txs ...*types.Transaction) (*eth.PayloadAttributes, error) {
	timestamp := d.L2Head.Timestamp + 2
	regolith := d.L2ChainConfig.IsRegolith(uint64(timestamp))
	l1Info, err := derive.L1InfoDepositBytes(d.sequenceNum, d.L1Head, d.SystemConfig, regolith)
	if err != nil {
		return nil, err
	}

	var txBytes []hexutil.Bytes
	txBytes = append(txBytes, l1Info)
	for _, tx := range txs {
		bin, err := tx.MarshalBinary()
		if err != nil {
			return nil, fmt.Errorf("tx marshalling failed: %w", err)
		}
		txBytes = append(txBytes, bin)
	}
	attrs := eth.PayloadAttributes{
		Timestamp:    timestamp,
		Transactions: txBytes,
		NoTxPool:     true,
		GasLimit:     (*eth.Uint64Quantity)(&d.SystemConfig.GasLimit),
	}
	return &attrs, nil
}
