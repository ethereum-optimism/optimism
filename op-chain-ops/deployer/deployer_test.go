package deployer

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

func TestCreate2Address(t *testing.T) {
	tests := []struct {
		name            string
		creatorAddress  []byte
		salt            []byte
		initCode        []byte
		expectedAddress common.Address
	}{
		{
			name:            "SafeL2",
			creatorAddress:  common.HexToAddress("0x4e59b44847b379578588920cA78FbF26c0B4956C").Bytes(),
			salt:            common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000000"),
			expectedAddress: common.HexToAddress("0x3E5c63644E683549055b9Be8653de26E0B4CD36E"),
		},
		{
			name:            "MultiSendCallOnly",
			creatorAddress:  common.HexToAddress("0x4e59b44847b379578588920cA78FbF26c0B4956C").Bytes(),
			salt:            common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000000"),
			expectedAddress: common.HexToAddress("0x40A2aCCbd92BCA938b02010E17A5b8929b49130D"),
		},
		{
			name:            "MultiSend",
			creatorAddress:  common.HexToAddress("0x4e59b44847b379578588920cA78FbF26c0B4956C").Bytes(),
			salt:            common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000000"),
			expectedAddress: common.HexToAddress("0xA238CBeb142c10Ef7Ad8442C6D1f9E89e07e7761"),
		},
		{
			name:            "Permit2",
			creatorAddress:  common.HexToAddress("0x4e59b44847b379578588920cA78FbF26c0B4956C").Bytes(),
			salt:            common.Hex2Bytes("0000000000000000000000000000000000000000d3af2663da51c10215000000"),
			expectedAddress: common.HexToAddress("0x000000000022D473030F116dDEE9F6B43aC78BA3"),
		},
		{
			name:            "EntryPoint",
			creatorAddress:  common.HexToAddress("0x4e59b44847b379578588920cA78FbF26c0B4956C").Bytes(),
			salt:            common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000000"),
			expectedAddress: common.HexToAddress("0x5FF137D4b0FDCD49DcA30c7CF57E578a026d2789"),
		},
	}

	for _, test := range tests {
		var err error
		test.initCode, err = getInitCode(test.name)
		if err != nil {
			t.Error(err)
		}

		t.Run(test.name, func(t *testing.T) {
			if got := create2Address(test.creatorAddress, test.salt, test.initCode); got != test.expectedAddress {
				t.Errorf("expected: %x, want: %x", got, test.expectedAddress)
			}
		})
	}
}
