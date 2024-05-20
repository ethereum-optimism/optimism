package genesis

import (
	"github.com/ledgerwatch/erigon/core/types"
)

// BuildL2DeveloperGenesis will build the developer Optimism Genesis
// Block. Suitable for devnets.
func BuildL2DeveloperGenesis(config *DeployConfig, l1StartHeader *types.Header) (*types.Genesis, error) {
	genspec, err := NewL2Genesis(config, l1StartHeader)
	if err != nil {
		return nil, err
	}

	storage, err := NewL2StorageConfig(config, l1StartHeader)
	if err != nil {
		return nil, err
	}

	immutable, err := NewL2ImmutableConfig(config, l1StartHeader)
	if err != nil {
		return nil, err
	}

	if err := SetL2Proxies(genspec); err != nil {
		return nil, err
	}

	if err := SetImplementations(genspec, storage, immutable); err != nil {
		return nil, err
	}

	if err := SetDevOnlyL2Implementations(genspec, storage, immutable); err != nil {
		return nil, err
	}

	SetBalanceToZero(genspec)

	return genspec, nil
}
