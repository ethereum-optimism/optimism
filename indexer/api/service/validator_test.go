package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseValidateLimit(t *testing.T) {
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

func TestParseValidateAddress(t *testing.T) {
	v := Validator{}

	// (1) Happy case
	addr := "0x95222290DD7278Aa3Ddd389Cc1E1d165CC4BAfe5"
	_, err := v.ParseValidateAddress(addr)
	require.NoError(t, err, "address should be pass")

	// (2) Invalid hex
	addr = "🫡"
	_, err = v.ParseValidateAddress(addr)
	require.Error(t, err, "address must be represented as a valid hexadecimal string")

	// (3) Zero address
	addr = "0x0000000000000000000000000000000000000000"
	_, err = v.ParseValidateAddress(addr)
	require.Error(t, err, "address cannot be black-hole value")
}

func Test_ParseValidateCursor(t *testing.T) {
	v := Validator{}

	// (1) Happy case
	cursor := "0xf3fd2eb696dab4263550b938726f9b3606e334cce6ebe27446bc26cb700b94e0"
	err := v.ValidateCursor(cursor)
	require.NoError(t, err, "cursor should be pass")

	// (2) Invalid length
	cursor = "0x000"
	err = v.ValidateCursor(cursor)
	require.Error(t, err, "cursor must be 32 byte hex string")

	// (3) Invalid hex
	cursor = "0🫡"
	err = v.ValidateCursor(cursor)
	require.Error(t, err, "cursor must start with 0x")
}
