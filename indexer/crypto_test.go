package indexer_test

import (
	"bytes"
	"errors"
	"testing"

	indexer "github.com/ethereum-optimism/optimism/indexer"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

// TestParseAddress asserts that ParseAddress correctly parses
// 40-characater hexadecimal strings with optional 0x prefix into valid 20-byte
// addresses
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
			addr, err := indexer.ParseAddress(test.addr)
			require.Equal(t, err, test.expErr)
			if test.expErr != nil {
				return
			}
			require.Equal(t, addr, test.expAddr)
		})
	}
}
