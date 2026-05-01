package devfeatures

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

var (
	FEATURE_A            = common.HexToHash("0000000000000000000000000000000000000000000000000000000000000001")
	FEATURE_B            = common.HexToHash("0000000000000000000000000000000000000000000000000000000000000100")
	FEATURE_C            = common.HexToHash("1000000000000000000000000000000000000000000000000000000000000000")
	FEATURES_AB          = or(FEATURE_A, FEATURE_B)
	FEATURES_ABC         = or(FEATURE_A, FEATURE_B, FEATURE_C)
	FEATURES_AB_INVERTED = not(FEATURES_AB)
	EMPTY_FEATURES       = [32]byte{}
	ALL_FEATURES         = common.HexToHash("1111111111111111111111111111111111111111111111111111111111111111")
)

func TestIsDevFeatureEnabled(t *testing.T) {
	t.Run("single feature exact match", func(t *testing.T) {
		require.True(t, IsDevFeatureEnabled(FEATURE_A, FEATURE_A))
		require.True(t, IsDevFeatureEnabled(FEATURE_B, FEATURE_B))
	})

	t.Run("single feature against superset", func(t *testing.T) {
		require.True(t, IsDevFeatureEnabled(FEATURES_AB, FEATURE_A))
		require.True(t, IsDevFeatureEnabled(FEATURES_AB, FEATURE_B))
		require.True(t, IsDevFeatureEnabled(FEATURES_ABC, FEATURE_A))
	})

	t.Run("single feature against all", func(t *testing.T) {
		require.True(t, IsDevFeatureEnabled(ALL_FEATURES, FEATURE_A))
		require.True(t, IsDevFeatureEnabled(ALL_FEATURES, FEATURE_B))
	})

	t.Run("single feature against mismatched bitmap", func(t *testing.T) {
		require.False(t, IsDevFeatureEnabled(FEATURE_B, FEATURE_A))
		require.False(t, IsDevFeatureEnabled(FEATURE_A, FEATURE_B))
		require.False(t, IsDevFeatureEnabled(FEATURES_AB_INVERTED, FEATURE_A))
		require.False(t, IsDevFeatureEnabled(FEATURES_AB_INVERTED, FEATURE_B))
	})

	t.Run("single feature against empty", func(t *testing.T) {
		require.False(t, IsDevFeatureEnabled(EMPTY_FEATURES, FEATURE_A))
		require.False(t, IsDevFeatureEnabled(EMPTY_FEATURES, FEATURE_B))
	})

	t.Run("combined features exact match", func(t *testing.T) {
		require.True(t, IsDevFeatureEnabled(FEATURES_AB, FEATURES_AB))
	})

	t.Run("combined features against superset", func(t *testing.T) {
		require.True(t, IsDevFeatureEnabled(ALL_FEATURES, FEATURES_AB))
		require.True(t, IsDevFeatureEnabled(FEATURES_ABC, FEATURES_AB))
	})

	t.Run("combined features against subset", func(t *testing.T) {
		require.False(t, IsDevFeatureEnabled(FEATURE_A, FEATURES_AB))
		require.False(t, IsDevFeatureEnabled(FEATURE_B, FEATURES_AB))
	})

	t.Run("combined features against mismatched bitmap", func(t *testing.T) {
		require.False(t, IsDevFeatureEnabled(FEATURES_AB_INVERTED, FEATURES_AB))
		require.False(t, IsDevFeatureEnabled(EMPTY_FEATURES, FEATURES_AB))
		require.False(t, IsDevFeatureEnabled(FEATURE_C, FEATURES_AB))
	})

	t.Run("empty vs empty", func(t *testing.T) {
		require.False(t, IsDevFeatureEnabled(EMPTY_FEATURES, EMPTY_FEATURES))
	})

	t.Run("all vs all", func(t *testing.T) {
		require.True(t, IsDevFeatureEnabled(ALL_FEATURES, ALL_FEATURES))
	})

	t.Run("empty against all", func(t *testing.T) {
		require.False(t, IsDevFeatureEnabled(ALL_FEATURES, EMPTY_FEATURES))
	})

	t.Run("all against empty", func(t *testing.T) {
		// Strip L2CM and CannonKona because they are hardcoded enabled regardless of bitmap.
		// TODO(#20084): remove with the broader L2CMFlag/CannonKonaFlag cleanup.
		require.False(t, IsDevFeatureEnabled(EMPTY_FEATURES, and(ALL_FEATURES, not(or(L2CMFlag, CannonKonaFlag)))))
	})

	// L2CM is hardcoded enabled. TODO(#20084): remove with the broader L2CMFlag cleanup.
	t.Run("L2CM always enabled regardless of bitmap", func(t *testing.T) {
		require.True(t, IsDevFeatureEnabled(EMPTY_FEATURES, L2CMFlag))
		require.True(t, IsDevFeatureEnabled(FEATURE_A, L2CMFlag))
		require.True(t, IsDevFeatureEnabled(L2CMFlag, L2CMFlag))
	})

	// L2CM is hardcoded enabled. TODO(#20084): remove with the broader L2CMFlag cleanup.
	t.Run("L2CM always enabled when combined with other flags", func(t *testing.T) {
		require.True(t, IsDevFeatureEnabled(EMPTY_FEATURES, or(FEATURE_A, L2CMFlag)))
		require.True(t, IsDevFeatureEnabled(EMPTY_FEATURES, or(FEATURE_B, L2CMFlag)))
	})

	// CannonKona is hardcoded enabled. TODO(#20084): remove with the broader CannonKonaFlag cleanup.
	t.Run("CannonKona always enabled regardless of bitmap", func(t *testing.T) {
		require.True(t, IsDevFeatureEnabled(EMPTY_FEATURES, CannonKonaFlag))
		require.True(t, IsDevFeatureEnabled(FEATURE_A, CannonKonaFlag))
		require.True(t, IsDevFeatureEnabled(CannonKonaFlag, CannonKonaFlag))
	})

	// CannonKona is hardcoded enabled. TODO(#20084): remove with the broader CannonKonaFlag cleanup.
	t.Run("CannonKona always enabled when combined with other flags", func(t *testing.T) {
		require.True(t, IsDevFeatureEnabled(EMPTY_FEATURES, or(FEATURE_A, CannonKonaFlag)))
		require.True(t, IsDevFeatureEnabled(EMPTY_FEATURES, or(FEATURE_B, CannonKonaFlag)))
	})
}

func TestEnableDevFeature(t *testing.T) {
	result := EnableDevFeature(EMPTY_FEATURES, FEATURE_A)
	require.Equal(t, FEATURE_A, result)
}

func or(values ...[32]byte) [32]byte {
	var out [32]byte
	for i := 0; i < 32; i++ {
		for _, v := range values {
			out[i] |= v[i]
		}
	}
	return out
}

func not(a [32]byte) [32]byte {
	var out [32]byte
	for i := 0; i < 32; i++ {
		out[i] = ^a[i]
	}
	return out
}

func and(a, b [32]byte) [32]byte {
	var out [32]byte
	for i := 0; i < 32; i++ {
		out[i] = a[i] & b[i]
	}
	return out
}
