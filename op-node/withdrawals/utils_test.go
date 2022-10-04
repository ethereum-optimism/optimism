package withdrawals

import (
	"encoding/json"
	"math/big"
	"os"
	"path"
	"testing"

	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
	"github.com/ethereum-optimism/optimism/op-node/testutils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

func TestParseMessagePassed(t *testing.T) {
	tests := []struct {
		name     string
		file     string
		expected *bindings.L2ToL1MessagePasserMessagePassed
	}{
		{
			"withdrawal through bridge",
			"bridge-withdrawal.json",
			&bindings.L2ToL1MessagePasserMessagePassed{
				Nonce:    new(big.Int),
				Sender:   common.HexToAddress("0x4200000000000000000000000000000000000007"),
				Target:   common.HexToAddress("0x6900000000000000000000000000000000000002"),
				Value:    new(big.Int),
				GasLimit: big.NewInt(203648),
				Data: hexutil.MustDecode(
					"0xd764ad0b00010000000000000000000000000000000000000000000000000000" +
						"0000000000000000000000000000000042000000000000000000000000000000" +
						"0000001000000000000000000000000069000000000000000000000000000000" +
						"0000000300000000000000000000000000000000000000000000000000000000" +
						"0000000000000000000000000000000000000000000000000000000000000000" +
						"0000000000000000000000000000000000000000000000000000000000000000" +
						"000000c000000000000000000000000000000000000000000000000000000000" +
						"000000e40166a07a00000000000000000000000089d51be807d98fc974a0f41b" +
						"2e67a8228d7846ef0000000000000000000000007c6b91d9be155a6db01f7492" +
						"17d76ff02a7227f2000000000000000000000000c20c5ec92fda6e611a084851" +
						"23cdc0d5b84bd3a2000000000000000000000000c20c5ec92fda6e611a084851" +
						"23cdc0d5b84bd3a2000000000000000000000000000000000000000000000000" +
						"00000000000001f4000000000000000000000000000000000000000000000000" +
						"00000000000000c0000000000000000000000000000000000000000000000000" +
						"0000000000000000000000000000000000000000000000000000000000000000" +
						"00000000",
				),
				Raw: types.Log{
					Address: common.HexToAddress("0x4200000000000000000000000000000000000016"),
					Topics: []common.Hash{
						common.HexToHash("0x87bf7b546c8de873abb0db5b579ec131f8d0cf5b14f39933551cf9ced23a6136"),
						common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000000"),
						common.HexToHash("0x0000000000000000000000004200000000000000000000000000000000000007"),
						common.HexToHash("0x0000000000000000000000006900000000000000000000000000000000000002"),
					},
					Data: hexutil.MustDecode(
						"0x00000000000000000000000000000000000000000000000000000000000000" +
							"000000000000000000000000000000000000000000000000000000000000031b80" +
							"000000000000000000000000000000000000000000000000000000000000006000" +
							"000000000000000000000000000000000000000000000000000000000001e4d764" +
							"ad0b00010000000000000000000000000000000000000000000000000000000000" +
							"000000000000000000000000004200000000000000000000000000000000000010" +
							"000000000000000000000000690000000000000000000000000000000000000300" +
							"000000000000000000000000000000000000000000000000000000000000000000" +
							"000000000000000000000000000000000000000000000000000000000000000000" +
							"00000000000000000000000000000000000000000000000000000000c000000000" +
							"000000000000000000000000000000000000000000000000000000e40166a07a00" +
							"000000000000000000000089d51be807d98fc974a0f41b2e67a8228d7846ef0000" +
							"000000000000000000007c6b91d9be155a6db01f749217d76ff02a7227f2000000" +
							"000000000000000000c20c5ec92fda6e611a08485123cdc0d5b84bd3a200000000" +
							"0000000000000000c20c5ec92fda6e611a08485123cdc0d5b84bd3a20000000000" +
							"0000000000000000000000000000000000000000000000000001f4000000000000" +
							"00000000000000000000000000000000000000000000000000c000000000000000" +
							"000000000000000000000000000000000000000000000000000000000000000000" +
							"000000000000000000000000000000000000000000000000000000000000000000" +
							"000000000000000000000000000000",
					),
					BlockNumber: 0x36,
					TxHash:      common.HexToHash("0x9346381068b59d2098495baa72ed2f773c1e09458610a7a208984859dff73add"),
					TxIndex:     1,
					BlockHash:   common.HexToHash("0xfdd4ad8a984b45687aca0463db491cbd0e85273d970019a3f8bf618b614938df"),
					Index:       2,
					Removed:     false,
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			f, err := os.Open(path.Join("testdata", test.file))
			require.NoError(t, err)
			dec := json.NewDecoder(f)
			receipt := new(types.Receipt)
			require.NoError(t, dec.Decode(receipt))
			parsed, err := ParseMessagePassed(receipt)
			require.NoError(t, err)

			// Have to do this weird thing to compare zero bigints.
			// When they're deserialized from JSON, the internal byte
			// array is an empty array whereas it is nil in the expectation.
			parsedNonce := parsed.Nonce
			parsedValue := parsed.Value
			expNonce := test.expected.Nonce
			expValue := test.expected.Value
			testutils.RequireBigEqual(t, expNonce, parsedNonce)
			testutils.RequireBigEqual(t, expValue, parsedValue)
			parsed.Nonce = nil
			parsed.Value = nil
			test.expected.Nonce = nil
			test.expected.Value = nil

			require.EqualValues(t, test.expected, parsed)
		})
	}
}

func TestParseMessagePassedExtension1(t *testing.T) {
	tests := []struct {
		name     string
		file     string
		expected *bindings.L2ToL1MessagePasserMessagePassedExtension1
	}{
		{
			"withdrawal through bridge",
			"bridge-withdrawal.json",
			&bindings.L2ToL1MessagePasserMessagePassedExtension1{
				Hash: common.HexToHash("0x0d827f8148288e3a2466018f71b968ece4ea9f9e2a81c30da9bd46cce2868285"),
				Raw: types.Log{
					Address: common.HexToAddress("0x4200000000000000000000000000000000000016"),
					Topics: []common.Hash{
						common.HexToHash("0x2ef6ceb1668fdd882b1f89ddd53a666b0c1113d14cf90c0fbf97c7b1ad880fbb"),
						common.HexToHash("0x0d827f8148288e3a2466018f71b968ece4ea9f9e2a81c30da9bd46cce2868285"),
					},
					Data:        []byte{},
					BlockNumber: 0x36,
					TxHash:      common.HexToHash("0x9346381068b59d2098495baa72ed2f773c1e09458610a7a208984859dff73add"),
					TxIndex:     0x1,
					BlockHash:   common.HexToHash("0xfdd4ad8a984b45687aca0463db491cbd0e85273d970019a3f8bf618b614938df"),
					Index:       0x3,
					Removed:     false,
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			f, err := os.Open(path.Join("testdata", test.file))
			require.NoError(t, err)
			dec := json.NewDecoder(f)
			receipt := new(types.Receipt)
			require.NoError(t, dec.Decode(receipt))
			parsed, err := ParseMessagePassedExtension1(receipt)
			require.NoError(t, err)
			require.EqualValues(t, test.expected, parsed)
		})
	}
}
