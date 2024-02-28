package hdwallet

import (
	"crypto/ecdsa"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"os"
	"sync"

	"github.com/btcsuite/btcd/btcutil/hdkeychain"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/tyler-smith/go-bip39"
)

// DefaultRootDerivationPath is the root path to which custom derivation endpoints
// are appended. As such, the first account will be at m/44'/60'/0'/0, the second
// at m/44'/60'/0'/1, etc.
var DefaultRootDerivationPath = accounts.DefaultRootDerivationPath

// DefaultBaseDerivationPath is the base path from which custom derivation endpoints
// are incremented. As such, the first account will be at m/44'/60'/0'/0, the second
// at m/44'/60'/0'/1, etc
var DefaultBaseDerivationPath = accounts.DefaultBaseDerivationPath

const issue179FixEnvar = "GO_ETHEREUM_HDWALLET_FIX_ISSUE_179"

// Wallet is the underlying wallet struct.
type Wallet struct {
	mnemonic    string
	masterKey   *hdkeychain.ExtendedKey
	seed        []byte
	url         accounts.URL
	paths       map[common.Address]accounts.DerivationPath
	accounts    []accounts.Account
	stateLock   sync.RWMutex
	fixIssue172 bool
}

func newWallet(seed []byte) (*Wallet, error) {
	masterKey, err := hdkeychain.NewMaster(seed, &chaincfg.MainNetParams)
	if err != nil {
		return nil, err
	}

	return &Wallet{
		masterKey:   masterKey,
		seed:        seed,
		accounts:    []accounts.Account{},
		paths:       map[common.Address]accounts.DerivationPath{},
		fixIssue172: false || len(os.Getenv(issue179FixEnvar)) > 0,
	}, nil
}

// NewFromMnemonic returns a new wallet from a BIP-39 mnemonic.
func NewFromMnemonic(mnemonic string) (*Wallet, error) {
	if mnemonic == "" {
		return nil, errors.New("mnemonic is required")
	}

	if !bip39.IsMnemonicValid(mnemonic) {
		return nil, errors.New("mnemonic is invalid")
	}

	seed, err := NewSeedFromMnemonic(mnemonic)
	if err != nil {
		return nil, err
	}

	wallet, err := newWallet(seed)
	if err != nil {
		return nil, err
	}
	wallet.mnemonic = mnemonic

	return wallet, nil
}

// NewFromSeed returns a new wallet from a BIP-39 seed.
func NewFromSeed(seed []byte) (*Wallet, error) {
	if len(seed) == 0 {
		return nil, errors.New("seed is required")
	}

	return newWallet(seed)
}

// URL implements accounts.Wallet, returning the URL of the device that
// the wallet is on, however this does nothing since this is not a hardware device.
func (w *Wallet) URL() accounts.URL {
	return w.url
}

// Status implements accounts.Wallet, returning a custom status message
// from the underlying vendor-specific hardware wallet implementation,
// however this does nothing since this is not a hardware device.
func (w *Wallet) Status() (string, error) {
	return "ok", nil
}

// Open implements accounts.Wallet, however this does nothing since this
// is not a hardware device.
func (w *Wallet) Open(passphrase string) error {
	return nil
}

// Close implements accounts.Wallet, however this does nothing since this
// is not a hardware device.
func (w *Wallet) Close() error {
	return nil
}

// Accounts implements accounts.Wallet, returning the list of accounts pinned to
// the wallet. If self-derivation was enabled, the account list is
// periodically expanded based on current chain state.
func (w *Wallet) Accounts() []accounts.Account {
	// Attempt self-derivation if it's running
	// Return whatever account list we ended up with
	w.stateLock.RLock()
	defer w.stateLock.RUnlock()

	cpy := make([]accounts.Account, len(w.accounts))
	copy(cpy, w.accounts)
	return cpy
}

// Contains implements accounts.Wallet, returning whether a particular account is
// or is not pinned into this wallet instance.
func (w *Wallet) Contains(account accounts.Account) bool {
	w.stateLock.RLock()
	defer w.stateLock.RUnlock()

	_, exists := w.paths[account.Address]
	return exists
}

// Unpin unpins account from list of pinned accounts.
func (w *Wallet) Unpin(account accounts.Account) error {
	w.stateLock.RLock()
	defer w.stateLock.RUnlock()

	for i, acct := range w.accounts {
		if acct.Address.String() == account.Address.String() {
			w.accounts = removeAtIndex(w.accounts, i)
			delete(w.paths, account.Address)
			return nil
		}
	}

	return errors.New("account not found")
}

// SetFixIssue172 determines whether the standard (correct) bip39
// derivation path was used, or if derivation should be affected by
// Issue172 [0] which was how this library was originally implemented.
// [0] https://github.com/btcsuite/btcutil/pull/182/files
func (w *Wallet) SetFixIssue172(fixIssue172 bool) {
	w.fixIssue172 = fixIssue172
}

// Derive implements accounts.Wallet, deriving a new account at the specific
// derivation path. If pin is set to true, the account will be added to the list
// of tracked accounts.
func (w *Wallet) Derive(path accounts.DerivationPath, pin bool) (accounts.Account, error) {
	// Try to derive the actual account and update its URL if successful
	w.stateLock.RLock() // Avoid device disappearing during derivation

	address, err := w.deriveAddress(path)

	w.stateLock.RUnlock()

	// If an error occurred or no pinning was requested, return
	if err != nil {
		return accounts.Account{}, err
	}

	account := accounts.Account{
		Address: address,
		URL: accounts.URL{
			Scheme: "",
			Path:   path.String(),
		},
	}

	if !pin {
		return account, nil
	}

	// Pinning needs to modify the state
	w.stateLock.Lock()
	defer w.stateLock.Unlock()

	if _, ok := w.paths[address]; !ok {
		w.accounts = append(w.accounts, account)
		w.paths[address] = path
	}

	return account, nil
}

// SelfDerive implements accounts.Wallet, trying to discover accounts that the
// user used previously (based on the chain state), but ones that he/she did not
// explicitly pin to the wallet manually. To avoid chain head monitoring, self
// derivation only runs during account listing (and even then throttled).
func (w *Wallet) SelfDerive(base []accounts.DerivationPath, chain ethereum.ChainStateReader) {
	// TODO: self derivation
}

// SignHash implements accounts.Wallet, which allows signing arbitrary data.
func (w *Wallet) SignHash(account accounts.Account, hash []byte) ([]byte, error) {
	// Make sure the requested account is contained within
	path, ok := w.paths[account.Address]
	if !ok {
		return nil, accounts.ErrUnknownAccount
	}

	privateKey, err := w.derivePrivateKey(path)
	if err != nil {
		return nil, err
	}

	return crypto.Sign(hash, privateKey)
}

// SignTxEIP155 implements accounts.Wallet, which allows the account to sign an ERC-20 transaction.
func (w *Wallet) SignTxEIP155(account accounts.Account, tx *types.Transaction, chainID *big.Int) (*types.Transaction, error) {
	w.stateLock.RLock() // Comms have own mutex, this is for the state fields
	defer w.stateLock.RUnlock()

	// Make sure the requested account is contained within
	path, ok := w.paths[account.Address]
	if !ok {
		return nil, accounts.ErrUnknownAccount
	}

	privateKey, err := w.derivePrivateKey(path)
	if err != nil {
		return nil, err
	}

	signer := types.NewEIP155Signer(chainID)
	// Sign the transaction and verify the sender to avoid hardware fault surprises
	signedTx, err := types.SignTx(tx, signer, privateKey)
	if err != nil {
		return nil, err
	}

	sender, err := types.Sender(signer, signedTx)
	if err != nil {
		return nil, err
	}

	if sender != account.Address {
		return nil, fmt.Errorf("signer mismatch: expected %s, got %s", account.Address.Hex(), sender.Hex())
	}

	return signedTx, nil
}

// SignTx implements accounts.Wallet, which allows the account to sign an Ethereum transaction.
func (w *Wallet) SignTx(account accounts.Account, tx *types.Transaction, chainID *big.Int) (*types.Transaction, error) {
	w.stateLock.RLock() // Comms have own mutex, this is for the state fields
	defer w.stateLock.RUnlock()

	// Make sure the requested account is contained within
	path, ok := w.paths[account.Address]
	if !ok {
		return nil, accounts.ErrUnknownAccount
	}

	privateKey, err := w.derivePrivateKey(path)
	if err != nil {
		return nil, err
	}

	signer := types.LatestSignerForChainID(chainID)

	// Sign the transaction and verify the sender to avoid hardware fault surprises
	signedTx, err := types.SignTx(tx, signer, privateKey)
	if err != nil {
		return nil, err
	}

	sender, err := types.Sender(signer, signedTx)
	if err != nil {
		return nil, err
	}

	if sender != account.Address {
		return nil, fmt.Errorf("signer mismatch: expected %s, got %s", account.Address.Hex(), sender.Hex())
	}

	return signedTx, nil
}

// SignHashWithPassphrase implements accounts.Wallet, attempting
// to sign the given hash with the given account using the
// passphrase as extra authentication.
func (w *Wallet) SignHashWithPassphrase(account accounts.Account, passphrase string, hash []byte) ([]byte, error) {
	return w.SignHash(account, hash)
}

// SignTxWithPassphrase implements accounts.Wallet, attempting to sign the given
// transaction with the given account using passphrase as extra authentication.
func (w *Wallet) SignTxWithPassphrase(account accounts.Account, passphrase string, tx *types.Transaction, chainID *big.Int) (*types.Transaction, error) {
	return w.SignTx(account, tx, chainID)
}

// PrivateKey returns the ECDSA private key of the account.
func (w *Wallet) PrivateKey(account accounts.Account) (*ecdsa.PrivateKey, error) {
	path, err := ParseDerivationPath(account.URL.Path)
	if err != nil {
		return nil, err
	}

	return w.derivePrivateKey(path)
}

// PrivateKeyBytes returns the ECDSA private key in bytes format of the account.
func (w *Wallet) PrivateKeyBytes(account accounts.Account) ([]byte, error) {
	privateKey, err := w.PrivateKey(account)
	if err != nil {
		return nil, err
	}

	return crypto.FromECDSA(privateKey), nil
}

// PrivateKeyHex return the ECDSA private key in hex string format of the account.
func (w *Wallet) PrivateKeyHex(account accounts.Account) (string, error) {
	privateKeyBytes, err := w.PrivateKeyBytes(account)
	if err != nil {
		return "", err
	}

	return hexutil.Encode(privateKeyBytes)[2:], nil
}

// PublicKey returns the ECDSA public key of the account.
func (w *Wallet) PublicKey(account accounts.Account) (*ecdsa.PublicKey, error) {
	path, err := ParseDerivationPath(account.URL.Path)
	if err != nil {
		return nil, err
	}

	return w.derivePublicKey(path)
}

// PublicKeyBytes returns the ECDSA public key in bytes format of the account.
func (w *Wallet) PublicKeyBytes(account accounts.Account) ([]byte, error) {
	publicKey, err := w.PublicKey(account)
	if err != nil {
		return nil, err
	}

	return crypto.FromECDSAPub(publicKey), nil
}

// PublicKeyHex return the ECDSA public key in hex string format of the account.
func (w *Wallet) PublicKeyHex(account accounts.Account) (string, error) {
	publicKeyBytes, err := w.PublicKeyBytes(account)
	if err != nil {
		return "", err
	}

	return hexutil.Encode(publicKeyBytes)[4:], nil
}

// Address returns the address of the account.
func (w *Wallet) Address(account accounts.Account) (common.Address, error) {
	publicKey, err := w.PublicKey(account)
	if err != nil {
		return common.Address{}, err
	}

	return crypto.PubkeyToAddress(*publicKey), nil
}

// AddressBytes returns the address in bytes format of the account.
func (w *Wallet) AddressBytes(account accounts.Account) ([]byte, error) {
	address, err := w.Address(account)
	if err != nil {
		return nil, err
	}
	return address.Bytes(), nil
}

// AddressHex returns the address in hex string format of the account.
func (w *Wallet) AddressHex(account accounts.Account) (string, error) {
	address, err := w.Address(account)
	if err != nil {
		return "", err
	}
	return address.Hex(), nil
}

// Path return the derivation path of the account.
func (w *Wallet) Path(account accounts.Account) (string, error) {
	return account.URL.Path, nil
}

// SignData signs keccak256(data). The mimetype parameter describes the type of data being signed
func (w *Wallet) SignData(account accounts.Account, mimeType string, data []byte) ([]byte, error) {
	// Make sure the requested account is contained within
	if !w.Contains(account) {
		return nil, accounts.ErrUnknownAccount
	}

	return w.SignHash(account, crypto.Keccak256(data))
}

// SignDataWithPassphrase signs keccak256(data). The mimetype parameter describes the type of data being signed
func (w *Wallet) SignDataWithPassphrase(account accounts.Account, passphrase, mimeType string, data []byte) ([]byte, error) {
	// Make sure the requested account is contained within
	if !w.Contains(account) {
		return nil, accounts.ErrUnknownAccount
	}

	return w.SignHashWithPassphrase(account, passphrase, crypto.Keccak256(data))
}

// SignText requests the wallet to sign the hash of a given piece of data, prefixed
// the needed details via SignHashWithPassphrase, or by other means (e.g. unlock
// the account in a keystore).
func (w *Wallet) SignText(account accounts.Account, text []byte) ([]byte, error) {
	// Make sure the requested account is contained within
	if !w.Contains(account) {
		return nil, accounts.ErrUnknownAccount
	}

	return w.SignHash(account, accounts.TextHash(text))
}

// SignTextWithPassphrase implements accounts.Wallet, attempting to sign the
// given text (which is hashed) with the given account using passphrase as extra authentication.
func (w *Wallet) SignTextWithPassphrase(account accounts.Account, passphrase string, text []byte) ([]byte, error) {
	// Make sure the requested account is contained within
	if !w.Contains(account) {
		return nil, accounts.ErrUnknownAccount
	}

	return w.SignHashWithPassphrase(account, passphrase, accounts.TextHash(text))
}

// ParseDerivationPath parses the derivation path in string format into []uint32
func ParseDerivationPath(path string) (accounts.DerivationPath, error) {
	return accounts.ParseDerivationPath(path)
}

// MustParseDerivationPath parses the derivation path in string format into
// []uint32 but will panic if it can't parse it.
func MustParseDerivationPath(path string) accounts.DerivationPath {
	parsed, err := accounts.ParseDerivationPath(path)
	if err != nil {
		panic(err)
	}

	return parsed
}

// NewMnemonic returns a randomly generated BIP-39 mnemonic using 128-256 bits of entropy.
func NewMnemonic(bits int) (string, error) {
	entropy, err := bip39.NewEntropy(bits)
	if err != nil {
		return "", err
	}
	return bip39.NewMnemonic(entropy)
}

// NewMnemonicFromEntropy returns a BIP-39 mnemonic from entropy.
func NewMnemonicFromEntropy(entropy []byte) (string, error) {
	return bip39.NewMnemonic(entropy)
}

// NewEntropy returns a randomly generated entropy.
func NewEntropy(bits int) ([]byte, error) {
	return bip39.NewEntropy(bits)
}

// NewSeed returns a randomly generated BIP-39 seed.
func NewSeed() ([]byte, error) {
	b := make([]byte, 64)
	_, err := rand.Read(b)
	return b, err
}

// NewSeedFromMnemonic returns a BIP-39 seed based on a BIP-39 mnemonic.
func NewSeedFromMnemonic(mnemonic string) ([]byte, error) {
	if mnemonic == "" {
		return nil, errors.New("mnemonic is required")
	}

	return bip39.NewSeedWithErrorChecking(mnemonic, "")
}

// DerivePrivateKey derives the private key of the derivation path.
func (w *Wallet) derivePrivateKey(path accounts.DerivationPath) (*ecdsa.PrivateKey, error) {
	var err error
	key := w.masterKey
	for _, n := range path {
		if w.fixIssue172 && key.IsAffectedByIssue172() {
			key, err = key.Derive(n)
		} else {
			key, err = key.DeriveNonStandard(n)
		}
		if err != nil {
			return nil, err
		}
	}

	privateKey, err := key.ECPrivKey()
	privateKeyECDSA := privateKey.ToECDSA()
	if err != nil {
		return nil, err
	}

	return privateKeyECDSA, nil
}

// DerivePublicKey derives the public key of the derivation path.
func (w *Wallet) derivePublicKey(path accounts.DerivationPath) (*ecdsa.PublicKey, error) {
	privateKeyECDSA, err := w.derivePrivateKey(path)
	if err != nil {
		return nil, err
	}

	publicKey := privateKeyECDSA.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		return nil, errors.New("failed to get public key")
	}

	return publicKeyECDSA, nil
}

// DeriveAddress derives the account address of the derivation path.
func (w *Wallet) deriveAddress(path accounts.DerivationPath) (common.Address, error) {
	publicKeyECDSA, err := w.derivePublicKey(path)
	if err != nil {
		return common.Address{}, err
	}

	address := crypto.PubkeyToAddress(*publicKeyECDSA)
	return address, nil
}

// removeAtIndex removes an account at index.
func removeAtIndex(accts []accounts.Account, index int) []accounts.Account {
	return append(accts[:index], accts[index+1:]...)
}
