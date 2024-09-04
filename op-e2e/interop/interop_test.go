package interop

import (
	"context"
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-chain-ops/interopgen"
	op_e2e "github.com/ethereum-optimism/optimism/op-e2e"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/stretchr/testify/require"
)

// TestInterop stands up a basic L1
// and multiple L2 states
func TestInterop(t *testing.T) {
	ctx, _ := context.WithTimeout(context.Background(), 100*time.Second)
	recipe := interopgen.InteropDevRecipe{
		L1ChainID:        900100,
		L2ChainIDs:       []uint64{900200, 900201},
		GenesisTimestamp: uint64(time.Now().Unix() + 3), // start chain 3 seconds from now
	}

	// create a super system from the recipe
	// and get the L2 IDs for use in the test
	s2 := NewSuperSystem(t, &recipe)
	ids := s2.L2IDs()

	netA := ids[0]
	netABig, _ := new(big.Int).SetString(netA, 10)
	netB := ids[1]
	_ = netB

	client := s2.L2GethClient(t, netA)

	s2.AddUser("Alice")
	aliceASecret := s2.UserKey(netA, "Alice")
	aliceOpts, err := bind.NewKeyedTransactorWithChainID(
		&aliceASecret,
		netABig,
	)
	require.NoError(t, err)
	aliceAddr := aliceOpts.From
	aliceBalance, err := client.BalanceAt(ctx, aliceAddr, nil)
	fmt.Println("aliceBalance", aliceBalance)

	s2.AddUser("Bob")
	bobASecret := s2.UserKey(netA, "Bob")
	bobOpts, err := bind.NewKeyedTransactorWithChainID(
		&bobASecret,
		netABig,
	)
	require.NoError(t, err)
	bobAddr := bobOpts.From
	bobBalance, err := client.BalanceAt(ctx, bobAddr, nil)
	fmt.Println("bobBalance", bobBalance)

	// hack: sleep for 30 seconds to make up for indexing
	time.Sleep(30 * time.Second)

	// demonstration: sending a transaction from Alice to Bob
	// and checking the balance of Bob
	s2.SendL2Tx(
		t,
		netA,
		&aliceASecret,
		func(l2Opts *op_e2e.TxOpts) {
			l2Opts.ToAddr = &bobAddr
			l2Opts.Value = big.NewInt(1000000)
		},
	)

	fmt.Println("post-transaction")
	bobBalance, err = client.BalanceAt(ctx, bobAddr, nil)
	fmt.Println("bobBalance", bobBalance)

	// TODO (placeholder) Let the system test-run for a bit
	time.Sleep(time.Second * 30)
}
