package test

import (
	"testing"
)

func AssertNilError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func ExpectError(t *testing.T, err error, msg string) {
	t.Helper()
	if err == nil {
		t.Error(msg)
	}
}
