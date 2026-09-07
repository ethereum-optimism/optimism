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
	b := &RangeClaim{FirstBlock: 21, LastBlock: 30, Writes: []writes.Record{{Tag: tag, ValueCommitment: common.Hash{2}, BlockNumber: 25}}}
	require.NoError(t, CheckUnchanged(10, 30, []common.Hash{{3}}, []*RangeClaim{a, b}))
	require.ErrorIs(t, CheckUnchanged(10, 30, []common.Hash{tag}, []*RangeClaim{a, b}), ErrStateChanged)
	require.ErrorIs(t, CheckUnchanged(10, 30, nil, []*RangeClaim{b}), ErrWriteHistoryGap)
	require.ErrorIs(t, CheckUnchanged(10, 30, nil, []*RangeClaim{a}), ErrWriteHistoryGap)
	b.FirstBlock = 22
	require.ErrorIs(t, CheckUnchanged(10, 30, nil, []*RangeClaim{a, b}), ErrWriteHistoryGap)
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
	hashType, err := abi.NewType("bytes32", "", nil)
	require.NoError(t, err)
	bytesType, err := abi.NewType("bytes", "", nil)
	require.NoError(t, err)
	encoded, err := (abi.Arguments{{Type: hashType}, {Type: bytesType}}).Pack(common.Hash{}, []byte(vector.Encoded))
	require.NoError(t, err)
	require.Equal(t, vector.ClaimHash, crypto.Keccak256Hash(encoded))
}
