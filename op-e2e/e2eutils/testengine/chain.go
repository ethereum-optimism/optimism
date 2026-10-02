package testengine

import (
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"maps"
	"math/big"
	"os"
	"path/filepath"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"

	opeip1559 "github.com/ethereum-optimism/optimism/op-core/eip1559"
	opparams "github.com/ethereum-optimism/optimism/op-core/params"
	"github.com/ethereum-optimism/optimism/op-core/predeploys"
	optypes "github.com/ethereum-optimism/optimism/op-core/types"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// ChainID is the chain id of the chains WriteGenesis builds. It is deliberately not OP Mainnet's
// (10): op-reth pins that chain's genesis hash to the registry value, which a synthetic genesis
// cannot reproduce, so the engine's genesis hash would differ from the one the genesis describes.
const ChainID = 901

const (
	gasLimit           = 30_000_000
	eip1559Elasticity  = 6
	eip1559Denominator = 250
)

// ChainConfig returns the chain configuration of the chains WriteGenesis builds: every OP Stack
// fork through Karst active at genesis.
func ChainConfig() *opparams.ChainConfig {
	zero := uint64(0)
	return &opparams.ChainConfig{
		ChainID: big.NewInt(ChainID),
		Optimism: &opparams.OptimismConfig{
			EIP1559Elasticity:  eip1559Elasticity,
			EIP1559Denominator: eip1559Denominator,
		},
		BedrockBlock: big.NewInt(0),
		RegolithTime: &zero,
		CanyonTime:   &zero,
		EcotoneTime:  &zero,
		FjordTime:    &zero,
		GraniteTime:  &zero,
		HoloceneTime: &zero,
		IsthmusTime:  &zero,
		JovianTime:   &zero,
		KarstTime:    &zero,
	}
}

// WriteGenesis writes the genesis of a ChainConfig chain to a temporary file and returns its path.
// The genesis holds alloc plus the L1Block predeploy seeded with the L1-cost inputs, and mirrors
// the Rust engine's own test chain so the two agree on block construction.
func WriteGenesis(t testing.TB, alloc types.GenesisAlloc) string {
	t.Helper()
	cfg := ChainConfig()
	zero := uint64(0)
	genesisAlloc := types.GenesisAlloc{
		predeploys.L1BlockAddr: {Nonce: 1, Balance: big.NewInt(0), Storage: map[common.Hash]common.Hash{
			common.BigToHash(big.NewInt(1)): common.BigToHash(big.NewInt(1_000_000_000)),
			common.BigToHash(big.NewInt(7)): common.BigToHash(big.NewInt(1)),
			common.BigToHash(big.NewInt(3)): common.HexToHash("0x0000000000000000000000000000000000001db0000d27300000000000000005"),
		}},
	}
	maps.Copy(genesisAlloc, alloc)
	genesis := &core.Genesis{
		Config:        cfg.GethChainConfig(),
		GasLimit:      gasLimit,
		BaseFee:       big.NewInt(1_000_000_000),
		Difficulty:    big.NewInt(0),
		ExcessBlobGas: &zero,
		BlobGasUsed:   &zero,
		ExtraData:     opeip1559.EncodeJovianExtraData(eip1559Denominator, eip1559Elasticity, 0),
		Alloc:         genesisAlloc,
	}

	data, err := genesisJSON(genesis, cfg)
	require.NoError(t, err)
	path := filepath.Join(t.TempDir(), "genesis.json")
	require.NoError(t, os.WriteFile(path, data, 0o644))
	return path
}

// genesisJSON encodes genesis in the op-geth genesis JSON the engine parses, with the OP Stack
// fields of its "config" object taken from cfg: go-ethereum's ChainConfig has no OP Stack fields
// of its own, so genesis.Config only carries the Ethereum fork schedule.
func genesisJSON(genesis *core.Genesis, cfg *opparams.ChainConfig) ([]byte, error) {
	raw, err := json.Marshal(genesis)
	if err != nil {
		return nil, err
	}
	var doc map[string]json.RawMessage
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, err
	}
	var config map[string]json.RawMessage
	if err := json.Unmarshal(doc["config"], &config); err != nil {
		return nil, err
	}
	opRaw, err := json.Marshal(cfg)
	if err != nil {
		return nil, err
	}
	var opConfig map[string]json.RawMessage
	if err := json.Unmarshal(opRaw, &opConfig); err != nil {
		return nil, err
	}
	maps.Copy(config, opConfig)
	if doc["config"], err = json.Marshal(config); err != nil {
		return nil, err
	}
	return json.Marshal(doc)
}

// PayloadAttributes returns Karst-level payload attributes at timestamp that force-include
// deposits: withdrawals and parent beacon root (Canyon, Ecotone), the Holocene EIP-1559 params and
// the Jovian minimum base fee, as the Rust engine's own tests build them.
func PayloadAttributes(timestamp uint64, deposits []hexutil.Bytes) *eth.PayloadAttributes {
	gas := eth.Uint64Quantity(gasLimit)
	minBaseFee := uint64(0)
	txs := make([]eth.Data, len(deposits))
	for i, d := range deposits {
		txs[i] = eth.Data(d)
	}
	return &eth.PayloadAttributes{
		Timestamp:             eth.Uint64Quantity(timestamp),
		PrevRandao:            eth.Bytes32{},
		SuggestedFeeRecipient: common.Address{},
		Withdrawals:           &types.Withdrawals{},
		ParentBeaconBlockRoot: &common.Hash{},
		Transactions:          txs,
		NoTxPool:              false,
		GasLimit:              &gas,
		EIP1559Params:         &eth.Bytes8{},
		MinBaseFee:            &minBaseFee,
	}
}

// DepositTx returns an encoded minimal deposit transaction; seed sets its source hash, so deposits
// with different seeds are distinct transactions.
func DepositTx(seed byte) hexutil.Bytes {
	depositor := common.BytesToAddress([]byte{0xde})
	tx := &optypes.DepositTx{
		SourceHash: common.BytesToHash([]byte{seed}),
		From:       depositor,
		To:         &depositor,
		Value:      big.NewInt(0),
		Gas:        21_000,
	}
	raw, err := tx.MarshalBinary()
	if err != nil {
		panic(err)
	}
	return raw
}

// SignTx returns an encoded, signed EIP-1559 value transfer from key's account at nonce.
func SignTx(t testing.TB, key *ecdsa.PrivateKey, nonce uint64) hexutil.Bytes {
	t.Helper()
	tx, err := types.SignNewTx(key, types.LatestSignerForChainID(big.NewInt(ChainID)), &types.DynamicFeeTx{
		ChainID:   big.NewInt(ChainID),
		Nonce:     nonce,
		GasTipCap: big.NewInt(0),
		GasFeeCap: big.NewInt(10_000_000_000),
		Gas:       21_000,
		To:        &common.Address{},
	})
	require.NoError(t, err)
	raw, err := tx.MarshalBinary()
	require.NoError(t, err)
	return raw
}

// BuildBlock runs one sequencer round over the engine API on top of parent — forkchoice update
// with attributes, optest_includeTx of userTx (if not nil), getPayload, newPayload, and a
// forkchoice update that makes the new block head, safe and finalized — and returns its hash.
func BuildBlock(t testing.TB, cl *rpc.Client, parent common.Hash, timestamp uint64, deposits []hexutil.Bytes, userTx hexutil.Bytes) common.Hash {
	t.Helper()
	ctx := context.Background()
	fcState := eth.ForkchoiceState{HeadBlockHash: parent, SafeBlockHash: parent, FinalizedBlockHash: parent}

	var fcuRes eth.ForkchoiceUpdatedResult
	require.NoError(t, cl.CallContext(ctx, &fcuRes, "engine_forkchoiceUpdatedV3", fcState, PayloadAttributes(timestamp, deposits)),
		"forkchoiceUpdated(attrs)")
	require.Equal(t, eth.ExecutionValid, fcuRes.PayloadStatus.Status, "FCU status")
	require.NotNil(t, fcuRes.PayloadID, "payload id returned")

	if userTx != nil {
		var inc struct {
			GasUsed uint64 `json:"gasUsed"`
		}
		require.NoError(t, cl.CallContext(ctx, &inc, "optest_includeTx", userTx), "optest_includeTx")
		require.NotZero(t, inc.GasUsed, "user tx consumed gas")
	}

	var envelope eth.ExecutionPayloadEnvelope
	require.NoError(t, cl.CallContext(ctx, &envelope, "engine_getPayloadV4", fcuRes.PayloadID), "getPayload")
	require.NotNil(t, envelope.ExecutionPayload)

	var status eth.PayloadStatusV1
	require.NoError(t, cl.CallContext(ctx, &status, "engine_newPayloadV4",
		envelope.ExecutionPayload, []common.Hash{}, envelope.ParentBeaconBlockRoot, []hexutil.Bytes{}),
		"newPayload")
	require.Equal(t, eth.ExecutionValid, status.Status, "newPayload status: %+v", status)
	require.NotNil(t, status.LatestValidHash)
	head := *status.LatestValidHash
	require.Equal(t, common.Hash(envelope.ExecutionPayload.BlockHash), head, "newPayload head == built block")

	var advanced eth.ForkchoiceUpdatedResult
	newState := eth.ForkchoiceState{HeadBlockHash: head, SafeBlockHash: head, FinalizedBlockHash: head}
	require.NoError(t, cl.CallContext(ctx, &advanced, "engine_forkchoiceUpdatedV3", newState, nil),
		"forkchoiceUpdated(advance)")
	require.Equal(t, eth.ExecutionValid, advanced.PayloadStatus.Status, "advance FCU status")
	return head
}

// Block is the subset of an eth_getBlockByNumber result these helpers read.
type Block struct {
	Number     hexutil.Uint64 `json:"number"`
	Hash       common.Hash    `json:"hash"`
	ParentHash common.Hash    `json:"parentHash"`
}

// GetBlock reads the block at tag (a number or a label such as "latest") and fails the test if it
// does not exist.
func GetBlock(t testing.TB, cl *rpc.Client, tag string) Block {
	t.Helper()
	var raw json.RawMessage
	require.NoError(t, cl.CallContext(context.Background(), &raw, "eth_getBlockByNumber", tag, false),
		"eth_getBlockByNumber(%s)", tag)
	require.NotEqual(t, "null", string(raw), "block %s present", tag)
	var block Block
	require.NoError(t, json.Unmarshal(raw, &block))
	return block
}
