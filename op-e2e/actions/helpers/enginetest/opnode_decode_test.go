package enginetest

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	gethlog "github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"

	opservice "github.com/ethereum-optimism/optimism/op-service"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/engineipc"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/sources"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
)

// enginePath is the resolved op-reth-test-engine binary, provisioned once in TestMain.
var enginePath string

// TestMain provisions the Rust engine binary once for the package. It honours a prebuilt-binary
// path from the environment (as CI supplies) and otherwise builds it with cargo. It never skips:
// a missing binary or a failed build is a hard failure, so the switch gate can't silently pass.
func TestMain(m *testing.M) {
	path, err := provisionEngine()
	if err != nil {
		panic(fmt.Sprintf("provision op-reth-test-engine: %v", err))
	}
	enginePath = path
	os.Exit(m.Run())
}

func provisionEngine() (string, error) {
	if override := os.Getenv("RUST_BINARY_PATH_OP_RETH_TEST_ENGINE"); override != "" {
		if _, err := os.Stat(override); err != nil {
			return "", fmt.Errorf("RUST_BINARY_PATH_OP_RETH_TEST_ENGINE=%s: %w", override, err)
		}
		return override, nil
	}
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	root, err := opservice.FindMonorepoRoot(cwd)
	if err != nil {
		return "", err
	}
	rustDir := filepath.Join(root, "rust")
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, "cargo", "build", "-p", "op-reth-test-engine", "--bin", "op-reth-test-engine")
	cmd.Dir = rustDir
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("cargo build op-reth-test-engine (in %s): %w", rustDir, err)
	}
	return filepath.Join(rustDir, "target", "debug", "op-reth-test-engine"), nil
}

const chainID = 901

// codeAddr is a genesis-seeded account carrying bytecode, so eth_getCode has something non-empty to
// return. seededCode is PUSH1 1, PUSH1 0, SSTORE, STOP — never executed, just present as code.
var (
	codeAddr   = common.HexToAddress("0x00000000000000000000000000000000c0de5eed")
	seededCode = []byte{0x60, 0x01, 0x60, 0x00, 0x55, 0x00}

	// messagePasserAddr is the L2ToL1MessagePasser predeploy — its storage root is the Isthmus
	// withdrawals root. The literal avoids an op-core import (the switch stays on op-geth types).
	messagePasserAddr = common.HexToAddress("0x4200000000000000000000000000000000000016")
)

// TestOpNodeDecodesRethBlocks is the round-1 de-risk gate for the op-e2e/actions EL switch: it
// proves that op-node's real client stack (op-service/sources.EthClient) can reconstruct an
// ExecutionPayload and re-derive the block hash from the full-transaction blocks the Rust engine
// serves over the socket. Because DefaultEthClientConfig sets TrustRPC=false, a successful
// PayloadByNumber/PayloadByHash reconstructs the payload from the RPC JSON (header + every full
// transaction re-encoded to RLP, deposits included) and verifies the recomputed block hash against
// the engine's — the exact round-trip the switch relies on.
func TestOpNodeDecodesRethBlocks(t *testing.T) {
	key, err := crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
	require.NoError(t, err)
	funded := crypto.PubkeyToAddress(key.PublicKey)
	signer := types.LatestSignerForChainID(big.NewInt(chainID))

	genesisPath := writeGenesis(t, funded)

	var logs bytes.Buffer
	proc, err := engineipc.Spawn(enginePath, []string{"--genesis", genesisPath}, &logs, nil)
	require.NoError(t, err, "spawn engine\n%s", logs.String())
	defer func() {
		proc.Close()
		if t.Failed() {
			t.Logf("engine stderr:\n%s", logs.String())
		}
	}()
	ctx := context.Background()
	raw := proc.Client()

	// Drive two sequencer rounds directly over the socket to populate the chain. Every L2 block
	// leads with a deposit (op-node classifies L2 blocks by a leading deposit tx), then a user tx.
	genesisHash := blockHash(t, raw, "earliest")
	block1 := buildBlock(t, raw, genesisHash, 2, []hexutil.Bytes{depositTx(1)}, signTx(t, key, signer, 0))
	block2 := buildBlock(t, raw, block1, 4, []hexutil.Bytes{depositTx(2)}, signTx(t, key, signer, 1))

	// Now read the chain back through op-node's decoder, which reconstructs + verifies.
	logger := testlog.Logger(t, gethlog.LevelDebug)
	ethCl, err := sources.NewEthClient(
		client.NewBaseRPCClient(raw), logger, nil, sources.DefaultEthClientConfig(10))
	require.NoError(t, err)

	// PayloadByNumber round-trips the full block through op-node with block-hash verification on.
	env1, err := ethCl.PayloadByNumber(ctx, 1)
	require.NoError(t, err, "op-node PayloadByNumber(1) must reconstruct + verify the reth block")
	require.Equal(t, block1, common.Hash(env1.ExecutionPayload.BlockHash))
	require.EqualValues(t, 1, uint64(env1.ExecutionPayload.BlockNumber))
	require.Equal(t, genesisHash, common.Hash(env1.ExecutionPayload.ParentHash))
	require.Len(t, env1.ExecutionPayload.Transactions, 2, "deposit + user tx")

	// Explicit block-hash self-check: the payload op-node reconstructed hashes back to what the
	// engine reported (belt-and-braces on top of the TrustRPC=false verification above).
	got, ok := env1.CheckBlockHash()
	require.True(t, ok, "reconstructed payload hash %s != engine hash %s", got, block1)

	// PayloadByHash follows the same path keyed by hash.
	env1ByHash, err := ethCl.PayloadByHash(ctx, block1)
	require.NoError(t, err, "op-node PayloadByHash must reconstruct + verify")
	require.Equal(t, block1, common.Hash(env1ByHash.ExecutionPayload.BlockHash))

	env2, err := ethCl.PayloadByNumber(ctx, 2)
	require.NoError(t, err)
	require.Equal(t, block2, common.Hash(env2.ExecutionPayload.BlockHash))
	require.Equal(t, block1, common.Hash(env2.ExecutionPayload.ParentHash), "block2 parent is block1")

	// The full transaction bodies decode correctly: the first tx of block 1 is a deposit (type
	// 0x7E) — proving the deposit-tx JSON round-trips, the highest-risk part of the serialization.
	info, txs, err := ethCl.InfoAndTxsByHash(ctx, block1)
	require.NoError(t, err)
	require.Equal(t, block1, info.Hash())
	require.Len(t, txs, 2)
	require.Equal(t, uint8(types.DepositTxType), txs[0].Type(), "first tx is the L1-info/forced deposit")
	require.Equal(t, uint8(types.DynamicFeeTxType), txs[1].Type(), "second tx is the user 1559 tx")

	// eth_chainId is served for op-node's ChainID().
	cid, err := ethCl.ChainID(ctx)
	require.NoError(t, err)
	require.EqualValues(t, chainID, cid.Uint64())
}

// --- socket drive helpers (mirror the engineipc smoke test) ---

func signTx(t *testing.T, key *ecdsa.PrivateKey, signer types.Signer, nonce uint64) hexutil.Bytes {
	t.Helper()
	tx, err := types.SignNewTx(key, signer, &types.DynamicFeeTx{
		ChainID:   big.NewInt(chainID),
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

func buildBlock(t *testing.T, cl *rpc.Client, parent common.Hash, timestamp uint64, deposits []hexutil.Bytes, userTx hexutil.Bytes) common.Hash {
	t.Helper()
	ctx := context.Background()
	fcState := eth.ForkchoiceState{HeadBlockHash: parent, SafeBlockHash: parent, FinalizedBlockHash: parent}
	attrs := payloadAttributes(timestamp, deposits)

	var fcuRes eth.ForkchoiceUpdatedResult
	require.NoError(t, cl.CallContext(ctx, &fcuRes, "engine_forkchoiceUpdatedV3", fcState, attrs))
	require.Equal(t, eth.ExecutionValid, fcuRes.PayloadStatus.Status)
	require.NotNil(t, fcuRes.PayloadID)

	if userTx != nil {
		var inc struct {
			GasUsed uint64 `json:"gasUsed"`
		}
		require.NoError(t, cl.CallContext(ctx, &inc, "optest_includeTx", userTx))
		require.NotZero(t, inc.GasUsed)
	}

	var envelope eth.ExecutionPayloadEnvelope
	require.NoError(t, cl.CallContext(ctx, &envelope, "engine_getPayloadV4", fcuRes.PayloadID))
	require.NotNil(t, envelope.ExecutionPayload)

	var status eth.PayloadStatusV1
	require.NoError(t, cl.CallContext(ctx, &status, "engine_newPayloadV4",
		envelope.ExecutionPayload, []common.Hash{}, envelope.ParentBeaconBlockRoot, []hexutil.Bytes{}))
	require.Equal(t, eth.ExecutionValid, status.Status, "%+v", status)
	head := *status.LatestValidHash

	var advanced eth.ForkchoiceUpdatedResult
	newState := eth.ForkchoiceState{HeadBlockHash: head, SafeBlockHash: head, FinalizedBlockHash: head}
	require.NoError(t, cl.CallContext(ctx, &advanced, "engine_forkchoiceUpdatedV3", newState, nil))
	require.Equal(t, eth.ExecutionValid, advanced.PayloadStatus.Status)
	return head
}

func blockHash(t *testing.T, cl *rpc.Client, tag string) common.Hash {
	t.Helper()
	var block struct {
		Hash common.Hash `json:"hash"`
	}
	require.NoError(t, cl.CallContext(context.Background(), &block, "eth_getBlockByNumber", tag, false))
	return block.Hash
}

func payloadAttributes(timestamp uint64, deposits []hexutil.Bytes) *eth.PayloadAttributes {
	gasLimit := eth.Uint64Quantity(30_000_000)
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
		GasLimit:              &gasLimit,
		EIP1559Params:         &eth.Bytes8{},
		MinBaseFee:            &minBaseFee,
	}
}

func depositTx(seed byte) hexutil.Bytes {
	depositor := common.BytesToAddress([]byte{0xde})
	tx := types.NewTx(&types.DepositTx{
		SourceHash: common.BytesToHash([]byte{seed}),
		From:       depositor,
		To:         &depositor,
		Value:      big.NewInt(0),
		Gas:        21_000,
	})
	raw, err := tx.MarshalBinary()
	if err != nil {
		panic(err)
	}
	return raw
}

func writeGenesis(t *testing.T, funded common.Address) string {
	t.Helper()
	zero := uint64(0)
	config := &params.ChainConfig{
		ChainID:                 big.NewInt(chainID),
		HomesteadBlock:          big.NewInt(0),
		EIP150Block:             big.NewInt(0),
		EIP155Block:             big.NewInt(0),
		EIP158Block:             big.NewInt(0),
		ByzantiumBlock:          big.NewInt(0),
		ConstantinopleBlock:     big.NewInt(0),
		PetersburgBlock:         big.NewInt(0),
		IstanbulBlock:           big.NewInt(0),
		MuirGlacierBlock:        big.NewInt(0),
		BerlinBlock:             big.NewInt(0),
		LondonBlock:             big.NewInt(0),
		ArrowGlacierBlock:       big.NewInt(0),
		GrayGlacierBlock:        big.NewInt(0),
		MergeNetsplitBlock:      big.NewInt(0),
		BedrockBlock:            big.NewInt(0),
		ShanghaiTime:            &zero,
		CancunTime:              &zero,
		PragueTime:              &zero,
		RegolithTime:            &zero,
		CanyonTime:              &zero,
		EcotoneTime:             &zero,
		FjordTime:               &zero,
		GraniteTime:             &zero,
		HoloceneTime:            &zero,
		IsthmusTime:             &zero,
		JovianTime:              &zero,
		KarstTime:               &zero,
		TerminalTotalDifficulty: big.NewInt(0),
		Optimism: &params.OptimismConfig{
			EIP1559Elasticity:  6,
			EIP1559Denominator: 250,
		},
	}

	l1Block := common.HexToAddress("0x4200000000000000000000000000000000000015")
	storage := map[common.Hash]common.Hash{
		common.BigToHash(big.NewInt(1)): common.BigToHash(big.NewInt(1_000_000_000)),
		common.BigToHash(big.NewInt(7)): common.BigToHash(big.NewInt(1)),
		common.BigToHash(big.NewInt(3)): common.HexToHash("0x0000000000000000000000000000000000001db0000d27300000000000000005"),
	}

	// The Isthmus withdrawals-root is the L2ToL1MessagePasser predeploy's storage root; a real
	// deployed chain always has non-empty message-passer storage, so seed a slot here. Otherwise the
	// root equals the empty-trie hash and op-node's payload reconstruction misclassifies the block
	// as pre-Isthmus (dropping the withdrawals-root + requests-hash header fields).
	messagePasser := messagePasserAddr

	genesis := &core.Genesis{
		Config:        config,
		GasLimit:      30_000_000,
		BaseFee:       big.NewInt(1_000_000_000),
		Difficulty:    big.NewInt(0),
		ExcessBlobGas: &zero,
		BlobGasUsed:   &zero,
		ExtraData:     []byte{0x01, 0, 0, 0, 250, 0, 0, 0, 6, 0, 0, 0, 0, 0, 0, 0, 0},
		Alloc: types.GenesisAlloc{
			l1Block:       {Nonce: 1, Balance: big.NewInt(0), Storage: storage},
			messagePasser: {Nonce: 1, Balance: big.NewInt(0), Storage: map[common.Hash]common.Hash{common.BigToHash(big.NewInt(0)): common.BigToHash(big.NewInt(1))}},
			codeAddr:      {Balance: big.NewInt(0), Code: seededCode},
			funded:        {Balance: new(big.Int).SetUint64(1_000_000_000_000_000_000)},
		},
	}

	data, err := json.Marshal(genesis)
	require.NoError(t, err)
	path := filepath.Join(t.TempDir(), "genesis.json")
	require.NoError(t, os.WriteFile(path, data, 0o644))
	return path
}
