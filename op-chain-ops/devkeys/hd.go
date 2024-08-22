package devkeys

import (
	"crypto/ecdsa"
	"fmt"
	"math/big"

	hdwallet "github.com/ethereum-optimism/go-ethereum-hdwallet"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

const TestMnemonic = "test test test test test test test test test test test junk"

// MnemonicDevKeys derives dev keys from a mnemonic key-path structure as following:
// BIP-44: `m / purpose' / coin_type' / account' / change / address_index`
// purpose = standard secp256k1 usage (Eth2 BLS keys use different purpose data).
// coin_type = chain type, set to 60' for ETH. See SLIP-0044.
// account = for different identities, used here to separate domains.
// change = to separate external and internal addresses. Used here for chain ID, 0 if the domain is not a chain.
// address_index = used here to separate roles.
// The `'` char signifies BIP-32 hardened derivation.
//
// See:
// https://github.com/bitcoin/bips/blob/master/bip-0044.mediawiki
// https://github.com/satoshilabs/slips/blob/master/slip-0044.md
// https://github.com/bitcoin/bips/blob/master/bip-0032.mediawiki
type MnemonicDevKeys struct {
	w *hdwallet.Wallet
}

var _ DevKeys = (*MnemonicDevKeys)(nil)

func NewMnemonicDevKeys(mnemonic string) (*MnemonicDevKeys, error) {
	w, err := hdwallet.NewFromMnemonic(mnemonic)
	if err != nil {
		return nil, fmt.Errorf("invalid mnemonic: %w", err)
	}
	return &MnemonicDevKeys{w: w}, nil
}

func (d *MnemonicDevKeys) Secret(domain Domain, chainID *big.Int, role Role) (*ecdsa.PrivateKey, error) {
	account := accounts.Account{URL: accounts.URL{
		Path: fmt.Sprintf("m/44'/60'/%d'/%d/%d", uint64(domain), chainID, uint64(role)),
	}}
	priv, err := d.w.PrivateKey(account)
	if err != nil {
		return nil, fmt.Errorf("failed to derive key of path %s (domain: %s, chain: %d, role: %s): %w", account.URL.Path, domain, chainID, role, err)
	}
	return priv, nil
}

func (d *MnemonicDevKeys) Address(domain Domain, chainID *big.Int, role Role) (common.Address, error) {
	secret, err := d.Secret(domain, chainID, role)
	if err != nil {
		return common.Address{}, err
	}
	return crypto.PubkeyToAddress(secret.PublicKey), nil
}

// Scope is a helper method to not repeat domain and chainID on every address-getter call.
func Scope(addrs DevAddresses, domain Domain, chainID *big.Int) func(Role) (common.Address, error) {
	return func(role Role) (common.Address, error) {
		return addrs.Address(domain, chainID, role)
	}
}
