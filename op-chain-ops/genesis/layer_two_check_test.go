package genesis

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-chain-ops/foundry"
	"github.com/ethereum-optimism/optimism/op-core/predeploys"
)

// minimalValidAllocs builds the smallest alloc set that passes CheckL2GenesisAllocs:
// one proxied predeploy with its implementation, plus a precompile.
func minimalValidAllocs() *foundry.ForgeAllocs {
	proxy := predeploys.L2CrossDomainMessengerAddr
	impl := codeNamespaceCounterpart(proxy)
	return &foundry.ForgeAllocs{Accounts: types.GenesisAlloc{
		common.BytesToAddress([]byte{0x01}): {
			Balance: big.NewInt(1),
		},
		proxy: {
			Balance: new(big.Int),
			Code:    []byte{0xfe},
			Storage: map[common.Hash]common.Hash{
				AdminSlot:          common.BytesToHash(predeploys.ProxyAdminAddr.Bytes()),
				ImplementationSlot: common.BytesToHash(impl.Bytes()),
			},
		},
		impl: {
			Balance: new(big.Int),
			Code:    []byte{0xfe},
		},
	}}
}

func TestCheckL2GenesisAllocs(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		require.NoError(t, CheckL2GenesisAllocs(minimalValidAllocs(), CheckL2AllocsOpts{}))
	})

	t.Run("stray account with code", func(t *testing.T) {
		allocs := minimalValidAllocs()
		stray := common.HexToAddress("0xdeaDDeADDEaDdeaDdEAddEADDEAdDeadDEADDEaD")
		allocs.Accounts[stray] = types.Account{Balance: new(big.Int), Code: []byte{0x60, 0x00}}
		err := CheckL2GenesisAllocs(allocs, CheckL2AllocsOpts{})
		require.ErrorContains(t, err, "stray account")
		require.ErrorContains(t, err, stray.Hex())
	})

	t.Run("bad admin slot", func(t *testing.T) {
		allocs := minimalValidAllocs()
		proxy := predeploys.L2CrossDomainMessengerAddr
		allocs.Accounts[proxy].Storage[AdminSlot] = common.BytesToHash(common.HexToAddress("0x1234").Bytes())
		err := CheckL2GenesisAllocs(allocs, CheckL2AllocsOpts{})
		require.ErrorContains(t, err, "admin slot")
	})

	t.Run("impl slot outside predeploy namespace", func(t *testing.T) {
		allocs := minimalValidAllocs()
		stray := common.HexToAddress("0xdeaDDeADDEaDdeaDdEAddEADDEAdDeadDEADDEaD")
		allocs.Accounts[stray] = types.Account{
			Balance: new(big.Int),
			Code:    []byte{0xfe},
			Storage: map[common.Hash]common.Hash{
				ImplementationSlot: common.BytesToHash(common.HexToAddress("0x1234").Bytes()),
			},
		}
		err := CheckL2GenesisAllocs(allocs, CheckL2AllocsOpts{})
		require.ErrorContains(t, err, "implementation slot")
	})

	t.Run("impl slot points at wrong address", func(t *testing.T) {
		allocs := minimalValidAllocs()
		proxy := predeploys.L2CrossDomainMessengerAddr
		allocs.Accounts[proxy].Storage[ImplementationSlot] = common.BytesToHash(common.HexToAddress("0x1234").Bytes())
		err := CheckL2GenesisAllocs(allocs, CheckL2AllocsOpts{})
		require.ErrorContains(t, err, "implementation slot points at")
	})

	t.Run("dev account requires opt-in", func(t *testing.T) {
		allocs := minimalValidAllocs()
		allocs.Accounts[DevAccounts[0]] = types.Account{Balance: big.NewInt(1e18)}

		err := CheckL2GenesisAllocs(allocs, CheckL2AllocsOpts{})
		require.ErrorContains(t, err, "stray account")

		require.NoError(t, CheckL2GenesisAllocs(allocs, CheckL2AllocsOpts{FundDevAccounts: true}))
	})

	t.Run("allowed EOA", func(t *testing.T) {
		allocs := minimalValidAllocs()
		eoa := common.HexToAddress("0x4242424242424242424242424242424242424242")
		allocs.Accounts[eoa] = types.Account{Balance: big.NewInt(1e18), Nonce: 1}

		err := CheckL2GenesisAllocs(allocs, CheckL2AllocsOpts{})
		require.ErrorContains(t, err, "stray account")

		require.NoError(t, CheckL2GenesisAllocs(allocs, CheckL2AllocsOpts{AllowedEOAs: []common.Address{eoa}}))
	})

	t.Run("precompile with unexpected state", func(t *testing.T) {
		allocs := minimalValidAllocs()
		allocs.Accounts[common.BytesToAddress([]byte{0x02})] = types.Account{Balance: big.NewInt(2)}
		err := CheckL2GenesisAllocs(allocs, CheckL2AllocsOpts{})
		require.ErrorContains(t, err, "precompile")
	})
}
