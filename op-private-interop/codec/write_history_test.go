package codec

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/ethereum-optimism/optimism/op-private-interop/writes"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
)

func TestWriteHistoryRequiresCoverage(t *testing.T) {
	tag := common.Hash{1}
	a := &RangeClaim{FirstBlock: 11, LastBlock: 20}
	b := &RangeClaim{FirstBlock: 21, LastBlock: 30, Writes: writes.Publish(21, 30, []writes.Record{{Tag: tag, ValueCommitment: common.Hash{2}, BlockNumber: 25}})}
	require.NoError(t, CheckUnchanged(10, 30, []common.Hash{{3}}, []*RangeClaim{a, b}))
	require.ErrorIs(t, CheckUnchanged(10, 30, []common.Hash{tag}, []*RangeClaim{a, b}), ErrStateChanged)
	require.ErrorIs(t, CheckUnchanged(10, 30, nil, []*RangeClaim{b}), ErrWriteHistoryGap)
	require.ErrorIs(t, CheckUnchanged(10, 30, nil, []*RangeClaim{a}), ErrWriteHistoryGap)
	b.FirstBlock = 22
	require.ErrorIs(t, CheckUnchanged(10, 30, nil, []*RangeClaim{a, b}), ErrWriteHistoryGap)
}

func TestWriteHistoryFindsDependenciesInEachRange(t *testing.T) {
	tag := common.Hash{1}
	first := &RangeClaim{FirstBlock: 11, LastBlock: 20, Writes: writes.Publish(11, 20, []writes.Record{{Tag: tag, BlockNumber: 15}})}
	last := &RangeClaim{FirstBlock: 21, LastBlock: 30, Writes: writes.Publish(21, 30, []writes.Record{{Tag: tag, BlockNumber: 25}})}
	require.NotEqual(t, first.Writes[0].Tag, last.Writes[0].Tag)
	require.ErrorIs(t, CheckUnchanged(10, 20, []common.Hash{tag}, []*RangeClaim{first}), ErrStateChanged)
	require.ErrorIs(t, CheckUnchanged(20, 30, []common.Hash{tag}, []*RangeClaim{last}), ErrStateChanged)
	require.NoError(t, CheckUnchanged(10, 30, []common.Hash{{2}}, []*RangeClaim{first, last}))
}

func TestRetiredStableWriteClaimRejected(t *testing.T) {
	claim := &RangeClaim{FirstBlock: 11, LastBlock: 20, Writes: []writes.Record{{Tag: common.Hash{1}, BlockNumber: 15}}}
	old, err := encodeAtVersion(claim, 2)
	require.NoError(t, err)
	for _, mode := range []Mode{ModeAttested, ModeProven} {
		_, err := DecodeMode(old, mode)
		require.ErrorIs(t, err, ErrBadVersion, "stable-tag claims must not enter range-scoped freshness checking")
	}
}

func TestWriteClaimRoundTripAndBounds(t *testing.T) {
	claim := &RangeClaim{FirstBlock: 10, LastBlock: 20, Writes: []writes.Record{{Tag: common.Hash{1}, ValueCommitment: common.Hash{2}, BlockNumber: 15}}}
	data, err := Encode(claim)
	require.NoError(t, err)
	decoded, err := Decode(data)
	require.NoError(t, err)
	require.Equal(t, claim, decoded)
	claim.Writes[0].BlockNumber = 21
	_, err = Encode(claim)
	require.ErrorIs(t, err, writes.ErrInvalid)
}

func TestSharedSolidityWriteClaim(t *testing.T) {
	data, err := os.ReadFile("../writes/testdata/claim-write.json")
	require.NoError(t, err)
	var vector struct {
		Encoded   hexutil.Bytes `json:"encoded"`
		ClaimHash common.Hash   `json:"claimHash"`
	}
	require.NoError(t, json.Unmarshal(data, &vector))
	claim, err := Decode(vector.Encoded)
	require.NoError(t, err)
	require.Len(t, claim.Writes, 1)
	require.Equal(t, uint64(42), claim.Writes[0].BlockNumber)
	privateData, err := os.ReadFile("../writes/testdata/storage-write.json")
	require.NoError(t, err)
	var private struct {
		Record writes.Record `json:"record"`
	}
	require.NoError(t, json.Unmarshal(privateData, &private))
	require.Equal(t, writes.Publish(claim.FirstBlock, claim.LastBlock, []writes.Record{private.Record}), claim.Writes)
	hashType, err := abi.NewType("bytes32", "", nil)
	require.NoError(t, err)
	bytesType, err := abi.NewType("bytes", "", nil)
	require.NoError(t, err)
	encoded, err := (abi.Arguments{{Type: hashType}, {Type: bytesType}}).Pack(common.Hash{}, []byte(vector.Encoded))
	require.NoError(t, err)
	require.Equal(t, vector.ClaimHash, crypto.Keccak256Hash(encoded))
}
