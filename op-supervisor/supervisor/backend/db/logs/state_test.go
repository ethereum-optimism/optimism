package logs

import (
	"reflect"
	"testing"
	"unsafe"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

// TestLogContextCloneForWrite_Isolation asserts that mutating any field on the
// clone produced by cloneForWrite never affects the original. The test walks
// every field of logContext via reflection so adding a new field without
// updating cloneForWrite causes a deterministic failure.
func TestLogContextCloneForWrite_Isolation(t *testing.T) {
	original := logContext{
		nextEntryIndex: 42,
		blockHash:      common.HexToHash("0xdeadbeef"),
		blockNum:       7,
		timestamp:      1234,
		logsSince:      3,
		logHash:        common.HexToHash("0xfeedface"),
		execMsg: &types.ExecutingMessage{
			ChainID:   eth.ChainIDFromUInt64(99),
			BlockNum:  8,
			LogIdx:    2,
			Timestamp: 5678,
			Checksum:  types.MessageChecksum(common.HexToHash("0xcafebabe")),
		},
		need: FlagCanonicalHash,
		out:  []Entry{{0x01}, {0x02}},
	}

	// Snapshot the original via reflection.
	originalSnapshot := original
	originalExecMsgSnapshot := *original.execMsg
	originalOutSnapshot := make([]Entry, len(original.out))
	copy(originalOutSnapshot, original.out)

	clone := original.cloneForWrite()

	// Verify struct types match (so reflection walks the right shape).
	origType := reflect.TypeOf(original)
	cloneType := reflect.TypeOf(clone)
	require.Equal(t, origType, cloneType)

	origVal := reflect.ValueOf(&originalSnapshot).Elem()
	cloneVal := reflect.ValueOf(&clone).Elem()
	require.Equal(t, origVal.NumField(), cloneVal.NumField())

	// For each field, mutate the clone in a type-appropriate way and assert the
	// original is unchanged.
	for i := 0; i < cloneVal.NumField(); i++ {
		field := cloneVal.Field(i)
		name := cloneType.Field(i).Name

		// Use unsafe to get a settable handle even for unexported fields.
		settable := reflect.NewAt(field.Type(), unsafe.Pointer(field.UnsafeAddr())).Elem()

		switch name {
		case "nextEntryIndex":
			settable.SetInt(settable.Int() + 100)
		case "blockHash":
			settable.Set(reflect.ValueOf(common.HexToHash("0x1234")))
		case "blockNum":
			settable.SetUint(settable.Uint() + 100)
		case "timestamp":
			settable.SetUint(settable.Uint() + 100)
		case "logsSince":
			settable.SetUint(settable.Uint() + 100)
		case "logHash":
			settable.Set(reflect.ValueOf(common.HexToHash("0xabcd")))
		case "execMsg":
			// Mutate through the pointer the clone holds. Because cloneForWrite
			// allocates a fresh ExecutingMessage, this must not propagate.
			clone.execMsg.BlockNum = 9999
			clone.execMsg.LogIdx = 9999
			clone.execMsg.Timestamp = 9999
			clone.execMsg.Checksum = types.MessageChecksum(common.HexToHash("0xbad"))
			clone.execMsg.ChainID = eth.ChainIDFromUInt64(9999)
		case "need":
			settable.SetUint(settable.Uint() ^ 0xff)
		case "out":
			// Mutate an existing entry in the clone's slice. If the backing
			// array were shared, the original would see the change.
			if len(clone.out) > 0 {
				clone.out[0] = Entry{0xff}
			}
			// Also append, to catch the case where the backing array has
			// spare capacity shared with the original.
			clone.out = append(clone.out, Entry{0xee})
		default:
			t.Fatalf("unhandled field %q in cloneForWrite isolation test — update both cloneForWrite and this test", name)
		}
	}

	// The original struct and its referenced data must be untouched.
	require.True(t, reflect.DeepEqual(originalSnapshot, original),
		"original logContext changed: clone mutation leaked. original=%+v snapshot=%+v", original, originalSnapshot)
	require.Equal(t, originalExecMsgSnapshot, *original.execMsg,
		"original.execMsg changed: clone mutation leaked through pointer")
	require.Equal(t, originalOutSnapshot, original.out,
		"original.out changed: clone mutation leaked through slice backing array")
}
