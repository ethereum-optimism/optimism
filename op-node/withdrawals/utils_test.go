package withdrawals

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"path"
	"testing"

	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-node/bindings"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum-optimism/optimism/op-service/testutils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

type testHeaderClient struct {
	headers []*types.Header
}

func newTestHeaderClient(timestamps ...uint64) *testHeaderClient {
	headers := make([]*types.Header, len(timestamps))
	for i, timestamp := range timestamps {
		headers[i] = &types.Header{
			Number: new(big.Int).SetUint64(uint64(i)),
			Time:   timestamp,
		}
	}
	return &testHeaderClient{headers: headers}
}

func (c *testHeaderClient) HeaderByNumber(ctx context.Context, number *big.Int) (*types.Header, error) {
	if number == nil {
		return c.headers[len(c.headers)-1], nil
	}
	if !number.IsUint64() {
		return nil, fmt.Errorf("block number does not fit in uint64: %v", number)
	}
	n := bigs.Uint64Strict(number)
	if n >= uint64(len(c.headers)) {
		return nil, fmt.Errorf("missing header %d", n)
	}
	return c.headers[n], nil
}

func TestFindL2HeaderForTimestamp(t *testing.T) {
	ctx := context.Background()
	client := newTestHeaderClient(10, 12, 14, 16)

	tests := []struct {
		name            string
		targetTimestamp uint64
		expectedNumber  uint64
	}{
		{
			name:            "exact timestamp",
			targetTimestamp: 14,
			expectedNumber:  2,
		},
		{
			name:            "between timestamps",
			targetTimestamp: 15,
			expectedNumber:  2,
		},
		{
			name:            "genesis timestamp",
			targetTimestamp: 10,
			expectedNumber:  0,
		},
		{
			name:            "latest timestamp",
			targetTimestamp: 16,
			expectedNumber:  3,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			header, err := FindL2HeaderForTimestamp(ctx, client, test.targetTimestamp)
			require.NoError(t, err)
			require.Equal(t, test.expectedNumber, bigs.Uint64Strict(header.Number))
		})
	}
}

func TestFindL2HeaderForTimestampErrors(t *testing.T) {
	ctx := context.Background()
	client := newTestHeaderClient(10, 12, 14, 16)

	_, err := FindL2HeaderForTimestamp(ctx, client, 9)
	require.ErrorContains(t, err, "no l2 header found at or before target timestamp 9")

	_, err = FindL2HeaderForTimestamp(ctx, client, 17)
	require.ErrorContains(t, err, "latest l2 header timestamp 16 is before target timestamp 17")
}

func TestGameSequenceAndOutputRoot(t *testing.T) {
	l2ChainID := big.NewInt(420120092)
	expectedRoot := common.HexToHash("0x1111111111111111111111111111111111111111111111111111111111111111")

	t.Run("legacy game uses block number and root claim", func(t *testing.T) {
		game := bindings.IDisputeGameFactoryGameSearchResult{
			RootClaim: expectedRoot,
			ExtraData: common.LeftPadBytes(
				big.NewInt(1234).Bytes(),
				32,
			),
		}
		sequence, root, ok, err := gameSequenceAndOutputRoot(game, gameTypes.CannonGameType, nil)
		require.NoError(t, err)
		require.True(t, ok)
		require.Equal(t, uint64(1234), bigs.Uint64Strict(sequence))
		require.Equal(t, expectedRoot, root)
	})

	t.Run("fast game uses block number and root claim", func(t *testing.T) {
		game := bindings.IDisputeGameFactoryGameSearchResult{
			RootClaim: expectedRoot,
			ExtraData: common.LeftPadBytes(
				big.NewInt(1234).Bytes(),
				32,
			),
		}
		sequence, root, ok, err := gameSequenceAndOutputRoot(game, gameTypes.FastGameType, nil)
		require.NoError(t, err)
		require.True(t, ok)
		require.Equal(t, uint64(1234), bigs.Uint64Strict(sequence))
		require.Equal(t, expectedRoot, root)
	})

	t.Run("super game uses timestamp and matching chain output", func(t *testing.T) {
		game := bindings.IDisputeGameFactoryGameSearchResult{
			ExtraData: superRootExtraData(5678, l2ChainID, expectedRoot),
		}
		sequence, root, ok, err := gameSequenceAndOutputRoot(game, gameTypes.SuperCannonKonaGameType, l2ChainID)
		require.NoError(t, err)
		require.True(t, ok)
		require.Equal(t, uint64(5678), bigs.Uint64Strict(sequence))
		require.Equal(t, expectedRoot, root)
	})

	t.Run("super game without matching chain is not usable", func(t *testing.T) {
		game := bindings.IDisputeGameFactoryGameSearchResult{
			ExtraData: superRootExtraData(5678, big.NewInt(11155420), expectedRoot),
		}
		sequence, root, ok, err := gameSequenceAndOutputRoot(game, gameTypes.SuperCannonKonaGameType, l2ChainID)
		require.NoError(t, err)
		require.False(t, ok)
		require.Equal(t, uint64(5678), bigs.Uint64Strict(sequence))
		require.Equal(t, common.Hash{}, root)
	})
}

func superRootExtraData(timestamp uint64, chainID *big.Int, outputRoot common.Hash) []byte {
	extra := make([]byte, 9+64)
	extra[0] = 1
	binary.BigEndian.PutUint64(extra[1:9], timestamp)
	copy(extra[9:41], common.LeftPadBytes(chainID.Bytes(), 32))
	copy(extra[41:73], outputRoot[:])
	return extra
}

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
						"00000000000000000000000000000000420000000000000000000000000000000000" +
						"00100000000000000000000000006900000000000000000000000000000000000003" +
						"00000000000000000000000000000000000000000000000000000000000000000000" +
						"00000000000000000000000000000000000000000000000000000000000000000000" +
						"000000000000000000000000000000000000000000000000000000c0000000000000" +
						"00000000000000000000000000000000000000000000000000e40166a07a00000000" +
						"000000000000000089d51be807d98fc974a0f41b2e67a8228d7846ef000000000000" +
						"0000000000007c6b91d9be155a6db01f749217d76ff02a7227f20000000000000000" +
						"00000000c20c5ec92fda6e611a08485123cdc0d5b84bd3a200000000000000000000" +
						"0000c20c5ec92fda6e611a08485123cdc0d5b84bd3a2000000000000000000000000" +
						"00000000000000000000000000000000000001f40000000000000000000000000000" +
						"0000000000000000000000000000000000c000000000000000000000000000000000" +
						"00000000000000000000000000000000000000000000000000000000000000000000" +
						"00000000000000000000",
				),
				WithdrawalHash: common.HexToHash("0x0d827f8148288e3a2466018f71b968ece4ea9f9e2a81c30da9bd46cce2868285"),
				Raw: types.Log{
					Address: common.HexToAddress("0x4200000000000000000000000000000000000016"),
					Topics: []common.Hash{
						common.HexToHash("0x02a52367d10742d8032712c1bb8e0144ff1ec5ffda1ed7d70bb05a2744955054"),
						common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000000"),
						common.HexToHash("0x0000000000000000000000004200000000000000000000000000000000000007"),
						common.HexToHash("0x0000000000000000000000006900000000000000000000000000000000000002"),
					},
					Data: hexutil.MustDecode(
						"0x00000000000000000000000000000000000000000000000000000000000000" +
							"000000000000000000000000000000000000000000000000000000000000031b80" +
							"00000000000000000000000000000000000000000000000000000000000000800d" +
							"827f8148288e3a2466018f71b968ece4ea9f9e2a81c30da9bd46cce28682850000" +
							"0000000000000000000000000000000000000000000000000000000001e4d764ad" +
							"0b0001000000000000000000000000000000000000000000000000000000000000" +
							"000000000000000000000000420000000000000000000000000000000000001000" +
							"000000000000000000000069000000000000000000000000000000000000030000" +
							"000000000000000000000000000000000000000000000000000000000000000000" +
							"000000000000000000000000000000000000000000000000000000000000000000" +
							"000000000000000000000000000000000000000000000000000000c00000000000" +
							"0000000000000000000000000000000000000000000000000000e40166a07a0000" +
							"0000000000000000000089d51be807d98fc974a0f41b2e67a8228d7846ef000000" +
							"0000000000000000007c6b91d9be155a6db01f749217d76ff02a7227f200000000" +
							"0000000000000000c20c5ec92fda6e611a08485123cdc0d5b84bd3a20000000000" +
							"00000000000000c20c5ec92fda6e611a08485123cdc0d5b84bd3a2000000000000" +
							"00000000000000000000000000000000000000000000000001f400000000000000" +
							"000000000000000000000000000000000000000000000000c00000000000000000" +
							"000000000000000000000000000000000000000000000000000000000000000000" +
							"000000000000000000000000000000000000000000000000000000000000000000" +
							"0000000000000000000000000000",
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
