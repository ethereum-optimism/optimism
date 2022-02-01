package bsscore_test

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	bsscore "github.com/ethereum-optimism/optimism/go/bss-core"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
)

var (
	validMnemonic = strings.Join([]string{
		"abandon", "abandon", "abandon", "abandon",
		"abandon", "abandon", "abandon", "abandon",
		"abandon", "abandon", "abandon", "about",
	}, " ")

	validHDPath = "m/44'/60'/0'/128"

	// validPrivKeyStr is the private key string for the child derived at
	// validHDPath from validMnemonic.
	validPrivKeyStr = "69d3a0e79bf039ca788924cb98b6b60c5f5aaa5e770aef09b4b15fdb59944d02"

	// validPrivKeyBytes is the raw private key bytes for the child derived
	// at validHDPath from validMnemonic.
	validPrivKeyBytes = []byte{
		0x69, 0xd3, 0xa0, 0xe7, 0x9b, 0xf0, 0x39, 0xca,
		0x78, 0x89, 0x24, 0xcb, 0x98, 0xb6, 0xb6, 0x0c,
		0x5f, 0x5a, 0xaa, 0x5e, 0x77, 0x0a, 0xef, 0x09,
		0xb4, 0xb1, 0x5f, 0xdb, 0x59, 0x94, 0x4d, 0x02,
	}

	// invalidMnemonic has an invalid checksum.
	invalidMnemonic = strings.Join([]string{
		"abandon", "abandon", "abandon", "abandon",
		"abandon", "abandon", "abandon", "abandon",
		"abandon", "abandon", "abandon", "abandon",
	}, " ")
)

// TestParseAddress asserts that ParseAddress correctly parses 40-characater
// hexidecimal strings with optional 0x prefix into valid 20-byte addresses.
func TestParseAddress(t *testing.T) {
	tests := []struct {
		name    string
		addr    string
		expErr  error
		expAddr common.Address
	}{
		{
			name:   "empty address",
			addr:   "",
			expErr: errors.New("invalid address: "),
		},
		{
			name:   "only 0x",
			addr:   "0x",
			expErr: errors.New("invalid address: 0x"),
		},
		{
			name:   "non hex character",
			addr:   "0xaaaaaazaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			expErr: errors.New("invalid address: 0xaaaaaazaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"),
		},
		{
			name:    "valid address",
			addr:    "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			expErr:  nil,
			expAddr: common.BytesToAddress(bytes.Repeat([]byte{170}, 20)),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			addr, err := bsscore.ParseAddress(test.addr)
			require.Equal(t, err, test.expErr)
			if test.expErr != nil {
				return
			}
			require.Equal(t, addr, test.expAddr)
		})
	}
}

// TestDerivePrivateKey asserts that DerivePrivateKey properly parses a BIP39
// mnemonic and BIP32 HD path, and derives the corresponding private key.
func TestDerivePrivateKey(t *testing.T) {
	tests := []struct {
		name       string
		mnemonic   string
		hdPath     string
		expErr     error
		expPrivKey []byte
	}{
		{
			name:     "invalid mnemonic",
			mnemonic: invalidMnemonic,
			hdPath:   validHDPath,
			expErr:   errors.New("Checksum incorrect"),
		},
		{
			name:     "valid mnemonic invalid hdpath",
			mnemonic: validMnemonic,
			hdPath:   "",
			expErr: errors.New("ambiguous path: use 'm/' prefix for absolute " +
				"paths, or no leading '/' for relative ones"),
		},
		{
			name:     "valid mnemonic invalid hdpath",
			mnemonic: validMnemonic,
			hdPath:   "m/",
			expErr:   errors.New("invalid component: "),
		},
		{
			name:     "valid mnemonic valid hdpath no components",
			mnemonic: validMnemonic,
			hdPath:   "m/0",
			expPrivKey: []byte{
				0xba, 0xa8, 0x9a, 0x8b, 0xdd, 0x61, 0xc5, 0xe2,
				0x2b, 0x9f, 0x10, 0x60, 0x1d, 0x87, 0x91, 0xc9,
				0xf8, 0xfc, 0x4b, 0x2f, 0xa6, 0xdf, 0x9d, 0x68,
				0xd3, 0x36, 0xf0, 0xeb, 0x03, 0xb0, 0x6e, 0xb6,
			},
		},
		{
			name:     "valid mnemonic valid hdpath full path",
			mnemonic: validMnemonic,
			hdPath:   validHDPath,
			expPrivKey: []byte{
				0x69, 0xd3, 0xa0, 0xe7, 0x9b, 0xf0, 0x39, 0xca,
				0x78, 0x89, 0x24, 0xcb, 0x98, 0xb6, 0xb6, 0x0c,
				0x5f, 0x5a, 0xaa, 0x5e, 0x77, 0x0a, 0xef, 0x09,
				0xb4, 0xb1, 0x5f, 0xdb, 0x59, 0x94, 0x4d, 0x02,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			privKey, err := bsscore.DerivePrivateKey(test.mnemonic, test.hdPath)
			require.Equal(t, err, test.expErr)
			if test.expErr != nil {
				return
			}

			expPrivKey, err := crypto.ToECDSA(test.expPrivKey)
			require.Nil(t, err)
			require.Equal(t, privKey, expPrivKey)
		})
	}
}

// TestParsePrivateKeyStr asserts that ParsePrivateKey properly parses
// 64-character hexidecimal strings with optional 0x prefix into valid ECDSA
// private keys.
func TestParsePrivateKeyStr(t *testing.T) {
	tests := []struct {
		name       string
		privKeyStr string
		expErr     error
		expPrivKey []byte
	}{
		{
			name:       "empty privkey string",
			privKeyStr: "",
			expErr:     errors.New("invalid length, need 256 bits"),
		},
		{
			name:       "privkey string only 0x",
			privKeyStr: "0x",
			expErr:     errors.New("invalid length, need 256 bits"),
		},
		{
			name:       "non hex privkey string",
			privKeyStr: "0xaaaazaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			expErr:     errors.New("invalid hex character 'z' in private key"),
		},
		{
			name:       "valid privkey string",
			privKeyStr: validPrivKeyStr,
			expPrivKey: validPrivKeyBytes,
		},
		{
			name:       "valid privkey string with 0x",
			privKeyStr: "0x" + validPrivKeyStr,
			expPrivKey: validPrivKeyBytes,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			privKey, err := bsscore.ParsePrivateKeyStr(test.privKeyStr)
			require.Equal(t, err, test.expErr)
			if test.expErr != nil {
				return
			}

			expPrivKey, err := crypto.ToECDSA(test.expPrivKey)
			require.Nil(t, err)
			require.Equal(t, privKey, expPrivKey)
		})
	}
}

// TestGetConfiguredPrivateKey asserts that GetConfiguredPrivateKey either:
//  1) Derives the correct private key assuming the BIP39 mnemonic and BIP32
//     derivation path are both present and the private key string is ommitted.
//  2) Parses the correct private key assuming only the private key string is
//     present, but the BIP39 mnemonic and BIP32 derivation path are ommitted.
func TestGetConfiguredPrivateKey(t *testing.T) {
	tests := []struct {
		name       string
		mnemonic   string
		hdPath     string
		privKeyStr string
		expErr     error
		expPrivKey []byte
	}{
		{
			name:       "valid mnemonic+hdpath",
			mnemonic:   validMnemonic,
			hdPath:     validHDPath,
			privKeyStr: "",
			expPrivKey: validPrivKeyBytes,
		},
		{
			name:       "valid privkey",
			mnemonic:   "",
			hdPath:     "",
			privKeyStr: validPrivKeyStr,
			expPrivKey: validPrivKeyBytes,
		},
		{
			name:       "valid privkey with 0x",
			mnemonic:   "",
			hdPath:     "",
			privKeyStr: "0x" + validPrivKeyStr,
			expPrivKey: validPrivKeyBytes,
		},
		{
			name:       "valid menmonic+hdpath and privkey",
			mnemonic:   validMnemonic,
			hdPath:     validHDPath,
			privKeyStr: validPrivKeyStr,
			expErr:     bsscore.ErrCannotGetPrivateKey,
		},
		{
			name:       "neither menmonic+hdpath or privkey",
			mnemonic:   "",
			hdPath:     "",
			privKeyStr: "",
			expErr:     bsscore.ErrCannotGetPrivateKey,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			privKey, err := bsscore.GetConfiguredPrivateKey(
				test.mnemonic, test.hdPath, test.privKeyStr,
			)
			require.Equal(t, err, test.expErr)
			if test.expErr != nil {
				return
			}

			expPrivKey, err := crypto.ToECDSA(test.expPrivKey)
			require.Nil(t, err)
			require.Equal(t, privKey, expPrivKey)
		})
	}
}
