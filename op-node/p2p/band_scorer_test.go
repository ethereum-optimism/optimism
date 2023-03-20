package p2p

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestBandScorer_ParseDefault tests the [BandScorer.Parse] function
// on the default band scores cli flag value.
func TestBandScorer_ParseDefault(t *testing.T) {
	defaultScoringBands := "-40:graylist;-20:restricted;0:nopx;20:friend;"

	// Create a new band scorer.
	bandScorer := NewBandScorer()
	require.NoError(t, bandScorer.Parse(defaultScoringBands))

	// Validate the [BandScorer] internals.
	require.Len(t, bandScorer.(*bandScoreThresholds).bands, 4)
	require.Equal(t, bandScorer.(*bandScoreThresholds).bands["graylist"], float64(-40))
	require.Equal(t, bandScorer.(*bandScoreThresholds).bands["restricted"], float64(-20))
	require.Equal(t, bandScorer.(*bandScoreThresholds).bands["nopx"], float64(0))
	require.Equal(t, bandScorer.(*bandScoreThresholds).bands["friend"], float64(20))
	require.Equal(t, bandScorer.(*bandScoreThresholds).lowestBand, "graylist")
}

// TestBandScorer_ParseEmpty tests the [BandScorer.Parse] function
// on an empty string.
func TestBandScorer_ParseEmpty(t *testing.T) {
	// Create a band scorer on an empty string.
	bandScorer := NewBandScorer()
	require.NoError(t, bandScorer.Parse(""))

	// Validate the [BandScorer] internals.
	require.Len(t, bandScorer.(*bandScoreThresholds).bands, 0)
	require.Equal(t, bandScorer.(*bandScoreThresholds).bands["graylist"], float64(0))
	require.Equal(t, bandScorer.(*bandScoreThresholds).bands["restricted"], float64(0))
	require.Equal(t, bandScorer.(*bandScoreThresholds).bands["nopx"], float64(0))
	require.Equal(t, bandScorer.(*bandScoreThresholds).bands["friend"], float64(0))
	require.Equal(t, bandScorer.(*bandScoreThresholds).lowestBand, "")
}

// TestBandScorer_ParseWhitespace tests the [BandScorer.Parse] function
// on a variety of whitespaced strings.
func TestBandScorer_ParseWhitespace(t *testing.T) {
	// Create a band scorer on an empty string.
	bandScorer := NewBandScorer()
	require.NoError(t, bandScorer.Parse("  ;  ;  ;  "))

	// Validate the [BandScorer] internals.
	require.Len(t, bandScorer.(*bandScoreThresholds).bands, 0)
	require.Equal(t, bandScorer.(*bandScoreThresholds).bands["graylist"], float64(0))
	require.Equal(t, bandScorer.(*bandScoreThresholds).bands["restricted"], float64(0))
	require.Equal(t, bandScorer.(*bandScoreThresholds).bands["nopx"], float64(0))
	require.Equal(t, bandScorer.(*bandScoreThresholds).bands["friend"], float64(0))
	require.Equal(t, bandScorer.(*bandScoreThresholds).lowestBand, "")
}
