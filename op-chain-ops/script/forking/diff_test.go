package forking

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

// TestExportDiffCopyPreservesDeletedAccount is the regression test for the AccountDiff.Copy
// nil-deref fix: DeleteAccount records a deleted account as a nil *AccountDiff in the diff map, and
// Copy() must preserve that deletion marker rather than dereferencing nil (which panicked).
func TestExportDiffCopyPreservesDeletedAccount(t *testing.T) {
	updated := common.HexToAddress("0x1111")
	deleted := common.HexToAddress("0x2222")

	nonce := uint64(7)
	ed := &ExportDiff{
		Account: map[common.Address]*AccountDiff{
			updated: {Nonce: &nonce}, // an updated account
			deleted: nil,             // a deleted account (DeleteAccount stores nil)
		},
		Code: map[common.Hash][]byte{},
	}

	var out *ExportDiff
	require.NotPanics(t, func() { out = ed.Copy() }, "Copy must not deref the nil deletion marker")

	// The deletion marker survives the copy as a nil entry.
	got, ok := out.Account[deleted]
	require.True(t, ok, "deleted account must remain present in the copy")
	require.Nil(t, got, "deleted account must stay nil")

	// The updated account is deep-copied (distinct pointer, equal value).
	gotUpd, ok := out.Account[updated]
	require.True(t, ok)
	require.NotNil(t, gotUpd)
	require.NotSame(t, ed.Account[updated], gotUpd, "updated account must be deep-copied")
	require.Equal(t, nonce, *gotUpd.Nonce)
}
