package enum

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestEnumString_MultipleInputs tests the EnumString function with multiple inputs.
func TestEnumString_MultipleInputs(t *testing.T) {
	require.Equal(t, "a, b, c", EnumString([]string{"a", "b", "c"}))
}

// TestEnumString_SingleString tests the EnumString function with a single input.
func TestEnumString_SingleString(t *testing.T) {
	require.Equal(t, "a", EnumString([]string{"a"}))
}

// TestEnumString_EmptyString tests the EnumString function with no inputs.
func TestEnumString_EmptyString(t *testing.T) {
	require.Equal(t, "", EnumString([]string{}))
}
