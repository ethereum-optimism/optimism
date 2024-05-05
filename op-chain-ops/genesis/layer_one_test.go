package genesis

import (
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
	"math/big"
	"testing"
)

// TestFundDevAccounts ensures that the developer accounts are
// added to the genesis state correctly.
func TestFundDevAccounts(t *testing.T) {
	gen := core.Genesis{
		Alloc: make(types.GenesisAlloc),
	}
	FundDevAccounts(&gen)
	require.Equal(t, len(gen.Alloc), len(DevAccounts))
	for _, account := range gen.Alloc {
		require.Equal(t, devBalance, account.Balance)
	}
}

// TestSetPrecompileBalances ensures that the precompiles are
// initialized with a balance of 1.
func TestSetPrecompileBalances(t *testing.T) {
	gen := core.Genesis{
		Alloc: make(types.GenesisAlloc),
	}
	SetPrecompileBalances(&gen)
	require.Equal(t, len(gen.Alloc), PrecompileCount)
	for _, account := range gen.Alloc {
		require.Equal(t, big.NewInt(1), account.Balance)
	}
}

// TestIsEmptyAccount ensures that an empty account can be detected.
func TestIsEmptyAccount(t *testing.T) {
	require.True(t, isAccountEmpty(types.Account{}))
	require.False(t, isAccountEmpty(types.Account{
		Nonce: 1,
	}))
}
