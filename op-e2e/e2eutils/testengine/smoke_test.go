package testengine_test

import (
	"bytes"
	"context"
	"errors"
	"math/big"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/testengine"
)

// engineBinary resolves the op-reth-test-engine binary once per test binary.
var engineBinary = sync.OnceValues(func() (string, error) {
	return testengine.ResolveBinary(context.Background())
})

// requireEngine returns the engine binary. A checkout without one configured or built skips the
// test with the resolver's instructions; any other resolution failure fails it, including
// REQUIRE_RUST_ENGINE without a binary path, which CI sets so the smoke gate can't silently pass.
func requireEngine(t *testing.T) string {
	t.Helper()
	path, err := engineBinary()
	if errors.Is(err, testengine.ErrNotConfigured) {
		t.Skip(err.Error())
	}
	require.NoError(t, err, "resolve op-reth-test-engine")
	return path
}

// TestEngineSmoke drives the full sequencer flow over the Unix socket — spawn → forkchoiceUpdated
// with attributes → optest_includeTx → getPayload → newPayload → forkchoiceUpdated — twice, and
// asserts the result is a valid chain of two blocks (parent-linked, distinct hashes, ascending
// numbers) read back via eth_getBlockByNumber. This is the end-to-end gate for the engine binary
// and its Go client.
func TestEngineSmoke(t *testing.T) {
	key, err := crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
	require.NoError(t, err)
	funded := crypto.PubkeyToAddress(key.PublicKey)

	genesisPath := testengine.WriteGenesis(t, types.GenesisAlloc{
		funded: {Balance: new(big.Int).SetUint64(1_000_000_000_000_000_000)},
	})

	var logs bytes.Buffer
	proc, err := testengine.Spawn(requireEngine(t), genesisPath, &logs)
	if err != nil {
		t.Fatalf("spawn engine: %v\n%s", err, logs.String())
	}
	defer func() {
		proc.Close()
		if t.Failed() {
			t.Logf("engine stderr:\n%s", logs.String())
		}
	}()
	cl := proc.Client()

	// The genesis block anchors the chain.
	genesis := testengine.GetBlock(t, cl, "earliest")
	require.EqualValues(t, 0, genesis.Number)

	// Block 1: one forced deposit plus a user tx.
	block1 := testengine.BuildBlock(t, cl, genesis.Hash, 2,
		[]hexutil.Bytes{testengine.DepositTx(1)}, testengine.SignTx(t, key, 0))
	// Block 2 builds on block 1 with a second user tx (no deposit).
	block2 := testengine.BuildBlock(t, cl, block1, 4, nil, testengine.SignTx(t, key, 1))

	// A valid chain of two blocks on top of genesis, read back over eth_.
	h1 := testengine.GetBlock(t, cl, "0x1")
	h2 := testengine.GetBlock(t, cl, "0x2")
	require.EqualValues(t, 1, h1.Number)
	require.Equal(t, genesis.Hash, h1.ParentHash, "block1 parent is genesis")
	require.Equal(t, block1, h1.Hash)
	require.EqualValues(t, 2, h2.Number)
	require.Equal(t, block1, h2.ParentHash, "block2 parent is block1")
	require.Equal(t, block2, h2.Hash)
	require.NotEqual(t, block1, block2, "distinct block hashes")

	// The safe/finalized pointers moved with the head (forkchoice was applied over the socket).
	require.Equal(t, block2, testengine.GetBlock(t, cl, "latest").Hash)
	require.Equal(t, block2, testengine.GetBlock(t, cl, "safe").Hash)
}

// TestSpawnEngineBadGenesis spawns the engine over a genesis file that does not exist. It must fail
// as soon as the engine exits, naming the file.
func TestSpawnEngineBadGenesis(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing-genesis.json")

	start := time.Now()
	_, err := testengine.Spawn(requireEngine(t), missing, nil)
	t.Logf("spawn error: %v", err)
	require.Less(t, time.Since(start), 10*time.Second, "spawn failure must not wait out the ready timeout")
	require.ErrorContains(t, err, "exit status")
	require.ErrorContains(t, err, "missing-genesis.json")
}
