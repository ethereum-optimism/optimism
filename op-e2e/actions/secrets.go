package actions

import (
	"crypto/ecdsa"
	"fmt"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	hdwallet "github.com/miguelmota/go-ethereum-hdwallet"
)

var DefaultMnemonicConfig = &MnemonicConfig{
	Mnemonic:     "test test test test test test test test test test test junk",
	Deployer:     "m/44'/60'/0'/0/1",
	// clique signer: removed, use engine API instead
	Proposer:     "m/44'/60'/0'/0/3",
	Batcher:      "m/44'/60'/0'/0/4",
	SequencerP2P: "m/44'/60'/0'/0/5",
	Alice:        "m/44'/60'/0'/0/6",
	Bob:          "m/44'/60'/0'/0/7",
	Mallory:      "m/44'/60'/0'/0/8",
}

// MnemonicConfig configures the private keys for the hive testnet.
// It's json-serializable, so we can ship it to e.g. the hardhat script client.
type MnemonicConfig struct {
	Mnemonic string `json:"mnemonic"`

	Deployer     string `json:"deployer"`

	// rollup actors
	Proposer     string `json:"proposer"`
	Batcher      string `json:"batcher"`
	SequencerP2P string `json:"sequencerP2P"`

	// prefunded L1/L2 accounts for testing
	Alice   string `json:"alice"`
	Bob     string `json:"bob"`
	Mallory string `json:"mallory"`
}

func (m *MnemonicConfig) Secrets() (*Secrets, error) {
	wallet, err := hdwallet.NewFromMnemonic(m.Mnemonic)
	if err != nil {
		return nil, fmt.Errorf("failed to create wallet: %w", err)
	}
	account := func(path string) accounts.Account {
		return accounts.Account{URL: accounts.URL{Path: path}}
	}

	deployer, err := wallet.PrivateKey(account(m.Deployer))
	if err != nil {
		return nil, err
	}
	proposer, err := wallet.PrivateKey(account(m.Proposer))
	if err != nil {
		return nil, err
	}
	batcher, err := wallet.PrivateKey(account(m.Batcher))
	if err != nil {
		return nil, err
	}
	sequencerP2P, err := wallet.PrivateKey(account(m.SequencerP2P))
	if err != nil {
		return nil, err
	}
	alice, err := wallet.PrivateKey(account(m.Alice))
	if err != nil {
		return nil, err
	}
	bob, err := wallet.PrivateKey(account(m.Bob))
	if err != nil {
		return nil, err
	}
	mallory, err := wallet.PrivateKey(account(m.Mallory))
	if err != nil {
		return nil, err
	}

	return &Secrets{
		Deployer:     deployer,
		Proposer:     proposer,
		Batcher:      batcher,
		SequencerP2P: sequencerP2P,
		Alice:        alice,
		Bob:          bob,
		Mallory:      mallory,
	}, nil
}

type Secrets struct {
	Deployer     *ecdsa.PrivateKey

	// rollup actors
	Proposer     *ecdsa.PrivateKey
	Batcher      *ecdsa.PrivateKey
	SequencerP2P *ecdsa.PrivateKey

	// prefunded L1/L2 accounts for testing
	Alice   *ecdsa.PrivateKey
	Bob     *ecdsa.PrivateKey
	Mallory *ecdsa.PrivateKey
}

func EncodePrivKey(priv *ecdsa.PrivateKey) hexutil.Bytes {
	privkey := make([]byte, 32)
	blob := priv.D.Bytes()
	copy(privkey[32-len(blob):], blob)
	return privkey
}

func (s *Secrets) Addresses() *Addresses {
	return &Addresses{
		Deployer:     crypto.PubkeyToAddress(s.Deployer.PublicKey),
		Proposer:     crypto.PubkeyToAddress(s.Proposer.PublicKey),
		Batcher:      crypto.PubkeyToAddress(s.Batcher.PublicKey),
		SequencerP2P: crypto.PubkeyToAddress(s.SequencerP2P.PublicKey),
		Alice:        crypto.PubkeyToAddress(s.Alice.PublicKey),
		Bob:          crypto.PubkeyToAddress(s.Bob.PublicKey),
		Mallory:      crypto.PubkeyToAddress(s.Mallory.PublicKey),
	}
}

type Addresses struct {
	Deployer     common.Address

	// rollup actors
	Proposer     common.Address
	Batcher      common.Address
	SequencerP2P common.Address

	// prefunded L1/L2 accounts for testing
	Alice   common.Address
	Bob     common.Address
	Mallory common.Address
}
