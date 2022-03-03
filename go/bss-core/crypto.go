package bsscore

import (
	"crypto/ecdsa"
	"errors"
	"fmt"
	"strings"

	"github.com/decred/dcrd/hdkeychain/v3"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/tyler-smith/go-bip39"
)

var (
	// ErrCannotGetPrivateKey signals that an both or neither combination of
	// mnemonic+hdpath or private key string was used in the configuration.
	ErrCannotGetPrivateKey = errors.New("invalid combination of privkey " +
		"or mnemonic+hdpath")
)

// ParseAddress parses an ETH addres from a hex string. This method will fail if
// the address is not a valid hexidecimal address.
func ParseAddress(address string) (common.Address, error) {
	if common.IsHexAddress(address) {
		return common.HexToAddress(address), nil
	}
	return common.Address{}, fmt.Errorf("invalid address: %v", address)
}

// GetConfiguredPrivateKey computes the private key for our configured services.
// The two supported methods are:
//  - Derived from BIP39 mnemonic and BIP32 HD derivation path.
//  - Directly from a serialized private key.
func GetConfiguredPrivateKey(mnemonic, hdPath, privKeyStr string) (
	*ecdsa.PrivateKey, error) {

	useMnemonic := mnemonic != "" && hdPath != ""
	usePrivKeyStr := privKeyStr != ""

	switch {
	case useMnemonic && !usePrivKeyStr:
		return DerivePrivateKey(mnemonic, hdPath)

	case usePrivKeyStr && !useMnemonic:
		return ParsePrivateKeyStr(privKeyStr)

	default:
		return nil, ErrCannotGetPrivateKey
	}
}

// fakeNetworkParams implements the hdkeychain.NetworkParams interface. These
// methods are unused in the child derivation, and only needed for serializing
// xpubs/xprivs which we don't rely on.
type fakeNetworkParams struct{}

func (f fakeNetworkParams) HDPrivKeyVersion() [4]byte {
	return [4]byte{}
}

func (f fakeNetworkParams) HDPubKeyVersion() [4]byte {
	return [4]byte{}
}

// DerivePrivateKey derives the private key from a given mnemonic and BIP32
// deriviation path.
func DerivePrivateKey(mnemonic, hdPath string) (*ecdsa.PrivateKey, error) {
	// Parse the seed string into the master BIP32 key.
	seed, err := bip39.NewSeedWithErrorChecking(mnemonic, "")
	if err != nil {
		return nil, err
	}

	privKey, err := hdkeychain.NewMaster(seed, fakeNetworkParams{})
	if err != nil {
		return nil, err
	}

	// Parse the derivation path and derive a child for each level of the
	// BIP32 derivation path.
	derivationPath, err := accounts.ParseDerivationPath(hdPath)
	if err != nil {
		return nil, err
	}

	for _, child := range derivationPath {
		privKey, err = privKey.Child(child)
		if err != nil {
			return nil, err
		}
	}

	rawPrivKey, err := privKey.SerializedPrivKey()
	if err != nil {
		return nil, err
	}

	return crypto.ToECDSA(rawPrivKey)
}

// ParsePrivateKeyStr parses a hexidecimal encoded private key, the encoding may
// optionally have an "0x" prefix.
func ParsePrivateKeyStr(privKeyStr string) (*ecdsa.PrivateKey, error) {
	hex := strings.TrimPrefix(privKeyStr, "0x")
	return crypto.HexToECDSA(hex)
}

// ParseWalletPrivKeyAndContractAddr returns the wallet private key to use for
// sending transactions as well as the contract address to send to for a
// particular sub-service.
func ParseWalletPrivKeyAndContractAddr(
	name string,
	mnemonic string,
	hdPath string,
	privKeyStr string,
	contractAddrStr string,
) (*ecdsa.PrivateKey, common.Address, error) {

	// Parse wallet private key from either privkey string or BIP39 mnemonic
	// and BIP32 HD derivation path.
	privKey, err := GetConfiguredPrivateKey(mnemonic, hdPath, privKeyStr)
	if err != nil {
		return nil, common.Address{}, err
	}

	// Parse the target contract address the wallet will send to.
	contractAddress, err := ParseAddress(contractAddrStr)
	if err != nil {
		return nil, common.Address{}, err
	}

	// Log wallet address rather than private key...
	walletAddress := crypto.PubkeyToAddress(privKey.PublicKey)

	log.Info(name+" wallet params parsed successfully", "wallet_address",
		walletAddress, "contract_address", contractAddress)

	return privKey, contractAddress, nil
}
