package interop

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-chain-ops/interopgen"
	op_e2e "github.com/ethereum-optimism/optimism/op-e2e"
	"github.com/stretchr/testify/require"
)

// a test that demonstrates the use of the SuperSystem
// as a proto-development of an interop test
func TestDemonstrateSuperSystem(t *testing.T) {
	recipe := interopgen.InteropDevRecipe{
		L1ChainID:        900100,
		L2ChainIDs:       []uint64{900200, 900201},
		GenesisTimestamp: uint64(time.Now().Unix() + 3), // start chain 3 seconds from now
	}

	// create a super system from the recipe
	// and get the L2 IDs for use in the test
	s2 := NewSuperSystem(t, &recipe)
	ids := s2.L2IDs()

	// chainA is the first L2 chain
	chainA := ids[0]
	chainB := ids[1]

	client := s2.L2GethClient(chainA)

	// create two users on all L2 chains
	s2.AddUser("Alice")
	s2.AddUser("Bob")

	bobAddr := s2.Address(chainA, "Bob")

	// check the balance of Bob
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	bobBalance, err := client.BalanceAt(ctx, bobAddr, nil)
	require.NoError(t, err)
	expectedBalance, _ := big.NewInt(0).SetString("10000000000000000000000000", 10)
	require.Equal(t, expectedBalance, bobBalance)

	// send a tx from Alice to Bob
	s2.SendL2Tx(
		chainA,
		"Alice",
		func(l2Opts *op_e2e.TxOpts) {
			l2Opts.ToAddr = &bobAddr
			l2Opts.Value = big.NewInt(1000000)
			l2Opts.GasFeeCap = big.NewInt(1_000_000_000)
			l2Opts.GasTipCap = big.NewInt(1_000_000_000)
		},
	)

	// check the balance of Bob after the tx
	ctx, cancel = context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	bobBalance, err = client.BalanceAt(ctx, bobAddr, nil)
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
}
