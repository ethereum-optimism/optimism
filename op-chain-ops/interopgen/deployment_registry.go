package interopgen

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
)

// DeploymentRegistryPrecompile is inserted as precompile,
// substituting the regular DeploymentRegistry contract,
// that the Artifacts contract uses to load/store contract deployments.
//
// The Deployments data can be overridden,
// to sub in a context for a specific L2 deployment,
// rather than monolithically assuming a single global instance of everything.
type DeploymentRegistryPrecompile struct {
	Deployments map[string]common.Address `evm:"-"`
}

// DeploymentRegistryPrecompile:
// newDeployments() not supported, only used via CLI, not by scripts themselves.
// get(string) not supported, does not seem to be used.
// loadInitializedSlot(string) not supported. Only used by a test.

func (dr *DeploymentRegistryPrecompile) Has(name string) bool {
	_, ok := dr.Deployments[name]
	return ok
}

func (dr *DeploymentRegistryPrecompile) GetAddress(name string) common.Address {
	return dr.Deployments[name] // zero if not present
}

func (dr *DeploymentRegistryPrecompile) MustGetAddress(name string) (common.Address, error) {
	addr, ok := dr.Deployments[name]
	if !ok {
		return common.Address{}, fmt.Errorf("unknown deployment %q", name)
	}
	return addr, nil
}

func (dr *DeploymentRegistryPrecompile) Save(name string, addr common.Address) {
	dr.Deployments[name] = addr
}

func (dr *DeploymentRegistryPrecompile) PrankDeployment(name string, addr common.Address) {
	dr.Deployments[name] = addr
}
