package devkeys

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

func TestMnemonicDevKeys(t *testing.T) {
	m, err := NewMnemonicDevKeys(TestMnemonic)
	require.NoError(t, err)

	t.Run("default", func(t *testing.T) {
		defaultAccount, err := m.Address(DefaultKeyDomain, big.NewInt(0), 0)
		require.NoError(t, err)
		// Sanity check against a well-known dev account address,
		// to ensure the mnemonic path is formatted with the right hardening at each path segment.
		require.Equal(t, common.HexToAddress("0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266"), defaultAccount)
	})

	// Check that we have unique accounts for all possible dev combinations.
	// And domain/role toString will be exercised as part of sub-test names.
	addrs := make(map[common.Address]struct{})
	for domain := Domain(0); domain < 20; domain++ {
		t.Run("domain_"+domain.String(), func(t *testing.T) {
			for chID := 0; chID < 4; chID++ {
				chainID := big.NewInt(int64(chID))
				t.Run("chain_"+fmt.Sprintf("%d", chID), func(t *testing.T) {
					for role := Role(0); role < 20; role++ {
						t.Run("role_"+role.String(), func(t *testing.T) {
							secret, err := m.Secret(domain, chainID, role)
							require.NoError(t, err)
							addr, err := m.Address(domain, chainID, role)
							require.NoError(t, err)
							require.Equal(t, crypto.PubkeyToAddress(secret.PublicKey), addr)
							addrs[addr] = struct{}{}
						})
					}
				})
			}
		})
	}
	require.Len(t, addrs, 20*4*20, "unique address for each account")
}
