package resources

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSupernodeMetricsL1SourceCache(t *testing.T) {
	m := NewSupernodeMetrics()
	m.L1SourceCache.CacheGet("receipts", true)
	m.L1SourceCache.CacheGet("receipts", false)
	m.L1SourceCache.CacheAdd("receipts", 1, false)

	families, err := m.Registry().Gather()
	require.NoError(t, err)
	names := make(map[string]bool)
	for _, f := range families {
		names[f.GetName()] = true
	}
	require.True(t, names["supernode_l1_source_cache_get"])
	require.True(t, names["supernode_l1_source_cache_add"])
	require.True(t, names["supernode_l1_source_cache_size"])
}
