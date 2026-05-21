package types

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRevision(t *testing.T) {
	require.True(t, RevisionAny.Any())
	// RevisionAny does not have a sort-order
	require.Equal(t, 0, RevisionAny.Cmp(0))
	require.Equal(t, 0, RevisionAny.Cmp(1))
	require.Equal(t, 0, RevisionAny.Cmp(100))
	require.Equal(t, 0, RevisionAny.Cmp(1000))

	require.Equal(t, uint64(123), Revision(123).Number())
	require.Equal(t, uint64(0), Revision(0).Number())
	require.Equal(t, 0, Revision(0).Cmp(0))
	require.Equal(t, -1, Revision(0).Cmp(1))

	require.Equal(t, 1, Revision(123).Cmp(0))
	require.Equal(t, 1, Revision(123).Cmp(122))
	require.Equal(t, 0, Revision(123).Cmp(123))
	require.Equal(t, -1, Revision(123).Cmp(124))
	require.Equal(t, -1, Revision(123).Cmp(150))

	require.Equal(t, "Rev(any)", RevisionAny.String())
	require.Equal(t, "Rev(123)", Revision(123).String())
}
