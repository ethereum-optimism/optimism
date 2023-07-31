package crossdomain

import (
	"testing"

	"github.com/ledgerwatch/erigon-lib/common"
	"github.com/stretchr/testify/require"
)

func TestRead(t *testing.T) {
	witnesses, addresses, err := ReadWitnessData("testdata/witness.txt")
	require.NoError(t, err)

	require.Equal(t, []*SentMessage{
		{
			Who: common.HexToAddress("0x4200000000000000000000000000000000000007"),
			Msg: common.FromHex(
				"0xcbd4ece900000000000000000000000099c9fc46f92e8a1c0dec1b1747d01090" +
					"3e884be100000000000000000000000042000000000000000000000000000000" +
					"0000001000000000000000000000000000000000000000000000000000000000" +
					"0000008000000000000000000000000000000000000000000000000000000000" +
					"00019bd000000000000000000000000000000000000000000000000000000000" +
					"000000e4a9f9e675000000000000000000000000d533a949740bb3306d119cc7" +
					"77fa900ba034cd520000000000000000000000000994206dfe8de6ec6920ff4d" +
					"779b0d950605fb53000000000000000000000000e3a44dd2a8c108be56a78635" +
					"121ec914074da16d000000000000000000000000e3a44dd2a8c108be56a78635" +
					"121ec914074da16d0000000000000000000000000000000000000000000001b0" +
					"ac98ab3858d75478000000000000000000000000000000000000000000000000" +
					"00000000000000c0000000000000000000000000000000000000000000000000" +
					"0000000000000000000000000000000000000000000000000000000000000000" +
					"00000000",
			),
		},
		{
			Who: common.HexToAddress("0x8b1d477410344785ff1df52500032e6d5f532ee4"),
			Msg: common.FromHex("0x042069"),
		},
	}, witnesses)

	require.Equal(t, OVMETHAddresses{
		common.HexToAddress("0x6340d44c5174588B312F545eEC4a42f8a514eF50"): true,
	}, addresses)
}
