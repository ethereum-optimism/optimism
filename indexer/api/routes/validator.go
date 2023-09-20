package routes

import (
	"strconv"

	"errors"
)

// Validator ... Validates API user request parameters
type Validator struct {
}

// ValidateQueryParams ... Validates the limit and cursor query parameters
func (v *Validator) ValidateLimit(limit string) (int, error) {
	if limit == "" {
		return defaultPageLimit, nil
	}

	val, err := strconv.Atoi(limit)
	if err != nil {
		return 0, errors.New("limit must be an integer value")
	}

	if val <= 0 {
		return 0, errors.New("limit must be greater than 0")
	}

	// TODO - Add a check against a max limit value
	return val, nil
}
