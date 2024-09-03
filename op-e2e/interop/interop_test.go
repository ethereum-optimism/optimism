package interop

import (
	"context"
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	"github.com/ethereum-optimism/optimism/op-chain-ops/interopgen"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/services"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/stretchr/testify/require"
)

// TestInterop stands up a basic L1
// and multiple L2 states
func TestInterop(t *testing.T) {
	recipe := interopgen.InteropDevRecipe{
		L1ChainID:        900100,
		L2ChainIDs:       []uint64{900200, 900201},
		GenesisTimestamp: uint64(time.Now().Unix() + 3), // start chain 3 seconds from now
	}

	s2 := system2{recipe: &recipe}
	s2.prepare(t)

	ids := s2.getL2IDs()

	netA := ids[0]
	// netB := ids[1]

	// demonstration: getting a batcher for network A
	_ = s2.getBatcher(netA)
	// or by direct map access
	_ = s2.l2s[netA].batcher
	//batcherA.Start(context.Background())

	// demonstration: getting a batcher for network A
	// and getting a transactor for the batcher
	// TODO: this can be abstracted
	batcherASecret := s2.l2s[netA].secrets[devkeys.BatcherRole]
	netABig, ok := new(big.Int).SetString(netA, 10)
	require.True(t, ok)
	opts, err := bind.NewKeyedTransactorWithChainID(
		&batcherASecret,
		netABig,
	)
	require.NoError(t, err)
	fromAddr := opts.From

	// dmonstration: checking a balance
	ctx, _ := context.WithTimeout(context.Background(), 1*time.Second)
	var ethClient services.EthInstance = s2.l2s[netA].l2Geth
	client := s2.NodeClient(t, netA, ethClient.UserRPC())
	startBalance, err := client.BalanceAt(ctx, fromAddr, nil)
	fmt.Println("startBalance", startBalance)

	// TODO (placeholder) Let the system test-run for a bit
	time.Sleep(time.Second * 30)
}
