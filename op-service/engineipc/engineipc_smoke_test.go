package engineipc_test

import (
	"bytes"
	"context"
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
	"github.com/ethereum/go-ethereum/params"
	"github.com/stretchr/testify/require"

	opservice "github.com/ethereum-optimism/optimism/op-service"
	"github.com/ethereum-optimism/optimism/op-service/engineipc"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// enginePath is the resolved op-reth-test-engine binary, provisioned once in TestMain.
var enginePath string

// TestMain provisions the Rust engine binary once for the package. It honours a prebuilt-binary
// path from the environment (as CI supplies) and otherwise builds it with cargo. It never skips:
// a missing binary or a failed build is a hard failure, so the smoke gate can't silently pass.
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
	// REQUIRE_RUST_ENGINE arms CI: with it set, a missing prebuilt-binary path is a hard error rather
	// than a slow cargo fallback, so a misconfigured job fails loudly instead of masking the config
	// bug behind a 20-minute rebuild.
	if os.Getenv("REQUIRE_RUST_ENGINE") != "" {
		return "", fmt.Errorf("REQUIRE_RUST_ENGINE is set but RUST_BINARY_PATH_OP_RETH_TEST_ENGINE is empty: refusing to fall back to a cargo build")
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

// A synthetic OP chain id — deliberately not OP Mainnet (10), whose genesis hash op-reth pins to
// the registry value (which the state-computed hash cannot reproduce). A synthetic genesis under
// id 10 would be indexed under that pinned hash, so op-geth's genesis hash and the engine's would
// diverge and the first forkchoice update would fail to find the parent.
const chainID = 901

// TestEngineSmoke drives the full sequencer flow over the Unix socket — spawn → forkchoiceUpdated
// with attributes → optest_includeTx → getPayload → newPayload → forkchoiceUpdated — twice, and
// asserts the result is a valid chain of two blocks (parent-linked, distinct hashes, ascending
// numbers) read back via eth_getBlockByNumber. This is the end-to-end gate for the RPC binary and
// the shared engineipc transport.
func TestEngineSmoke(t *testing.T) {
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
	cl := proc.Client()
	ctx := context.Background()

	// The genesis block anchors the chain.
	genesis := getBlock(t, cl, "earliest")
	require.EqualValues(t, 0, genesis.Number)

	// signTx builds a signed EIP-1559 value transfer from the funded account.
	signTx := func(nonce uint64) hexutil.Bytes {
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

	// buildBlock runs one sequencer round on top of parent and returns the new head hash.
	buildBlock := func(parent common.Hash, timestamp uint64, deposits []hexutil.Bytes, userTx hexutil.Bytes) common.Hash {
		fcState := eth.ForkchoiceState{HeadBlockHash: parent, SafeBlockHash: parent, FinalizedBlockHash: parent}
		attrs := payloadAttributes(timestamp, deposits)

		var fcuRes eth.ForkchoiceUpdatedResult
		require.NoError(t, cl.CallContext(ctx, &fcuRes, "engine_forkchoiceUpdatedV3", fcState, attrs),
			"forkchoiceUpdated(attrs)")
		require.Equal(t, eth.ExecutionValid, fcuRes.PayloadStatus.Status, "FCU status")
		require.NotNil(t, fcuRes.PayloadID, "payload id returned")

		var inc struct {
			TxHash  common.Hash `json:"txHash"`
			GasUsed uint64      `json:"gasUsed"`
		}
		require.NoError(t, cl.CallContext(ctx, &inc, "optest_includeTx", hexutil.Bytes(userTx)),
			"optest_includeTx")
		require.NotZero(t, inc.GasUsed, "user tx consumed gas")

		var envelope eth.ExecutionPayloadEnvelope
		require.NoError(t, cl.CallContext(ctx, &envelope, "engine_getPayloadV4", fcuRes.PayloadID),
			"getPayload")
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

	// Block 1: one forced deposit plus a user tx.
	block1 := buildBlock(common.Hash(genesis.Hash), 2, []hexutil.Bytes{depositTx()}, signTx(0))
	// Block 2 builds on block 1 with a second user tx (no deposit).
	block2 := buildBlock(block1, 4, nil, signTx(1))

	// A valid chain of two blocks on top of genesis, read back over eth_.
	h1 := getBlock(t, cl, "0x1")
	h2 := getBlock(t, cl, "0x2")
	require.EqualValues(t, 1, h1.Number)
	require.Equal(t, common.Hash(genesis.Hash), common.Hash(h1.ParentHash), "block1 parent is genesis")
	require.Equal(t, block1, common.Hash(h1.Hash))
	require.EqualValues(t, 2, h2.Number)
	require.Equal(t, block1, common.Hash(h2.ParentHash), "block2 parent is block1")
	require.Equal(t, block2, common.Hash(h2.Hash))
	require.NotEqual(t, block1, block2, "distinct block hashes")

	// The safe/finalized pointers moved with the head (forkchoice was applied over the socket).
	latest := getBlock(t, cl, "latest")
	require.Equal(t, block2, common.Hash(latest.Hash))
	safe := getBlock(t, cl, "safe")
	require.Equal(t, block2, common.Hash(safe.Hash))
}

// rpcBlock is the subset of an eth_getBlock* result the smoke test asserts on.
type rpcBlock struct {
	Number     hexutil.Uint64 `json:"number"`
	Hash       common.Hash    `json:"hash"`
	ParentHash common.Hash    `json:"parentHash"`
}

func getBlock(t *testing.T, cl interface {
	CallContext(ctx context.Context, result any, method string, args ...any) error
}, tag string) rpcBlock {
	t.Helper()
	var raw json.RawMessage
	require.NoError(t, cl.CallContext(context.Background(), &raw, "eth_getBlockByNumber", tag, false),
		"eth_getBlockByNumber(%s)", tag)
	require.NotEqual(t, "null", string(raw), "block %s present", tag)
	var block rpcBlock
	require.NoError(t, json.Unmarshal(raw, &block))
	return block
}

// payloadAttributes builds Karst-level attributes: withdrawals + parent-beacon-root (Ecotone/
// Canyon), the Holocene EIP-1559 params, and the Jovian minimum base fee, mirroring the Rust
// testsupport so block assembly succeeds.
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

// depositTx builds a minimal deposit (type 0x7E) mirroring the Rust testsupport's deposit_tx.
func depositTx() hexutil.Bytes {
	depositor := common.BytesToAddress([]byte{0xde})
	tx := types.NewTx(&types.DepositTx{
		From:  depositor,
		To:    &depositor,
		Value: big.NewInt(0),
		Gas:   21_000,
	})
	raw, err := tx.MarshalBinary()
	if err != nil {
		panic(err)
	}
	return raw
}

// writeGenesis builds an op-geth core.Genesis with all OP hardforks active at genesis (through
// Karst), the L1Block predeploy seeded with the L1-cost inputs, and funded given a spendable
// balance — the op-geth JSON the Rust engine parses into its chain spec. It mirrors the Rust
// testsupport chain so the two engines agree on block construction.
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

	genesis := &core.Genesis{
		Config:        config,
		GasLimit:      30_000_000,
		BaseFee:       big.NewInt(1_000_000_000),
		Difficulty:    big.NewInt(0),
		ExcessBlobGas: &zero,
		BlobGasUsed:   &zero,
		// Jovian EIP-1559 extra data: version 1, canyon params (denominator 250, elasticity 6),
		// minimum base fee 0.
		ExtraData: []byte{0x01, 0, 0, 0, 250, 0, 0, 0, 6, 0, 0, 0, 0, 0, 0, 0, 0},
		Alloc: types.GenesisAlloc{
			l1Block: {Nonce: 1, Balance: big.NewInt(0), Storage: storage},
			funded:  {Balance: new(big.Int).SetUint64(1_000_000_000_000_000_000)},
		},
	}

	data, err := json.Marshal(genesis)
	require.NoError(t, err)
	path := filepath.Join(t.TempDir(), "genesis.json")
	require.NoError(t, os.WriteFile(path, data, 0o644))
	return path
}
