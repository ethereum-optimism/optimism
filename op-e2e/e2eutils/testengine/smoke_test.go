package testengine_test

import (
	"bytes"
	"context"
	"errors"
	"math/big"
	"sync"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/testengine"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/testengine/chainfixture"
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

// spawnEngine starts the engine over a chainfixture genesis holding alloc. The engine is closed at
// test cleanup, and its stderr is logged if the test failed.
func spawnEngine(t *testing.T, alloc types.GenesisAlloc) *testengine.Client {
	t.Helper()
	var logs bytes.Buffer
	engine, err := testengine.Spawn(requireEngine(t), chainfixture.WriteGenesis(t, alloc), &logs)
	if err != nil {
		t.Fatalf("spawn engine: %v\n%s", err, logs.String())
	}
	t.Cleanup(func() {
		engine.Close()
		if t.Failed() {
			t.Logf("engine stderr:\n%s", logs.String())
		}
	})
	return engine
}

// TestEngineSmoke builds one block over the Unix socket — forkchoiceUpdated with attributes (a
// forced deposit) → optest_includeTx (a user tx) → getPayload → newPayload → forkchoiceUpdated —
// and reads it back by number and as the latest and safe block. It is the end-to-end gate for the
// engine binary and its Go client.
func TestEngineSmoke(t *testing.T) {
	key, err := crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
	require.NoError(t, err)
	funded := crypto.PubkeyToAddress(key.PublicKey)

	engine := spawnEngine(t, types.GenesisAlloc{
		funded: {Balance: new(big.Int).SetUint64(1_000_000_000_000_000_000)},
	})

	genesis := chainfixture.BlockHash(t, engine, rpc.EarliestBlockNumber)
	block := chainfixture.BuildBlock(t, engine, genesis, 2,
		[]hexutil.Bytes{chainfixture.DepositTx(1)}, chainfixture.SignTx(t, key, 0))

	for _, number := range []rpc.BlockNumber{1, rpc.LatestBlockNumber, rpc.SafeBlockNumber} {
		require.Equal(t, block, chainfixture.BlockHash(t, engine, number), "block %s", number)
	}
}

// TestEngineDumpAccounts builds one block and dumps accounts at it over optest_dumpAccounts: a
// contract's genesis code and storage come back unchanged, the sender's nonce reflects the block,
// an absent account is left out, and an unknown block is an error.
func TestEngineDumpAccounts(t *testing.T) {
	key, err := crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
	require.NoError(t, err)
	funded := crypto.PubkeyToAddress(key.PublicKey)
	contract := common.HexToAddress("0xc0de")
	missing := common.HexToAddress("0xdead")
	contractAccount := types.Account{
		Code:    []byte{0x60, 0x00},
		Balance: big.NewInt(7),
		Nonce:   1,
		Storage: map[common.Hash]common.Hash{{0x01}: {0x02}},
	}

	engine := spawnEngine(t, types.GenesisAlloc{
		funded:   {Balance: new(big.Int).SetUint64(1_000_000_000_000_000_000)},
		contract: contractAccount,
	})

	genesis := chainfixture.BlockHash(t, engine, rpc.EarliestBlockNumber)
	block := chainfixture.BuildBlock(t, engine, genesis, 2,
		[]hexutil.Bytes{chainfixture.DepositTx(1)}, chainfixture.SignTx(t, key, 0))

	ctx := t.Context()
	dump, err := engine.DumpAccounts(ctx, block, []common.Address{funded, contract, missing})
	require.NoError(t, err)
	require.Equal(t, contractAccount, dump[contract])
	require.Equal(t, uint64(1), dump[funded].Nonce)
	require.NotContains(t, dump, missing)

	_, err = engine.DumpAccounts(ctx, common.Hash{0x01}, []common.Address{contract})
	require.Error(t, err)
}
