package ether

import (
	"math/big"
	"math/rand"
	"testing"

	"github.com/bobanetwork/boba/boba-bindings/predeploys"
	"github.com/bobanetwork/boba/boba-chain-ops/crossdomain"
	"github.com/ledgerwatch/erigon-lib/chain"
	"github.com/ledgerwatch/erigon-lib/common"
	"github.com/ledgerwatch/erigon/core/types"
	"github.com/stretchr/testify/require"
)

func TestSetBalance(t *testing.T) {
	g := &types.Genesis{
		Alloc: types.GenesisAlloc{
			predeploys.LegacyERC20ETHAddr: types.GenesisAccount{
				Storage: map[common.Hash]common.Hash{
					CalcOVMETHStorageKey(common.Address{1}):                       {1},
					CalcOVMETHStorageKey(common.Address{2}):                       {2},
					CalcAllowanceStorageKey(common.Address{1}, common.Address{2}): {3},
				},
			},
		},
	}
	SetBalance(g, common.Address{1}, big.NewInt(1), CalcOVMETHStorageKey(common.Address{1}))
	accountState := types.GenesisAccount{
		Balance: big.NewInt(1),
	}
	storageState := map[common.Hash]common.Hash{
		CalcOVMETHStorageKey(common.Address{2}):                       {2},
		CalcAllowanceStorageKey(common.Address{1}, common.Address{2}): {3},
	}
	require.Equal(t, accountState, g.Alloc[common.Address{1}])
	require.Equal(t, storageState, g.Alloc[predeploys.LegacyERC20ETHAddr].Storage)
	SetBalance(g, common.Address{1}, big.NewInt(1), CalcAllowanceStorageKey(common.Address{1}, common.Address{2}))
	accountState = types.GenesisAccount{
		Balance: big.NewInt(1),
	}
	storageState = map[common.Hash]common.Hash{
		CalcOVMETHStorageKey(common.Address{2}): {2},
	}
	require.Equal(t, accountState, g.Alloc[common.Address{1}])
	require.Equal(t, storageState, g.Alloc[predeploys.LegacyERC20ETHAddr].Storage)
}

func TestSetTotalSupply(t *testing.T) {
	g := &types.Genesis{
		Alloc: types.GenesisAlloc{
			predeploys.LegacyERC20ETHAddr: types.GenesisAccount{
				Storage: map[common.Hash]common.Hash{
					CalcOVMETHTotalSupplyKey():                                    {100},
					CalcOVMETHStorageKey(common.Address{1}):                       {1},
					CalcOVMETHStorageKey(common.Address{2}):                       {2},
					CalcAllowanceStorageKey(common.Address{1}, common.Address{2}): {3},
				},
			},
		},
	}

	SetTotalSupply(g)
	expectedStorage := map[common.Hash]common.Hash{
		CalcOVMETHTotalSupplyKey():                                    {},
		CalcOVMETHStorageKey(common.Address{1}):                       {1},
		CalcOVMETHStorageKey(common.Address{2}):                       {2},
		CalcAllowanceStorageKey(common.Address{1}, common.Address{2}): {3},
	}
	require.Equal(t, expectedStorage, g.Alloc[predeploys.LegacyERC20ETHAddr].Storage)
}

func TestMigrateBalances(t *testing.T) {
	tests := []struct {
		name        string
		totalSupply *big.Int
		expDiff     *big.Int
		addresses   []common.Address
		allowances  []*crossdomain.Allowance
		genesis     *types.Genesis
		noCheck     bool
		check       func(t *testing.T, g *types.Genesis, err error)
	}{
		{
			name:        "everything matches",
			totalSupply: big.NewInt(3),
			expDiff:     big.NewInt(0),
			addresses: []common.Address{
				{101},
				{102},
			},
			allowances: []*crossdomain.Allowance{
				{
					From: common.Address{102},
					To:   common.Address{101},
				},
			},
			genesis: &types.Genesis{
				Config: &chain.Config{
					ChainID: big.NewInt(28),
				},
				Alloc: types.GenesisAlloc{
					predeploys.LegacyERC20ETHAddr: types.GenesisAccount{
						Storage: map[common.Hash]common.Hash{
							CalcOVMETHTotalSupplyKey():                                        common.BigToHash(common.Big3),
							CalcOVMETHStorageKey(common.Address{101}):                         common.BigToHash(common.Big1),
							CalcOVMETHStorageKey(common.Address{102}):                         common.BigToHash(common.Big2),
							CalcAllowanceStorageKey(common.Address{102}, common.Address{101}): common.BigToHash(common.Big1),
						},
					},
				},
			},
			noCheck: true,
			check: func(t *testing.T, g *types.Genesis, err error) {
				require.NoError(t, err)
				require.EqualValues(t, common.Big1, g.Alloc[common.Address{101}].Balance)
				require.EqualValues(t, common.Big2, g.Alloc[common.Address{102}].Balance)
				require.EqualValues(t, common.Hash{}, g.Alloc[predeploys.LegacyERC20ETHAddr].Storage[CalcOVMETHTotalSupplyKey()])
				require.EqualValues(t, common.Hash{}, g.Alloc[predeploys.LegacyERC20ETHAddr].Storage[CalcOVMETHStorageKey(common.Address{101})])
				require.EqualValues(t, common.Hash{}, g.Alloc[predeploys.LegacyERC20ETHAddr].Storage[CalcOVMETHStorageKey(common.Address{102})])
			},
		},
		{
			name:        "extra input addresses",
			totalSupply: big.NewInt(1),
			expDiff:     big.NewInt(0),
			addresses: []common.Address{
				{1},
				{2},
			},
			allowances: []*crossdomain.Allowance{},
			genesis: &types.Genesis{
				Config: &chain.Config{
					ChainID: big.NewInt(28),
				},
				Alloc: types.GenesisAlloc{
					predeploys.LegacyERC20ETHAddr: types.GenesisAccount{
						Storage: map[common.Hash]common.Hash{
							CalcOVMETHTotalSupplyKey():              common.BigToHash(common.Big1),
							CalcOVMETHStorageKey(common.Address{1}): common.BigToHash(common.Big1),
						},
					},
				},
			},
			noCheck: false,
			check: func(t *testing.T, g *types.Genesis, err error) {
				require.NoError(t, err)
				require.EqualValues(t, common.Big1, g.Alloc[common.Address{1}].Balance)
				require.EqualValues(t, common.Hash{}, g.Alloc[predeploys.LegacyERC20ETHAddr].Storage[CalcOVMETHTotalSupplyKey()])
				require.EqualValues(t, common.Hash{}, g.Alloc[predeploys.LegacyERC20ETHAddr].Storage[CalcOVMETHStorageKey(common.Address{1})])
			},
		},
		{
			name:        "extra input allowances",
			totalSupply: big.NewInt(1),
			expDiff:     big.NewInt(0),
			addresses: []common.Address{
				{1},
				{2},
			},
			allowances: []*crossdomain.Allowance{
				{
					From: common.Address{1},
					To:   common.Address{2},
				},
				{
					From: common.Address{1},
					To:   common.Address{3},
				},
			},
			genesis: &types.Genesis{
				Config: &chain.Config{
					ChainID: big.NewInt(28),
				},
				Alloc: types.GenesisAlloc{
					predeploys.LegacyERC20ETHAddr: types.GenesisAccount{
						Storage: map[common.Hash]common.Hash{
							CalcOVMETHTotalSupplyKey():              common.BigToHash(common.Big1),
							CalcOVMETHStorageKey(common.Address{1}): common.BigToHash(common.Big1),
						},
					},
				},
			},
			noCheck: false,
			check: func(t *testing.T, g *types.Genesis, err error) {
				require.NoError(t, err)
				require.EqualValues(t, common.Big1, g.Alloc[common.Address{1}].Balance)
				require.EqualValues(t, common.Hash{}, g.Alloc[predeploys.LegacyERC20ETHAddr].Storage[CalcOVMETHTotalSupplyKey()])
				require.EqualValues(t, common.Hash{}, g.Alloc[predeploys.LegacyERC20ETHAddr].Storage[CalcOVMETHStorageKey(common.Address{1})])
			},
		},
		{
			name:        "missing input addresses",
			totalSupply: big.NewInt(2),
			expDiff:     big.NewInt(0),
			addresses: []common.Address{
				{1},
			},
			genesis: &types.Genesis{
				Config: &chain.Config{
					ChainID: big.NewInt(28),
				},
				Alloc: types.GenesisAlloc{
					predeploys.LegacyERC20ETHAddr: types.GenesisAccount{
						Storage: map[common.Hash]common.Hash{
							CalcOVMETHTotalSupplyKey():              common.BigToHash(common.Big2),
							CalcOVMETHStorageKey(common.Address{1}): common.BigToHash(common.Big1),
							CalcOVMETHStorageKey(common.Address{2}): common.BigToHash(common.Big1),
						},
					},
				},
			},
			noCheck: false,
			check: func(t *testing.T, g *types.Genesis, err error) {
				require.ErrorContains(t, err, "unknown storage slot")
			},
		},
		{
			name:        "missing input allowances",
			totalSupply: big.NewInt(2),
			expDiff:     big.NewInt(0),
			addresses: []common.Address{
				{1},
			},
			allowances: []*crossdomain.Allowance{
				{
					From: common.Address{1},
					To:   common.Address{2},
				},
			},
			genesis: &types.Genesis{
				Config: &chain.Config{
					ChainID: big.NewInt(28),
				},
				Alloc: types.GenesisAlloc{
					predeploys.LegacyERC20ETHAddr: types.GenesisAccount{
						Storage: map[common.Hash]common.Hash{
							CalcOVMETHTotalSupplyKey():                                    common.BigToHash(common.Big2),
							CalcOVMETHStorageKey(common.Address{3}):                       common.BigToHash(common.Big1),
							CalcAllowanceStorageKey(common.Address{1}, common.Address{2}): common.BigToHash(common.Big1),
							CalcAllowanceStorageKey(common.Address{1}, common.Address{3}): common.BigToHash(common.Big1),
						},
					},
				},
			},
			noCheck: false,
			check: func(t *testing.T, g *types.Genesis, err error) {
				require.ErrorContains(t, err, "unknown storage slot")
			},
		},
		{
			name:        "bad supply diff",
			totalSupply: big.NewInt(4),
			expDiff:     big.NewInt(0),
			addresses: []common.Address{
				{1},
				{2},
			},
			genesis: &types.Genesis{
				Config: &chain.Config{
					ChainID: big.NewInt(28),
				},
				Alloc: types.GenesisAlloc{
					predeploys.LegacyERC20ETHAddr: types.GenesisAccount{
						Storage: map[common.Hash]common.Hash{
							CalcOVMETHTotalSupplyKey():              common.BigToHash(big.NewInt(4)),
							CalcOVMETHStorageKey(common.Address{1}): common.BigToHash(common.Big1),
							CalcOVMETHStorageKey(common.Address{2}): common.BigToHash(common.Big2),
						},
					},
				},
			},
			noCheck: false,
			check: func(t *testing.T, g *types.Genesis, err error) {
				require.ErrorContains(t, err, "supply mismatch")
			},
		},
		{
			name:        "good supply diff",
			totalSupply: big.NewInt(4),
			expDiff:     big.NewInt(1),
			addresses: []common.Address{
				{1},
				{2},
			},
			genesis: &types.Genesis{
				Config: &chain.Config{
					ChainID: big.NewInt(28),
				},
				Alloc: types.GenesisAlloc{
					predeploys.LegacyERC20ETHAddr: types.GenesisAccount{
						Storage: map[common.Hash]common.Hash{
							CalcOVMETHTotalSupplyKey():              common.BigToHash(big.NewInt(4)),
							CalcOVMETHStorageKey(common.Address{1}): common.BigToHash(common.Big1),
							CalcOVMETHStorageKey(common.Address{2}): common.BigToHash(common.Big2),
						},
					},
				},
			},
			noCheck: false,
			check: func(t *testing.T, g *types.Genesis, err error) {
				require.NoError(t, err)
				require.EqualValues(t, common.Big1, g.Alloc[common.Address{1}].Balance)
				require.EqualValues(t, common.Big2, g.Alloc[common.Address{2}].Balance)
				require.EqualValues(t, common.Hash{}, g.Alloc[predeploys.LegacyERC20ETHAddr].Storage[CalcOVMETHTotalSupplyKey()])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := tt.genesis
			addr := tt.addresses
			allow := tt.allowances
			diff := tt.expDiff
			noCheck := tt.noCheck
			err := doMigration(g, addr, allow, diff, noCheck)
			tt.check(t, g, err)
		})
	}
}

// TestMigrateBalancesRandomOK tests that the pre-check balances function works
// with random addresses. This test makes sure that the partition logic doesn't
// miss anything, and helps detect concurrency errors.
func TestMigrateBalancesRandomOK(t *testing.T) {
	for i := 0; i < 100; i++ {
		g, addresses, allowances, balances := setupRandTest(t)

		err := doMigration(g, addresses, allowances, big.NewInt(0), false)
		require.NoError(t, err)

		for _, addr := range addresses {
			actBal := balances[addr]
			expBal := GetBalance(g, addr)
			require.EqualValues(t, expBal, actBal)
		}
	}
}

// TestMigrateBalancesRandomMissing tests that the pre-check balances function works
// with random addresses when some of them are missing. This helps make sure that the
// partition logic doesn't miss anything, and helps detect concurrency errors.
func TestMigrateBalancesRandomMissing(t *testing.T) {
	for i := 0; i < 100; i++ {
		g, addresses, allowances, _ := setupRandTest(t)

		if len(addresses) == 0 {
			continue
		}

		// Remove a random address from the list of witnesses
		idx := rand.Intn(len(addresses))
		addresses = append(addresses[:idx], addresses[idx+1:]...)

		err := doMigration(g, addresses, allowances, big.NewInt(0), false)
		require.ErrorContains(t, err, "unknown storage slot")
	}
}

func randAddr(t *testing.T) common.Address {
	var addr common.Address
	_, err := rand.Read(addr[:])
	require.NoError(t, err)
	return addr
}

func setupRandTest(t *testing.T) (*types.Genesis, []common.Address, []*crossdomain.Allowance, map[common.Address]*big.Int) {
	genesis := &types.Genesis{
		Config: &chain.Config{
			ChainID: big.NewInt(28),
		},
		Alloc: types.GenesisAlloc{
			predeploys.LegacyERC20ETHAddr: types.GenesisAccount{
				Storage: map[common.Hash]common.Hash{},
			},
		},
	}
	addresses := make([]common.Address, 0)
	allowances := make([]*crossdomain.Allowance, 0)
	balances := make(map[common.Address]*big.Int)

	totalSupply := big.NewInt(0)

	for j := 0; j < rand.Intn(10000); j++ {
		addr := randAddr(t)
		addresses = append(addresses, addr)
		balance := common.BigToHash(big.NewInt(int64(rand.Intn(1_000_000))))
		genesis.Alloc[predeploys.LegacyERC20ETHAddr].Storage[CalcOVMETHStorageKey(addr)] = balance
		totalSupply = new(big.Int).Add(totalSupply, balance.Big())
		balances[addr] = balance.Big()
	}

	for j := 0; j < rand.Intn(1000); j++ {
		addr := randAddr(t)
		to := randAddr(t)
		allowances = append(allowances, &crossdomain.Allowance{
			From: addr,
			To:   to,
		})
		genesis.Alloc[predeploys.LegacyERC20ETHAddr].Storage[CalcAllowanceStorageKey(addr, to)] = common.BigToHash(big.NewInt(int64(rand.Intn(1_000_000))))
	}

	genesis.Alloc[predeploys.LegacyERC20ETHAddr].Storage[CalcOVMETHTotalSupplyKey()] = common.BigToHash(totalSupply)

	return genesis, addresses, allowances, balances
}
