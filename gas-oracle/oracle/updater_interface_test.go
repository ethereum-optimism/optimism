package oracle

import (
	"context"
	"crypto/ecdsa"
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/gas-oracle/bindings"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/backends"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb"
)

func TestWrapGetLatestBlockNumberFn(t *testing.T) {
	key, _ := crypto.GenerateKey()
	sim, db := newSimulatedBackend(key)
	chain := sim.Blockchain()

	getLatest := wrapGetLatestBlockNumberFn(sim)

	// Generate a valid chain of 10 blocks
	blocks, _ := core.GenerateChain(chain.Config(), chain.CurrentBlock(), chain.Engine(), db, 10, nil)

	// Check that the latest is 0 to start
	latest, err := getLatest()
	if err != nil {
		t.Fatal(err)
	}
	if latest != 0 {
		t.Fatal("not zero")
	}

	// Insert the blocks one by one and assert that they are incrementing
	for i, block := range blocks {
		if _, err := chain.InsertChain([]*types.Block{block}); err != nil {
			t.Fatal(err)
		}
		latest, err := getLatest()
		if err != nil {
			t.Fatal(err)
		}
		// Handle zero index by adding 1
		if latest != uint64(i+1) {
			t.Fatal("mismatch")
		}
	}
}

func TestWrapUpdateL2GasPriceFn(t *testing.T) {
	key, _ := crypto.GenerateKey()
	sim, _ := newSimulatedBackend(key)

	opts, _ := bind.NewKeyedTransactorWithChainID(key, big.NewInt(1337))
	addr, _, gpo, err := bindings.DeployGasPriceOracle(opts, sim, opts.From)
	if err != nil {
		t.Fatal(err)
	}
	sim.Commit()

	cfg := &Config{
		privateKey:            key,
		l2ChainID:             big.NewInt(1337),
		gasPriceOracleAddress: addr,
		gasPrice:              big.NewInt(783460975),
	}

	updateL2GasPriceFn, err := wrapUpdateL2GasPriceFn(sim, cfg)
	if err != nil {
		t.Fatal(err)
	}

	for i := uint64(0); i < 10; i++ {
		err := updateL2GasPriceFn(i)
		if err != nil {
			t.Fatal(err)
		}
		sim.Commit()
		gasPrice, err := gpo.GasPrice(&bind.CallOpts{Context: context.Background()})
		if err != nil {
			t.Fatal(err)
		}
		if gasPrice.Uint64() != i {
			t.Fatal("mismatched gas price")
		}
	}
}

func TestWrapUpdateL2GasPriceFnNoUpdates(t *testing.T) {
	key, _ := crypto.GenerateKey()
	sim, _ := newSimulatedBackend(key)

	opts, _ := bind.NewKeyedTransactorWithChainID(key, big.NewInt(1337))
	// Deploy the contract
	addr, _, gpo, err := bindings.DeployGasPriceOracle(opts, sim, opts.From)
	if err != nil {
		t.Fatal(err)
	}
	sim.Commit()

	cfg := &Config{
		privateKey:            key,
		l2ChainID:             big.NewInt(1337),
		gasPriceOracleAddress: addr,
		gasPrice:              big.NewInt(772763153),
		// the new gas price must change be 50% for it to actually update
		l2GasPriceSignificanceFactor: 0.5,
	}
	updateL2GasPriceFn, err := wrapUpdateL2GasPriceFn(sim, cfg)
	if err != nil {
		t.Fatal(err)
	}

	// Create a function to do the assertions
	tryUpdate := func(price uint64, shouldUpdate bool) {
		// Get a reference to the original gas price
		original, err := gpo.GasPrice(&bind.CallOpts{Context: context.Background()})
		if err != nil {
			t.Fatal(err)
		}

		// Call the updateL2GasPriceFn and commit the state
		if err := updateL2GasPriceFn(price); err != nil {
			t.Fatal(err)
		}
		sim.Commit()

		// Get a reference to the potentially updated state
		updated, err := gpo.GasPrice(&bind.CallOpts{Context: context.Background()})
		if err != nil {
			t.Fatal(err)
		}

		// the assertion differs depending on if it is expected that the
		// update occurs or not
		switch shouldUpdate {
		case true:
			// the price passed in should equal updated
			if updated.Uint64() != price {
				t.Fatalf("mismatched gas price, expect %d - got %d - should update (%t)", updated, price, shouldUpdate)
			}
		case false:
			// the original should match the updated
			if original.Uint64() != updated.Uint64() {
				t.Fatalf("mismatched gas price, expect %d - got %d - should update (%t)", original, updated, shouldUpdate)
			}
		}
	}

	// tryUpdate(newGasPrice, shouldUpdate)
	// The gas price starts out at 0
	// try to update it to 0 and it should not update
	tryUpdate(0, false)
	// update it to 2 and it should update
	tryUpdate(2, true)
	// it should not update to 3
	tryUpdate(3, false)
	// it should update to 4
	tryUpdate(4, true)
	// it should not update back down to 3
	tryUpdate(3, false)
	// it should update to 1
	tryUpdate(1, true)
}

func TestIsDifferenceSignificant(t *testing.T) {
	tests := []struct {
		name   string
		a      uint64
		b      uint64
		sig    float64
		expect bool
	}{
		{name: "test 1", a: 1, b: 1, sig: 0.05, expect: false},
		{name: "test 2", a: 4, b: 1, sig: 0.25, expect: true},
		{name: "test 3", a: 3, b: 1, sig: 0.1, expect: true},
		{name: "test 4", a: 4, b: 1, sig: 0.9, expect: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := isDifferenceSignificant(tc.a, tc.b, tc.sig)
			if result != tc.expect {
				t.Fatalf("mismatch %s", tc.name)
			}
		})
	}
}

func newSimulatedBackend(key *ecdsa.PrivateKey) (*backends.SimulatedBackend, ethdb.Database) {
	var gasLimit uint64 = 9_000_000
	auth, _ := bind.NewKeyedTransactorWithChainID(key, big.NewInt(1337))
	genAlloc := make(core.GenesisAlloc)
	genAlloc[auth.From] = core.GenesisAccount{Balance: big.NewInt(9223372036854775807)}
	db := rawdb.NewMemoryDatabase()
	sim := backends.NewSimulatedBackendWithDatabase(db, genAlloc, gasLimit)
	return sim, db
}
