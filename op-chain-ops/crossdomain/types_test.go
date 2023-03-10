package crossdomain

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestInvalidMessage(t *testing.T) {
	tests := []struct {
		name string
		msg  InvalidMessage
		slot common.Hash
	}{
		{
			name: "unparseable x-domain message on mainnet",
			msg: InvalidMessage{
				Who: common.HexToAddress("0x8b1d477410344785ff1df52500032e6d5f532ee4"),
				Msg: common.FromHex("0x042069"),
			},
			slot: common.HexToHash("0x2a49ae6579c3878f10cf87ecdbebc6c4e2b2159ffe2b1af88af6ca9697fc32cb"),
		},
		{
			name: "valid x-domain message on mainnet for validation",
			msg: InvalidMessage{
				Who: common.HexToAddress("0x4200000000000000000000000000000000000007"),
				Msg: common.FromHex("" +
					"0xcbd4ece900000000000000000000000099c9fc46f92e8a1c0dec1b1747d01090" +
					"3e884be100000000000000000000000042000000000000000000000000000000" +
					"0000001000000000000000000000000000000000000000000000000000000000" +
					"0000008000000000000000000000000000000000000000000000000000000000" +
					"00019be200000000000000000000000000000000000000000000000000000000" +
					"000000e4a9f9e675000000000000000000000000a0b86991c6218b36c1d19d4a" +
					"2e9eb0ce3606eb480000000000000000000000007f5c764cbc14f9669b88837c" +
					"a1490cca17c31607000000000000000000000000a420b2d1c0841415a695b81e" +
					"5b867bcd07dff8c9000000000000000000000000c186fa914353c44b2e33ebe0" +
					"5f21846f1048beda000000000000000000000000000000000000000000000000" +
					"00000000295d681d000000000000000000000000000000000000000000000000" +
					"00000000000000c0000000000000000000000000000000000000000000000000" +
					"0000000000000000000000000000000000000000000000000000000000000000" +
					"00000000",
				),
			},
			slot: common.HexToHash("0x8f8f6be7a4c5048f46ca41897181d17c10c39365ead5ac27c23d1e8e466d0ed5"),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// StorageSlot() tests Hash() and Encode() so we don't
			// need to test these separately.
			slot, err := test.msg.StorageSlot()
			require.NoError(t, err)
			require.Equal(t, test.slot, slot)
		})
	}
}
