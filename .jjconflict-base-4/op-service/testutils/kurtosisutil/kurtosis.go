package kurtosisutil

import (
	"os"
	"testing"
)

func Test(t *testing.T) {
	if os.Getenv("ENABLE_KURTOSIS") == "" {
		t.Skip("skipping Kurtosis test")
	}
}
