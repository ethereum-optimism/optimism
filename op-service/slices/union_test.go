package slices

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUnion(t *testing.T) {
	sequencers := []string{"seq1", "seq2"}
	builders := []string{"builder1", "builder2", "seq1"} // note duplicate

	result := Union(sequencers, builders)
	sort.Strings(result)
	require.Equal(t, []string{"builder1", "builder2", "seq1", "seq2"}, result)
}
