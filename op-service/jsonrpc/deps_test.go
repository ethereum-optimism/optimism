package jsonrpc

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-service/testutils/depguard"
)

// TestLeafPackage keeps op-service/jsonrpc free of in-repo imports, so RPC clients, servers and
// metrics can share its types without an import cycle, and of go-ethereum imports, so the types
// stay independent of any RPC implementation.
func TestLeafPackage(t *testing.T) {
	const pkg = "github.com/ethereum-optimism/optimism/op-service/jsonrpc"
	depguard.RequireNoTransitiveImportUnder(t, pkg, "github.com/ethereum-optimism/optimism")
	depguard.RequireNoTransitiveImportUnder(t, pkg, "github.com/ethereum/go-ethereum")
}
