package db

import "github.com/google/uuid"

// NewGUID returns a new guid.
func NewGUID() string {
	return uuid.New().String()
}
