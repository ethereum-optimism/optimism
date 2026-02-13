package eth

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

func TestAddressAsLeftPaddedHash(t *testing.T) {
	// Test cases with different addresses
	testCases := []struct {
		name   string
		addr   common.Address
		expect common.Hash
	}{
		{
			name:   "empty address",
			addr:   common.Address{},
			expect: common.HexToHash("0x0000000000000000000000000000000000000000"),
		},
		{
			name:   "simple address",
			addr:   common.HexToAddress("0x1234567890AbCdEf1234567890AbCdEf"),
			expect: common.HexToHash("0x000000000000000000000000000000001234567890abcdef1234567890abcdef"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			output := AddressAsLeftPaddedHash(tc.addr)
			if output != tc.expect {
				t.Fatalf("Expected output %v for test case %s, got %v", tc.expect, tc.name, output)
			}
		})
	}
}
