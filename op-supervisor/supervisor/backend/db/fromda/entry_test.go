package fromda

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

func FuzzRoundtripLinkEntry(f *testing.F) {
	f.Fuzz(func(t *testing.T, invalidated bool, revision uint64, aHash []byte, aNum uint64, aTimestamp uint64, bHash []byte, bNum uint64, bTimestamp uint64) {
		x := LinkEntry{
			source: types.BlockSeal{
				Hash:      common.BytesToHash(aHash),
				Number:    aNum,
				Timestamp: aTimestamp,
			},
			derived: types.BlockSeal{
				Hash:      common.BytesToHash(bHash),
				Number:    bNum,
				Timestamp: bTimestamp,
			},
			revision:    types.Revision(revision &^ (1 << 63)), // only block-number revision values that can roundtrip
			invalidated: invalidated,
		}
		entry := x.encode()
		require.True(t, SourceV0 == entry.Type() || InvalidatedFromV0 == entry.Type())
		var y LinkEntry
		err := y.decode(entry)
		require.NoError(t, err)
		require.Equal(t, x, y)
	})
}

func TestLinkEntry(t *testing.T) {
	t.Run("invalid type", func(t *testing.T) {
		var entry Entry
		entry[0] = 123
		var x LinkEntry
		require.ErrorContains(t, x.decode(entry), "unexpected")
	})
}
