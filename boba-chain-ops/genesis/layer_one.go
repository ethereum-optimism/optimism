package genesis

import (
	"github.com/bobanetwork/boba/boba-bindings/bindings"
	"github.com/ledgerwatch/erigon/params"
)

var (
	// The default values for the ResourceConfig, used as part of
	// an EIP-1559 curve for deposit gas.
	defaultResourceConfig = bindings.ResourceMeteringResourceConfig{
		MaxResourceLimit:            20_000_000,
		ElasticityMultiplier:        10,
		BaseFeeMaxChangeDenominator: 8,
		MinimumBaseFee:              params.GWei,
		SystemTxMaxGas:              1_000_000,
	}
)
