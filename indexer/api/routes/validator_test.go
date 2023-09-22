package routes

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_ValidateLimit(t *testing.T) {
	validator := Validator{}

	// (1)
	limit := "100"
	_, err := validator.ValidateLimit(limit)
	require.NoError(t, err, "limit should be valid")

	// (2)
	limit = "0"
	_, err = validator.ValidateLimit(limit)
	require.Error(t, err, "limit must be greater than 0")

	// (3)
	limit = "abc"
	_, err = validator.ValidateLimit(limit)
	require.Error(t, err, "limit must be an integer value")
}
