package bindings

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"errors"
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/l2geth/accounts/abi/bind"
	"github.com/ethereum-optimism/optimism/l2geth/accounts/abi/bind/backends"
	"github.com/ethereum-optimism/optimism/l2geth/common"
	"github.com/ethereum-optimism/optimism/l2geth/core"
	"github.com/ethereum-optimism/optimism/l2geth/core/rawdb"
	"github.com/ethereum-optimism/optimism/l2geth/core/types"
	"github.com/ethereum-optimism/optimism/l2geth/crypto"
	"github.com/ethereum-optimism/optimism/l2geth/eth/gasprice"
	"github.com/ethereum-optimism/optimism/l2geth/ethdb"
	"github.com/ethereum-optimism/optimism/l2geth/rollup/fees"
)

// Test that the fee calculation is the same in both go and solidity
func TestCalculateFee(t *testing.T) {
	key, _ := crypto.GenerateKey()
	sim, _ := newSimulatedBackend(key)
	chain := sim.Blockchain()

	opts, _ := NewKeyedTransactor(key)
	addr, _, gpo, err := DeployGasPriceOracle(opts, sim, opts.From)
	if err != nil {
		t.Fatal(err)
	}
	sim.Commit()
	callopts := bind.CallOpts{}

	signer := types.NewEIP155Signer(big.NewInt(1337))
	gasOracle := gasprice.NewRollupOracle()

	// Set the L1 base fee
	if _, err := gpo.SetL1BaseFee(opts, big.NewInt(1)); err != nil {
		t.Fatal("cannot set 1l base fee")
	}
	sim.Commit()

	tests := map[string]struct {
		tx *types.Transaction
	}{
		"simple": {
			types.NewTransaction(0, common.Address{}, big.NewInt(0), 0, big.NewInt(0), []byte{}),
		},
		"high-nonce": {
			types.NewTransaction(12345678, common.Address{}, big.NewInt(0), 0, big.NewInt(0), []byte{}),
		},
		"full-tx": {
			types.NewTransaction(20, common.HexToAddress("0x"), big.NewInt(1234), 215000, big.NewInt(769109341), common.FromHex(GasPriceOracleBin)),
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			tx := tt.tx
			raw := new(bytes.Buffer)
			if err := tx.EncodeRLP(raw); err != nil {
				t.Fatal("cannot rlp encode tx")
			}

			l1BaseFee, err := gpo.L1BaseFee(&callopts)
			if err != nil {
				t.Fatal("cannot get l1 base fee")
			}
			overhead, err := gpo.Overhead(&callopts)
			if err != nil {
				t.Fatal("cannot get overhead")
			}
			scalar, err := gpo.Scalar(&callopts)
			if err != nil {
				t.Fatal("cannot get scalar")
			}
			decimals, err := gpo.Decimals(&callopts)
			if err != nil {
				t.Fatal("cannot get decimals")
			}
			l2GasPrice, err := gpo.GasPrice(&callopts)
			if err != nil {
				t.Fatal("cannot get l2 gas price")
			}

			gasOracle.SetL1GasPrice(l1BaseFee)
			gasOracle.SetL2GasPrice(l2GasPrice)
			gasOracle.SetOverhead(overhead)
			gasOracle.SetScalar(scalar, decimals)

			l1Fee, err := gpo.GetL1Fee(&callopts, raw.Bytes())
			if err != nil {
				t.Fatal("cannot get l1 fee")
			}

			scaled := fees.ScaleDecimals(scalar, decimals)
			expectL1Fee := fees.CalculateL1Fee(raw.Bytes(), overhead, l1BaseFee, scaled)
			if expectL1Fee.Cmp(l1Fee) != 0 {
				t.Fatal("solidity does not match go")
			}

			state, err := chain.State()
			if err != nil {
				t.Fatal("cannot get state")
			}

			// Ignore the error here because the tx isn't signed
			msg, _ := tx.AsMessage(signer)

			l1MsgFee, err := fees.CalculateL1MsgFee(msg, state, &addr)
			if err != nil {
				t.Fatal(err)
			}
			if l1MsgFee.Cmp(expectL1Fee) != 0 {
				t.Fatal("l1 msg fee not computed correctly")
			}

			msgFee, err := fees.CalculateTotalMsgFee(msg, state, new(big.Int).SetUint64(msg.Gas()), &addr)
			if err != nil {
				t.Fatal("cannot calculate total msg fee")
			}
			txFee, err := fees.CalculateTotalFee(tx, gasOracle)
			if err != nil {
				t.Fatal("cannot calculate total tx fee")
			}
			if msgFee.Cmp(txFee) != 0 {
				t.Fatal("msg fee and tx fee mismatch")
			}
		})
	}
}

func newSimulatedBackend(key *ecdsa.PrivateKey) (*backends.SimulatedBackend, ethdb.Database) {
	var gasLimit uint64 = 9_000_000
	auth, _ := NewKeyedTransactor(key)
	genAlloc := make(core.GenesisAlloc)
	genAlloc[auth.From] = core.GenesisAccount{Balance: big.NewInt(9223372036854775807)}
	db := rawdb.NewMemoryDatabase()
	sim := backends.NewSimulatedBackendWithDatabase(db, genAlloc, gasLimit)
	return sim, db
}

// NewKeyedTransactor is a utility method to easily create a transaction signer
// from a single private key. This was copied and modified from upstream geth
func NewKeyedTransactor(key *ecdsa.PrivateKey) (*bind.TransactOpts, error) {
	keyAddr := crypto.PubkeyToAddress(key.PublicKey)
	return &bind.TransactOpts{
		From: keyAddr,
		Signer: func(signer types.Signer, address common.Address, tx *types.Transaction) (*types.Transaction, error) {
			if address != keyAddr {
				return nil, errors.New("unauthorized")
			}
			signature, err := crypto.Sign(signer.Hash(tx).Bytes(), key)
			if err != nil {
				return nil, err
			}
			return tx.WithSignature(signer, signature)
		},
		Context: context.Background(),
	}, nil
}
