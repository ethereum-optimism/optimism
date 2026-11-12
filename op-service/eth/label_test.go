package eth

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCheckConfirmationLabel(t *testing.T) {
	for _, label := range []BlockLabel{"", Unsafe, Safe, Finalized} {
		require.NoError(t, CheckConfirmationLabel(label), "label %q", label)
	}
	for _, label := range []BlockLabel{"pending", "earliest", "Safe", "0x1"} {
		require.ErrorContains(t, CheckConfirmationLabel(label), "is not one of", "label %q", label)
	}
}
