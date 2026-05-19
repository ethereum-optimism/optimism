package types

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSafetyLevel(t *testing.T) {
	for _, lvl := range []SafetyLevel{
		Finalized,
		CrossSafe,
		LocalSafe,
		CrossUnsafe,
		LocalUnsafe,
		Invalid,
	} {
		upper := strings.ToUpper(lvl.String())
		var x SafetyLevel
		require.ErrorContains(t, json.Unmarshal([]byte(fmt.Sprintf("%q", upper)), &x), "unrecognized", "case sensitive")
		require.NoError(t, json.Unmarshal([]byte(fmt.Sprintf("%q", lvl.String())), &x))
		dat, err := json.Marshal(x)
		require.NoError(t, err)
		require.Equal(t, fmt.Sprintf("%q", lvl.String()), string(dat))
	}
	var x SafetyLevel
	require.ErrorContains(t, json.Unmarshal([]byte(`""`), &x), "unrecognized", "empty")
	require.ErrorContains(t, json.Unmarshal([]byte(`"foobar"`), &x), "unrecognized", "other")
}

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
