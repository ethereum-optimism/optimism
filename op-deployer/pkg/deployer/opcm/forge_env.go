package opcm

import (
	"context"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/forge"
)

// ForgeEnv contains the forge-related configuration needed to run forge scripts
type ForgeEnv struct {
	Client     *forge.Client
	Context    context.Context
	L1RPCUrl   string
	PrivateKey string
}

// buildForgeOpts constructs the standard forge options for deployments
func (e *ForgeEnv) buildForgeOpts() []string {
	return []string{
		"--rpc-url", e.L1RPCUrl,
		"--broadcast",
		"--private-key", e.PrivateKey,
	}
}

// buildForgeOptsReadOnly constructs forge options for read-only operations (no broadcast)
func (e *ForgeEnv) buildForgeOptsReadOnly() []string {
	return []string{
		"--rpc-url", e.L1RPCUrl,
	}
}

// validate checks that all required fields are set
func (e *ForgeEnv) validate(requirePrivateKey bool) error {
	if e.Client == nil {
		return fmt.Errorf("Forge client is nil")
	}
	if e.Context == nil {
		e.Context = context.Background()
	}
	if e.L1RPCUrl == "" {
		return fmt.Errorf("L1 RPC URL is required for Forge deployments")
	}
	if requirePrivateKey && e.PrivateKey == "" {
		return fmt.Errorf("private key is required for Forge deployments")
	}
	return nil
}
