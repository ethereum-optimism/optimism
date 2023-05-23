package genesis

import (
	"math/big"

	"github.com/ledgerwatch/erigon-lib/common"
	"github.com/ledgerwatch/erigon/boba-bindings/bindings"
	"github.com/ledgerwatch/erigon/params"
)

var (
	// proxies represents the set of proxies in front of contracts.
	proxies = []string{
		"SystemConfigProxy",
		"L2OutputOracleProxy",
		"L1CrossDomainMessengerProxy",
		"L1StandardBridgeProxy",
		"OptimismPortalProxy",
		"OptimismMintableERC20FactoryProxy",
	}
	// portalMeteringSlot is the storage slot containing the metering params.
	portalMeteringSlot = common.Hash{31: 0x01}
	// zeroHash represents the zero value for a hash.
	zeroHash = common.Hash{}
	// uint128Max is type(uint128).max and is set in the init function.
	uint128Max = new(big.Int)
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
