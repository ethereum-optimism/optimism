package interop

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-chain-ops/interopgen"
	"github.com/ethereum-optimism/optimism/op-e2e/system/helpers"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

// TestInteropTrivial tests a simple interop scenario
// Chains A and B exist, but no messages are sent between them
// and in fact no event-logs are emitted by either chain at all.
// A transaction is sent from Alice to Bob on Chain A.
// The balance of Bob on Chain A is checked before and after the tx.
// The balance of Bob on Chain B is checked after the tx.
func TestInteropTrivial(t *testing.T) {
	genesisTimestamp := uint64(time.Now().Unix() + 3) // start chain 3 seconds from now
	recipe := interopgen.InteropDevRecipe{
		L1ChainID:        900100,
		L2ChainIDs:       []uint64{900200, 900201},
		GenesisTimestamp: genesisTimestamp, // start chain 3 seconds from now
	}
	worldResources := worldResourcePaths{
		foundryArtifacts: "../../packages/contracts-bedrock/forge-artifacts",
		sourceMap:        "../../packages/contracts-bedrock",
	}

	// create a super system from the recipe
	// and get the L2 IDs for use in the test
	s2 := NewSuperSystem(t, &recipe, worldResources)
	ids := s2.L2IDs()

	// chainA is the first L2 chain
	chainA := ids[0]
	// chainB is the second L2 chain
	chainB := ids[1]

	// create two users on all L2 chains
	s2.AddUser("Alice")
	s2.AddUser("Bob")

	bobAddr := s2.Address(chainA, "Bob")

	// check the balance of Bob
	clientA := s2.L2GethClient(chainA)
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	bobBalance, err := clientA.BalanceAt(ctx, bobAddr, nil)
	require.NoError(t, err)
	expectedBalance, _ := big.NewInt(0).SetString("10000000000000000000000000", 10)
	require.Equal(t, expectedBalance, bobBalance)

	// send a tx from Alice to Bob
	s2.SendL2Tx(
		chainA,
		"Alice",
		func(l2Opts *helpers.TxOpts) {
			l2Opts.ToAddr = &bobAddr
			l2Opts.Value = big.NewInt(1000000)
			l2Opts.GasFeeCap = big.NewInt(1_000_000_000)
			l2Opts.GasTipCap = big.NewInt(1_000_000_000)
		},
	)

	// check the balance of Bob after the tx
	ctx, cancel = context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	bobBalance, err = clientA.BalanceAt(ctx, bobAddr, nil)
	require.NoError(t, err)
	expectedBalance, _ = big.NewInt(0).SetString("10000000000000000001000000", 10)
	require.Equal(t, expectedBalance, bobBalance)

	// check that the balance of Bob on ChainB hasn't changed
	bobAddrB := s2.Address(chainB, "Bob")
	clientB := s2.L2GethClient(chainB)
	ctx, cancel = context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	bobBalance, err = clientB.BalanceAt(ctx, bobAddrB, nil)
	require.NoError(t, err)
	expectedBalance, _ = big.NewInt(0).SetString("10000000000000000000000000", 10)
	require.Equal(t, expectedBalance, bobBalance)

	// put some generic log event data down
	s2.DeployEmitterContract(chainA, "Alice")

	rec := s2.EmitData(chainA, "Alice", "emit this!")

	client := s2.L2GethClient(chainA)
	ctx, cancel = context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	block, err := client.BlockByHash(ctx, rec.BlockHash)
	require.NoError(t, err)

	logIndex, ok, err := getStartingLogIndex(client, rec.BlockHash, rec.TxHash)
	require.NoError(t, err, "Getting starting log index")
	require.True(t, ok, "Transaction not found in block")

	// build an identifier to that log event using the receipt
	identifier := types.Identifier{
		Origin:      s2.Address(chainA, "Alice"),
		BlockNumber: rec.BlockNumber.Uint64(),
		LogIndex:    logIndex,
		Timestamp:   block.Time(),
		ChainID:     types.ChainIDFromBig(s2.ChainID(chainA)),
	}

	// indicate the executing message dependency on the log event
	s2.ExecuteMessage(chainA, "Alice", identifier, "Alice", []byte("emit this!"))

	time.Sleep(60 * time.Second)

}

// getStartingLogIndex returns the block-level index of the first log emitted by the given transaction, and a bool
// indicating whether the transaction was found in the block.
// It does not validate that the given transaction actually has logs.
func getStartingLogIndex(client *ethclient.Client, blockHash common.Hash, txHash common.Hash) (uint64, bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	receipts, err := client.BlockReceipts(ctx, rpc.BlockNumberOrHash{BlockHash: &blockHash})
	if err != nil {
		return 0, false, err
	}

	// Count the logs until we find the tx
	logCount := uint64(0)
	txFound := false
	for _, receipt := range receipts {
		if receipt.TxHash == txHash {
			txFound = true
			break
		}
		logCount += uint64(len(receipt.Logs))
	}
	return logCount, txFound, nil
}
