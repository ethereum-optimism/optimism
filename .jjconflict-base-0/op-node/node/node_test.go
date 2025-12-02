package node

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestUnixTimeStale(t *testing.T) {
	require.True(t, unixTimeStale(1_600_000_000, 1*time.Hour))
	require.False(t, unixTimeStale(uint64(time.Now().Unix()), 1*time.Hour))
}
