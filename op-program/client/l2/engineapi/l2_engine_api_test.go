package engineapi

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/op-chain-ops/genesis"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/beacon/engine"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	geth "github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/eth/ethconfig"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/stretchr/testify/require"
)

// TestNewPayloadV4 tests the NewPayloadV4 behavior with pre- and post-Isthmus payload
// attributes.
func TestNewPayloadV4(t *testing.T) {
	cases := []struct {
		isthmusTime       uint64
		blockTime         uint64
		expectedError     string
		nilWithdrawalRoot bool
	}{
		{6, 5, engine.UnsupportedFork.Error(), false}, // before isthmus
		{6, 8, "", false},                  // after isthmus
		{6, 8, "Invalid parameters", true}, // after isthmus, nil withdrawal root
	}
	logger, _ := testlog.CaptureLogger(t, log.LvlInfo)

	for _, c := range cases {
		genesis := createGenesisWithIsthmusTimeOffset(c.isthmusTime)
		ethCfg := &ethconfig.Config{
			NetworkId:   genesis.Config.ChainID.Uint64(),
			Genesis:     genesis,
			StateScheme: rawdb.HashScheme,
			NoPruning:   true,
		}
		backend := newStubBackendWithConfig(t, ethCfg)
		engineAPI := NewL2EngineAPI(logger, backend, nil)
		require.NotNil(t, engineAPI)
		genesisBlock := backend.GetHeaderByNumber(0)
		genesisHash := genesisBlock.Hash()
		attribs := createPayloadAttributes(genesisBlock.Time + c.blockTime)
		result, err := engineAPI.ForkchoiceUpdatedV3(context.Background(), &eth.ForkchoiceState{
			HeadBlockHash:      genesisHash,
			SafeBlockHash:      genesisHash,
			FinalizedBlockHash: genesisHash,
		}, attribs)
		require.NoError(t, err)
		require.EqualValues(t, engine.VALID, result.PayloadStatus.Status)
		require.NotNil(t, result.PayloadID)

		var envelope *eth.ExecutionPayloadEnvelope
		if c.blockTime >= c.isthmusTime {
			envelope, err = engineAPI.GetPayloadV4(context.Background(), *result.PayloadID)
		} else {
			envelope, err = engineAPI.GetPayloadV3(context.Background(), *result.PayloadID)
		}
		require.NoError(t, err)
		require.NotNil(t, envelope)

		if c.nilWithdrawalRoot {
			envelope.ExecutionPayload.WithdrawalsRoot = nil
		}

		newPayloadResult, err := engineAPI.NewPayloadV4(context.Background(), envelope.ExecutionPayload, []common.Hash{}, envelope.ParentBeaconBlockRoot, []hexutil.Bytes{})
		if c.expectedError != "" {
			require.ErrorContains(t, err, c.expectedError)
		} else {
			require.NoError(t, err)
			require.EqualValues(t, engine.VALID, newPayloadResult.Status)
		}
	}
}

func TestCreatedBlocksAreCached(t *testing.T) {
	logger, logs := testlog.CaptureLogger(t, log.LvlInfo)

	backend := newStubBackend(t)
	engineAPI := NewL2EngineAPI(logger, backend, nil)
	require.NotNil(t, engineAPI)
	genesis := backend.GetHeaderByNumber(0)
	genesisHash := genesis.Hash()
	attribs := createPayloadAttributes(genesis.Time + 1)
	result, err := engineAPI.ForkchoiceUpdatedV3(context.Background(), &eth.ForkchoiceState{
		HeadBlockHash:      genesisHash,
		SafeBlockHash:      genesisHash,
		FinalizedBlockHash: genesisHash,
	}, attribs)
	require.NoError(t, err)
	require.EqualValues(t, engine.VALID, result.PayloadStatus.Status)
	require.NotNil(t, result.PayloadID)

	envelope, err := engineAPI.GetPayloadV4(context.Background(), *result.PayloadID)
	require.NoError(t, err)
	require.NotNil(t, envelope)
	newPayloadResult, err := engineAPI.NewPayloadV4(context.Background(), envelope.ExecutionPayload, []common.Hash{}, envelope.ParentBeaconBlockRoot, []hexutil.Bytes{})
	require.NoError(t, err)
	require.EqualValues(t, engine.VALID, newPayloadResult.Status)

	foundLog := logs.FindLog(testlog.NewMessageFilter("Using existing beacon payload"))
	require.NotNil(t, foundLog)
	require.Equal(t, envelope.ExecutionPayload.BlockHash, foundLog.AttrValue("hash"))
}

func createPayloadAttributes(ts uint64) *eth.PayloadAttributes {
	eip1559Params := eth.Bytes8([]byte{0x1, 0x2, 0x3, 0x4, 0x5, 0x6, 0x7, 0x8})
	gasLimit := eth.Uint64Quantity(30e6)
	return &eth.PayloadAttributes{
		Timestamp:             eth.Uint64Quantity(ts),
		PrevRandao:            eth.Bytes32{0x11},
		SuggestedFeeRecipient: common.Address{0x33},
		Withdrawals:           &types.Withdrawals{},
		ParentBeaconBlockRoot: &common.Hash{0x22},
		GasLimit:              &gasLimit,
		EIP1559Params:         &eip1559Params,
	}
}

func newStubBackendWithConfig(t *testing.T, ethCfg *ethconfig.Config) *stubCachingBackend {
	nodeCfg := &node.Config{
		Name: "l2-geth",
	}
	n, err := node.New(nodeCfg)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = n.Close()
	})
	backend, err := geth.New(n, ethCfg)
	require.NoError(t, err)

	chain := backend.BlockChain()
	return &stubCachingBackend{EngineBackend: chain}
}

func newStubBackend(t *testing.T) *stubCachingBackend {
	genesis := createIsthmusGenesis()
	ethCfg := &ethconfig.Config{
		NetworkId:   genesis.Config.ChainID.Uint64(),
		Genesis:     genesis,
		StateScheme: rawdb.HashScheme,
		NoPruning:   true,
	}
	return newStubBackendWithConfig(t, ethCfg)
}

func createIsthmusGenesis() *core.Genesis {
	return createGenesisWithIsthmusTimeOffset(0)
}

func createGenesisWithIsthmusTimeOffset(forkTimeOffset uint64) *core.Genesis {
	deployConfig := &genesis.DeployConfig{
		L2InitializationConfig: genesis.L2InitializationConfig{
			DevDeployConfig: genesis.DevDeployConfig{
				FundDevAccounts: true,
			},
			L2GenesisBlockDeployConfig: genesis.L2GenesisBlockDeployConfig{
				L2GenesisBlockGasLimit:   30_000_000,
				L2GenesisBlockDifficulty: (*hexutil.Big)(big.NewInt(100)),
			},
			L2CoreDeployConfig: genesis.L2CoreDeployConfig{
				L1ChainID:   900,
				L2ChainID:   901,
				L2BlockTime: 2,
			},
			UpgradeScheduleDeployConfig: genesis.UpgradeScheduleDeployConfig{
				L1CancunTimeOffset: new(hexutil.Uint64),
			},
		},
	}

	deployConfig.ActivateForkAtOffset(rollup.Isthmus, forkTimeOffset)

	l1Genesis, err := genesis.NewL1Genesis(deployConfig)
	if err != nil {
		panic(err)
	}
	l2Genesis, err := genesis.NewL2Genesis(deployConfig, eth.BlockRefFromHeader(l1Genesis.ToBlock().Header()))
	if err != nil {
		panic(err)
	}

	return l2Genesis
}

type stubCachingBackend struct {
	EngineBackend
}

func (s *stubCachingBackend) AssembleAndInsertBlockWithoutSetHead(processor *BlockProcessor) (*types.Block, error) {
	block, _, err := processor.Assemble()
	if err != nil {
		return nil, err
	}
	if _, err := s.EngineBackend.InsertBlockWithoutSetHead(block, false); err != nil {
		return nil, err
	}
	return block, nil
}

func (s *stubCachingBackend) GetReceiptsByBlockHash(hash common.Hash) types.Receipts {
	panic("unsupported")
}

var _ CachingEngineBackend = (*stubCachingBackend)(nil)
