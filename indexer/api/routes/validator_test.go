package routes

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_ParseValidateLimit(t *testing.T) {
	v := Validator{}

	// (1) Happy case
	limit := "100"
	_, err := v.ParseValidateLimit(limit)
	require.NoError(t, err, "limit should be valid")

	// (2) Boundary validation
	limit = "0"
	_, err = v.ParseValidateLimit(limit)
	require.Error(t, err, "limit must be greater than 0")

	// (3) Type validation
	limit = "abc"
	_, err = v.ParseValidateLimit(limit)
	require.Error(t, err, "limit must be an integer value")
}

func Test_ParseValidateAddress(t *testing.T) {
	v := Validator{}

	// (1) Happy case
	addr := "0x1"
	_, err := v.ParseValidateAddress(addr)
	require.NoError(t, err, "address should be pass")

	// (2) Invalid hex
	addr = "🫡"
	_, err = v.ParseValidateAddress(addr)
	require.Error(t, err, "address must be represented as a valid hexadecimal string")

	// (3) Zero address
	addr = "0x0"
	_, err = v.ParseValidateAddress(addr)
	require.Error(t, err, "address cannot be black-hole value")
}
