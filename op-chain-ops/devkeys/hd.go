package devkeys

import (
	"crypto/ecdsa"
	"fmt"

	hdwallet "github.com/ethereum-optimism/go-ethereum-hdwallet"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

const TestMnemonic = "test test test test test test test test test test test junk"

type MnemonicDevKeys struct {
	w *hdwallet.Wallet
}

var _ Keys = (*MnemonicDevKeys)(nil)

func NewMnemonicDevKeys(mnemonic string) (*MnemonicDevKeys, error) {
	w, err := hdwallet.NewFromMnemonic(mnemonic)
	if err != nil {
		return nil, fmt.Errorf("invalid mnemonic: %w", err)
	}
	return &MnemonicDevKeys{w: w}, nil
}

func (d *MnemonicDevKeys) Secret(key Key) (*ecdsa.PrivateKey, error) {
	account := accounts.Account{URL: accounts.URL{
		Path: key.HDPath(),
	}}
	priv, err := d.w.PrivateKey(account)
	if err != nil {
		return nil, fmt.Errorf("failed to derive key of path %s (key description: %s): %w", account.URL.Path, key.String(), err)
	}
	return priv, nil
}

func (d *MnemonicDevKeys) Address(key Key) (common.Address, error) {
	secret, err := d.Secret(key)
	if err != nil {
		return common.Address{}, err
	}
	return crypto.PubkeyToAddress(secret.PublicKey), nil
}
