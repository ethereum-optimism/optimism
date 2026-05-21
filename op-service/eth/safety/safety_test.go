package safety

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSafetyLevel(t *testing.T) {
	for _, lvl := range []Level{
		Finalized,
		CrossSafe,
		LocalSafe,
		CrossUnsafe,
		LocalUnsafe,
		Invalid,
	} {
		upper := strings.ToUpper(lvl.String())
		var x Level
		require.ErrorContains(t, json.Unmarshal([]byte(fmt.Sprintf("%q", upper)), &x), "unrecognized", "case sensitive")
		require.NoError(t, json.Unmarshal([]byte(fmt.Sprintf("%q", lvl.String())), &x))
		dat, err := json.Marshal(x)
		require.NoError(t, err)
		require.Equal(t, fmt.Sprintf("%q", lvl.String()), string(dat))
	}
	var x Level
	require.ErrorContains(t, json.Unmarshal([]byte(`""`), &x), "unrecognized", "empty")
	require.ErrorContains(t, json.Unmarshal([]byte(`"foobar"`), &x), "unrecognized", "other")
}
