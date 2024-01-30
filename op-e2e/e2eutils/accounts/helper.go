package accounts

import (
	"crypto/ecdsa"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

type Account struct {
	PrivKey      *ecdsa.PrivateKey
	Addr         common.Address
	TransactOpts *bind.TransactOpts
}

func NewAccount(chainID *big.Int) (*Account, error) {
	key, err := crypto.GenerateKey()
	if err != nil {
		return nil, fmt.Errorf("unable to generate ecdsa key: %w", err)
	}
	opts, err := bind.NewKeyedTransactorWithChainID(key, chainID)
	if err != nil {
		return nil, fmt.Errorf("failed to create transactor: %w", err)
	}
	return &Account{
		PrivKey:      key,
		Addr:         crypto.PubkeyToAddress(key.PublicKey),
		TransactOpts: opts,
	}, nil
}
