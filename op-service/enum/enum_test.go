package enum

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestEnumString_MultipleInputs tests the EnumString function with multiple inputs.
func TestEnumString_MultipleInputs(t *testing.T) {
	require.Equal(t, "a, b, c", EnumString([]Stringered{"a", "b", "c"}))
}

// TestEnumString_SingleString tests the EnumString function with a single input.
func TestEnumString_SingleString(t *testing.T) {
	require.Equal(t, "a", EnumString([]Stringered{"a"}))
}

// TestEnumString_EmptyString tests the EnumString function with no inputs.
func TestEnumString_EmptyString(t *testing.T) {
	require.Equal(t, "", EnumString([]Stringered{}))
}

// TestStringeredList_MultipleInputs tests the StringeredList function with multiple inputs.
func TestStringeredList_MultipleInputs(t *testing.T) {
	require.Equal(t, []Stringered{"a", "b", "c"}, StringeredList([]string{"a", "b", "c"}))
}

// TestStringeredList_SingleString tests the StringeredList function with a single input.
func TestStringeredList_SingleString(t *testing.T) {
	require.Equal(t, []Stringered{"a"}, StringeredList([]string{"a"}))
}

// TestStringeredList_EmptyString tests the StringeredList function with no inputs.
func TestStringeredList_EmptyString(t *testing.T) {
	require.Equal(t, []Stringered(nil), StringeredList([]string{}))
}
