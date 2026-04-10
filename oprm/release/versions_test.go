package release

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseVersionCanonicalizesLeadingV(t *testing.T) {
	v1, err := ParseVersion("1.2.3")
	require.NoError(t, err)
	require.Equal(t, "v1.2.3", v1.String())

	v2, err := ParseVersion("v1.2.3-rc.4")
	require.NoError(t, err)
	require.Equal(t, "v1.2.3-rc.4", v2.String())
	require.True(t, v2.IsRC())
	require.Equal(t, 4, v2.RCNumber())
}

func TestValidateManualReleaseVersionRejectsRC(t *testing.T) {
	_, err := ValidateManualReleaseVersion("v1.2.3-rc.1")
	require.Error(t, err)
}

func TestNextReleaseVersion(t *testing.T) {
	patch, err := NextReleaseVersion("v1.2.3", BumpPatch)
	require.NoError(t, err)
	require.Equal(t, "v1.2.4", patch.String())

	minor, err := NextReleaseVersion("v1.2.3", BumpMinor)
	require.NoError(t, err)
	require.Equal(t, "v1.3.0", minor.String())

	major, err := NextReleaseVersion("v1.2.3", BumpMajor)
	require.NoError(t, err)
	require.Equal(t, "v2.0.0", major.String())
}

func TestNextReleaseVersionSupportsOPGethStyleVersions(t *testing.T) {
	next, err := NextReleaseVersion("v1.101605.0", BumpPatch)
	require.NoError(t, err)
	require.Equal(t, "v1.101605.1", next.String())
}

func TestNextRCVersionStartsNewSeries(t *testing.T) {
	next, err := NextRCVersion("v1.2.4", "")
	require.NoError(t, err)
	require.Equal(t, "v1.2.4-rc.1", next.String())
}

func TestNextRCVersionIncrementsExistingSeries(t *testing.T) {
	next, err := NextRCVersion("v1.2.4", "v1.2.4-rc.2")
	require.NoError(t, err)
	require.Equal(t, "v1.2.4-rc.3", next.String())
}

func TestNextRCVersionResetsForDifferentTarget(t *testing.T) {
	next, err := NextRCVersion("v1.3.0", "v1.2.4-rc.7")
	require.NoError(t, err)
	require.Equal(t, "v1.3.0-rc.1", next.String())
}

func TestProposeNextRC(t *testing.T) {
	next, err := ProposeNextRC("v1.2.3", "v1.2.4-rc.1", BumpPatch)
	require.NoError(t, err)
	require.Equal(t, "v1.2.4-rc.2", next.String())
}

func TestProposeNextRCFromManualTarget(t *testing.T) {
	next, err := ProposeNextRCFromManualTarget("v1.9.0", "v1.8.1-rc.9")
	require.NoError(t, err)
	require.Equal(t, "v1.9.0-rc.1", next.String())
}

func TestProposeNextRCRejectsInvalidVersions(t *testing.T) {
	_, err := ProposeNextRC("not-a-version", "", BumpPatch)
	require.Error(t, err)

	_, err = NextRCVersion("v1.2.4", "v1.2.4")
	require.Error(t, err)
}
