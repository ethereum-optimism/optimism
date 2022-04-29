package oracle

import (
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/gas-oracle/bindings"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

func TestBaseFeeUpdate(t *testing.T) {
	key, _ := crypto.GenerateKey()
	sim, _ := newSimulatedBackend(key)
	chain := sim.Blockchain()

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
		gasPrice:              big.NewInt(784637584),
	}

	update, err := wrapUpdateBaseFee(sim, sim, cfg)
	if err != nil {
		t.Fatal(err)
	}
	// Get the initial base fee
	l1BaseFee, err := gpo.L1BaseFee(&bind.CallOpts{})
	if err != nil {
		t.Fatal(err)
	}
	// base fee should start at 0
	if l1BaseFee.Cmp(common.Big0) != 0 {
		t.Fatal("does not start at 0")
	}
	// get the header to know what the base fee
	// should be updated to
	tip := chain.CurrentHeader()
	if tip.BaseFee == nil {
		t.Fatal("no base fee found")
	}
	// Ensure that there is no false negative by
	// checking that the values don't start out the same
	if l1BaseFee.Cmp(tip.BaseFee) == 0 {
		t.Fatal("values are already the same")
	}
	// Call the update function to do the update
	if err := update(); err != nil {
		t.Fatalf("cannot update base fee: %s", err)
	}
	sim.Commit()
	// Check the updated base fee
	l1BaseFee, err = gpo.L1BaseFee(&bind.CallOpts{})
	if err != nil {
		t.Fatal(err)
	}
	// the base fee should be equal to the value
	// on the header
	if tip.BaseFee.Cmp(l1BaseFee) != 0 {
		t.Fatal("base fee not updated")
	}
}
