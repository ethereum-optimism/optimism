package ether

import (
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/op-chain-ops/crossdomain"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/stretchr/testify/require"
)

func TestMigrateLegacyETH(t *testing.T) {
	tests := []struct {
		name            string
		totalSupply     *big.Int
		expDiff         *big.Int
		stateBalances   map[common.Address]*big.Int
		stateAllowances map[common.Address]common.Address
		inputAddresses  []common.Address
		inputAllowances []*crossdomain.Allowance
		check           func(t *testing.T, db *state.StateDB, err error)
	}{
		{
			name:        "everything matches",
			totalSupply: big.NewInt(3),
			expDiff:     big.NewInt(0),
			stateBalances: map[common.Address]*big.Int{
				common.HexToAddress("0x123"): big.NewInt(1),
				common.HexToAddress("0x456"): big.NewInt(2),
			},
			stateAllowances: map[common.Address]common.Address{
				common.HexToAddress("0x123"): common.HexToAddress("0x456"),
			},
			inputAddresses: []common.Address{
				common.HexToAddress("0x123"),
				common.HexToAddress("0x456"),
			},
			inputAllowances: []*crossdomain.Allowance{
				{
					From: common.HexToAddress("0x123"),
					To:   common.HexToAddress("0x456"),
				},
			},
			check: func(t *testing.T, db *state.StateDB, err error) {
				require.NoError(t, err)
				require.Equal(t, db.GetBalance(common.HexToAddress("0x123")), big.NewInt(1))
				require.Equal(t, db.GetBalance(common.HexToAddress("0x456")), big.NewInt(2))
				require.Equal(t, db.GetState(OVMETHAddress, CalcOVMETHStorageKey(common.HexToAddress("0x123"))), common.Hash{})
				require.Equal(t, db.GetState(OVMETHAddress, CalcOVMETHStorageKey(common.HexToAddress("0x456"))), common.Hash{})
				require.Equal(t, db.GetState(OVMETHAddress, getOVMETHTotalSupplySlot()), common.Hash{})
			},
		},
		{
			name:        "extra input addresses",
			totalSupply: big.NewInt(1),
			expDiff:     big.NewInt(0),
			stateBalances: map[common.Address]*big.Int{
				common.HexToAddress("0x123"): big.NewInt(1),
			},
			inputAddresses: []common.Address{
				common.HexToAddress("0x123"),
				common.HexToAddress("0x456"),
			},
			check: func(t *testing.T, db *state.StateDB, err error) {
				require.NoError(t, err)
				require.Equal(t, db.GetBalance(common.HexToAddress("0x123")), big.NewInt(1))
				require.Equal(t, db.GetState(OVMETHAddress, CalcOVMETHStorageKey(common.HexToAddress("0x123"))), common.Hash{})
				require.Equal(t, db.GetState(OVMETHAddress, getOVMETHTotalSupplySlot()), common.Hash{})
			},
		},
		{
			name:        "extra input allowances",
			totalSupply: big.NewInt(1),
			expDiff:     big.NewInt(0),
			stateBalances: map[common.Address]*big.Int{
				common.HexToAddress("0x123"): big.NewInt(1),
			},
			stateAllowances: map[common.Address]common.Address{
				common.HexToAddress("0x123"): common.HexToAddress("0x456"),
			},
			inputAddresses: []common.Address{
				common.HexToAddress("0x123"),
				common.HexToAddress("0x456"),
			},
			inputAllowances: []*crossdomain.Allowance{
				{
					From: common.HexToAddress("0x123"),
					To:   common.HexToAddress("0x456"),
				},
				{
					From: common.HexToAddress("0x123"),
					To:   common.HexToAddress("0x789"),
				},
			},
			check: func(t *testing.T, db *state.StateDB, err error) {
				require.NoError(t, err)
				require.Equal(t, db.GetBalance(common.HexToAddress("0x123")), big.NewInt(1))
				require.Equal(t, db.GetState(OVMETHAddress, CalcOVMETHStorageKey(common.HexToAddress("0x123"))), common.Hash{})
				require.Equal(t, db.GetState(OVMETHAddress, getOVMETHTotalSupplySlot()), common.Hash{})
			},
		},
		{
			name:        "missing input addresses",
			totalSupply: big.NewInt(2),
			expDiff:     big.NewInt(0),
			stateBalances: map[common.Address]*big.Int{
				common.HexToAddress("0x123"): big.NewInt(1),
				common.HexToAddress("0x456"): big.NewInt(1),
			},
			inputAddresses: []common.Address{
				common.HexToAddress("0x123"),
			},
			check: func(t *testing.T, db *state.StateDB, err error) {
				require.Error(t, err)
				require.ErrorContains(t, err, "unknown storage slot")
			},
		},
		{
			name:        "missing input allowances",
			totalSupply: big.NewInt(2),
			expDiff:     big.NewInt(0),
			stateBalances: map[common.Address]*big.Int{
				common.HexToAddress("0x123"): big.NewInt(1),
			},
			stateAllowances: map[common.Address]common.Address{
				common.HexToAddress("0x123"): common.HexToAddress("0x456"),
				common.HexToAddress("0x123"): common.HexToAddress("0x789"),
			},
			inputAddresses: []common.Address{
				common.HexToAddress("0x123"),
			},
			inputAllowances: []*crossdomain.Allowance{
				{
					From: common.HexToAddress("0x123"),
					To:   common.HexToAddress("0x456"),
				},
			},
			check: func(t *testing.T, db *state.StateDB, err error) {
				require.Error(t, err)
				require.ErrorContains(t, err, "unknown storage slot")
			},
		},
		{
			name:        "bad supply diff",
			totalSupply: big.NewInt(4),
			expDiff:     big.NewInt(0),
			stateBalances: map[common.Address]*big.Int{
				common.HexToAddress("0x123"): big.NewInt(1),
				common.HexToAddress("0x456"): big.NewInt(2),
			},
			inputAddresses: []common.Address{
				common.HexToAddress("0x123"),
				common.HexToAddress("0x456"),
			},
			check: func(t *testing.T, db *state.StateDB, err error) {
				require.Error(t, err)
				require.ErrorContains(t, err, "supply mismatch")
			},
		},
		{
			name:        "good supply diff",
			totalSupply: big.NewInt(4),
			expDiff:     big.NewInt(1),
			stateBalances: map[common.Address]*big.Int{
				common.HexToAddress("0x123"): big.NewInt(1),
				common.HexToAddress("0x456"): big.NewInt(2),
			},
			inputAddresses: []common.Address{
				common.HexToAddress("0x123"),
				common.HexToAddress("0x456"),
			},
			check: func(t *testing.T, db *state.StateDB, err error) {
				require.NoError(t, err)
				require.Equal(t, db.GetBalance(common.HexToAddress("0x123")), big.NewInt(1))
				require.Equal(t, db.GetBalance(common.HexToAddress("0x456")), big.NewInt(2))
				require.Equal(t, db.GetState(OVMETHAddress, CalcOVMETHStorageKey(common.HexToAddress("0x123"))), common.Hash{})
				require.Equal(t, db.GetState(OVMETHAddress, CalcOVMETHStorageKey(common.HexToAddress("0x456"))), common.Hash{})
				require.Equal(t, db.GetState(OVMETHAddress, getOVMETHTotalSupplySlot()), common.Hash{})
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := makeLegacyETH(t, tt.totalSupply, tt.stateBalances, tt.stateAllowances)
			err := doMigration(db, tt.inputAddresses, tt.inputAllowances, tt.expDiff, false, true)
			tt.check(t, db, err)
		})
	}
}

func makeLegacyETH(t *testing.T, totalSupply *big.Int, balances map[common.Address]*big.Int, allowances map[common.Address]common.Address) *state.StateDB {
	db, err := state.New(common.Hash{}, state.NewDatabaseWithConfig(rawdb.NewMemoryDatabase(), &trie.Config{
		Preimages: true,
		Cache:     1024,
	}), nil)
	require.NoError(t, err)

	db.CreateAccount(OVMETHAddress)
	db.SetState(OVMETHAddress, getOVMETHTotalSupplySlot(), common.BigToHash(totalSupply))

	for slot := range OVMETHIgnoredSlots {
		if slot == getOVMETHTotalSupplySlot() {
			continue
		}
		db.SetState(OVMETHAddress, slot, common.Hash{31: 0xff})
	}
	for addr, balance := range balances {
		db.SetState(OVMETHAddress, CalcOVMETHStorageKey(addr), common.BigToHash(balance))
	}
	for from, to := range allowances {
		db.SetState(OVMETHAddress, CalcAllowanceStorageKey(from, to), common.BigToHash(big.NewInt(1)))
	}

	root, err := db.Commit(false)
	require.NoError(t, err)

	err = db.Database().TrieDB().Commit(root, true)
	require.NoError(t, err)

	return db
}
