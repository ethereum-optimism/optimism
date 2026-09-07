package writes

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/stretchr/testify/require"
)

func TestCanonicalRecordsAndLastWrite(t *testing.T) {
	a := Accumulator{}
	first := Record{Tag: common.Hash{1}, ValueCommitment: common.Hash{2}, BlockNumber: 10}
	latest := first
	latest.ValueCommitment = common.Hash{3}
	latest.BlockNumber = 11
	require.NoError(t, a.Apply([]Record{first}))
	require.NoError(t, a.Apply([]Record{latest}))
	require.Equal(t, []Record{latest}, a.Records())
	require.ErrorIs(t, a.Apply([]Record{first}), ErrInvalid)
	require.Equal(t, []Record{latest}, a.Records())
	encoded, err := Encode(a.Records())
	require.NoError(t, err)
	require.Len(t, encoded, RecordSize)
	decoded, err := Decode(encoded)
	require.NoError(t, err)
	require.Equal(t, a.Records(), decoded)
	_, err = Encode([]Record{first, first})
	require.ErrorIs(t, err, ErrInvalid)
	_, err = Decode(append(encoded, encoded...))
	require.ErrorIs(t, err, ErrInvalid)
	_, err = Decode(encoded[:len(encoded)-1])
	require.ErrorIs(t, err, ErrInvalid)
}

type unavailableRPC struct{}

func (unavailableRPC) CallContext(context.Context, any, string, ...any) error {
	return errors.New("offline")
}
func TestMissingWritesFailClosed(t *testing.T) {
	_, err := (Source{unavailableRPC{}}).FetchWrites(context.Background(), common.Hash{1})
	require.ErrorContains(t, err, "offline")
}

func TestDomainsAndStorageReset(t *testing.T) {
	addr := common.Address{1}
	slot := common.Hash{2}
	tag := Tag(902, Storage, addr, slot)
	require.NotEqual(t, tag, Tag(901, Storage, addr, slot))
	require.NotEqual(t, tag, Tag(902, StorageReset, addr, common.Hash{}))
	require.NotEqual(t, Commit(tag, common.Hash{}), Commit(Tag(901, Storage, addr, slot), common.Hash{}))
}

func TestSharedRustVector(t *testing.T) {
	data, err := os.ReadFile("testdata/storage-write.json")
	require.NoError(t, err)
	var v struct {
		Record  Record        `json:"record"`
		Encoded hexutil.Bytes `json:"encoded"`
	}
	require.NoError(t, json.Unmarshal(data, &v))
	tag := Tag(902, Storage, common.HexToAddress("0x1111111111111111111111111111111111111111"), common.HexToHash("0x05"))
	require.Equal(t, v.Record.Tag, tag)
	require.Equal(t, v.Record.ValueCommitment, Commit(tag, common.HexToHash("0x07")))
	encoded, err := Encode([]Record{v.Record})
	require.NoError(t, err)
	require.Equal(t, []byte(v.Encoded), encoded)
}

type responseRPC string

func (r responseRPC) CallContext(_ context.Context, out any, _ string, _ ...any) error {
	return json.Unmarshal([]byte(r), out)
}
func TestRPCRejectsMissingAndWrongBlock(t *testing.T) {
	for _, body := range []string{"null", `{}`, `{"writes":[]}`, `{"blockHash":"0x0000000000000000000000000000000000000000000000000000000000000000"}`} {
		_, err := (Source{responseRPC(body)}).FetchWrites(context.Background(), common.Hash{1})
		require.ErrorIs(t, err, ErrInvalid)
	}
}
