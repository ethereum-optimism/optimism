package log

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-service/testutils/depguard"
)

// TestLeafPackage keeps op-service/log free of in-repo imports, so that every
// package in the monorepo can log through it without an import cycle.
func TestLeafPackage(t *testing.T) {
	depguard.RequireNoTransitiveImportUnder(t,
		"github.com/ethereum-optimism/optimism/op-service/log",
		"github.com/ethereum-optimism/optimism")
}
