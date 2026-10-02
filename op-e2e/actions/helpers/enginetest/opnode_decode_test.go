package enginetest

import (
	"bytes"
	"context"
	"math/big"
	"sync"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-core/predeploys"
	optypes "github.com/ethereum-optimism/optimism/op-core/types"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/testengine"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum-optimism/optimism/op-service/sources"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
)

// engineBinary resolves the op-reth-test-engine binary once per test binary.
var engineBinary = sync.OnceValues(func() (string, error) {
	return testengine.ResolveBinary(context.Background())
})

// requireEngine returns the engine binary, failing (never skipping) the test when it is not
// available.
func requireEngine(t *testing.T) string {
	t.Helper()
	path, err := engineBinary()
	require.NoError(t, err, "resolve op-reth-test-engine")
	return path
}

// codeAddr is a genesis-seeded account carrying bytecode, so eth_getCode has something non-empty to
// return. seededCode is PUSH1 1, PUSH1 0, SSTORE, STOP — never executed, just present as code.
var (
	codeAddr   = common.HexToAddress("0x00000000000000000000000000000000c0de5eed")
	seededCode = []byte{0x60, 0x01, 0x60, 0x00, 0x55, 0x00}

	// messagePasserAddr is the L2ToL1MessagePasser predeploy — its storage root is the Isthmus
	// withdrawals root.
	messagePasserAddr = predeploys.L2ToL1MessagePasserAddr
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
	raw := startEngine(t, crypto.PubkeyToAddress(key.PublicKey))
	ctx := context.Background()

	// Drive two sequencer rounds directly over the socket to populate the chain. Every L2 block
	// leads with a deposit (op-node classifies L2 blocks by a leading deposit tx), then a user tx.
	genesisHash := testengine.GetBlock(t, raw, "earliest").Hash
	block1 := testengine.BuildBlock(t, raw, genesisHash, 2, []hexutil.Bytes{testengine.DepositTx(1)}, testengine.SignTx(t, key, 0))
	block2 := testengine.BuildBlock(t, raw, block1, 4, []hexutil.Bytes{testengine.DepositTx(2)}, testengine.SignTx(t, key, 1))

	// Now read the chain back through op-node's decoder, which reconstructs + verifies.
	logger := testlog.Logger(t, log.LevelDebug)
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
	require.Equal(t, uint8(optypes.DepositTxType), txs[0].Type(), "first tx is the L1-info/forced deposit")
	require.Equal(t, uint8(types.DynamicFeeTxType), txs[1].Type(), "second tx is the user 1559 tx")

	// eth_chainId is served for op-node's ChainID().
	cid, err := ethCl.ChainID(ctx)
	require.NoError(t, err)
	require.EqualValues(t, testengine.ChainID, bigs.Uint64Strict(cid))
}

// startEngine spawns the engine over a genesis that funds funded and seeds codeAddr and the message
// passer, and returns its client. The engine is closed when the test ends, and its stderr logged if
// the test failed.
func startEngine(t *testing.T, funded common.Address) *rpc.Client {
	t.Helper()
	// The Isthmus withdrawals-root is the L2ToL1MessagePasser predeploy's storage root; a real
	// deployed chain always has non-empty message-passer storage, so seed a slot here. Otherwise the
	// root equals the empty-trie hash and op-node's payload reconstruction misclassifies the block
	// as pre-Isthmus (dropping the withdrawals-root + requests-hash header fields).
	genesisPath := testengine.WriteGenesis(t, types.GenesisAlloc{
		messagePasserAddr: {Nonce: 1, Balance: big.NewInt(0), Storage: map[common.Hash]common.Hash{common.BigToHash(big.NewInt(0)): common.BigToHash(big.NewInt(1))}},
		codeAddr:          {Balance: big.NewInt(0), Code: seededCode},
		funded:            {Balance: new(big.Int).SetUint64(1_000_000_000_000_000_000)},
	})

	var logs bytes.Buffer
	proc, err := testengine.Spawn(requireEngine(t), genesisPath, &logs)
	require.NoError(t, err, "spawn engine; stderr:\n%s", logs.String())
	t.Cleanup(func() {
		proc.Close()
		if t.Failed() {
			t.Logf("engine stderr:\n%s", logs.String())
		}
	})
	return proc.Client()
}
