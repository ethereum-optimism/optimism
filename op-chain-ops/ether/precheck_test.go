package ether

import (
	"bytes"
	"math/big"
	"math/rand"
	"os"
	"sort"
	"testing"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-chain-ops/crossdomain"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/stretchr/testify/require"
)

func TestPreCheckBalances(t *testing.T) {
	log.Root().SetHandler(log.StreamHandler(os.Stderr, log.TerminalFormat(true)))

	tests := []struct {
		name            string
		totalSupply     *big.Int
		expDiff         *big.Int
		stateBalances   map[common.Address]*big.Int
		stateAllowances map[common.Address]common.Address
		inputAddresses  []common.Address
		inputAllowances []*crossdomain.Allowance
		check           func(t *testing.T, addrs FilteredOVMETHAddresses, err error)
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
			check: func(t *testing.T, addrs FilteredOVMETHAddresses, err error) {
				require.NoError(t, err)
				require.EqualValues(t, FilteredOVMETHAddresses{
					common.HexToAddress("0x123"),
					common.HexToAddress("0x456"),
				}, addrs)
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
			check: func(t *testing.T, addrs FilteredOVMETHAddresses, err error) {
				require.NoError(t, err)
				require.EqualValues(t, FilteredOVMETHAddresses{
					common.HexToAddress("0x123"),
				}, addrs)
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
			check: func(t *testing.T, addrs FilteredOVMETHAddresses, err error) {
				require.NoError(t, err)
				require.EqualValues(t, FilteredOVMETHAddresses{
					common.HexToAddress("0x123"),
				}, addrs)
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
			check: func(t *testing.T, addrs FilteredOVMETHAddresses, err error) {
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
			check: func(t *testing.T, addrs FilteredOVMETHAddresses, err error) {
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
			check: func(t *testing.T, addrs FilteredOVMETHAddresses, err error) {
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
			check: func(t *testing.T, addrs FilteredOVMETHAddresses, err error) {
				require.NoError(t, err)
				require.EqualValues(t, FilteredOVMETHAddresses{
					common.HexToAddress("0x123"),
					common.HexToAddress("0x456"),
				}, addrs)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := makeLegacyETH(t, tt.totalSupply, tt.stateBalances, tt.stateAllowances)
			factory := func() (*state.StateDB, error) {
				return db, nil
			}
			addrs, err := doMigration(factory, tt.inputAddresses, tt.inputAllowances, tt.expDiff, false)

			// Sort the addresses since they come in in a random order.
			sort.Slice(addrs, func(i, j int) bool {
				return bytes.Compare(addrs[i][:], addrs[j][:]) < 0
			})

			tt.check(t, addrs, err)
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

	for slot := range ignoredSlots {
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

// TestPreCheckBalancesRandom tests that the pre-check balances function works
// with random addresses. This test makes sure that the partition logic doesn't
// miss anything.
func TestPreCheckBalancesRandom(t *testing.T) {
	addresses := make([]common.Address, 0)
	stateBalances := make(map[common.Address]*big.Int)

	allowances := make([]*crossdomain.Allowance, 0)
	stateAllowances := make(map[common.Address]common.Address)

	totalSupply := big.NewInt(0)

	for i := 0; i < 100; i++ {
		for i := 0; i < rand.Intn(1000); i++ {
			addr := randAddr(t)
			addresses = append(addresses, addr)
			stateBalances[addr] = big.NewInt(int64(rand.Intn(1_000_000)))
			totalSupply = new(big.Int).Add(totalSupply, stateBalances[addr])
		}

		sort.Slice(addresses, func(i, j int) bool {
			return bytes.Compare(addresses[i][:], addresses[j][:]) < 0
		})

		for i := 0; i < rand.Intn(1000); i++ {
			addr := randAddr(t)
			to := randAddr(t)
			allowances = append(allowances, &crossdomain.Allowance{
				From: addr,
				To:   to,
			})
			stateAllowances[addr] = to
		}

		db := makeLegacyETH(t, totalSupply, stateBalances, stateAllowances)
		factory := func() (*state.StateDB, error) {
			return db, nil
		}

		outAddrs, err := doMigration(factory, addresses, allowances, big.NewInt(0), false)
		require.NoError(t, err)

		sort.Slice(outAddrs, func(i, j int) bool {
			return bytes.Compare(outAddrs[i][:], outAddrs[j][:]) < 0
		})
		require.EqualValues(t, addresses, outAddrs)
	}
}

func randAddr(t *testing.T) common.Address {
	var addr common.Address
	_, err := rand.Read(addr[:])
	require.NoError(t, err)
	return addr
}
