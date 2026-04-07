package deployer

import (
	"github.com/ethereum-optimism/optimism/op-core/devfeatures"
	"github.com/ethereum/go-ethereum/common"
)

// Development feature flag constants. These delegate to op-core/devfeatures,
// which is the canonical source of truth. The longer names with "DevFlag" suffix
// are preserved for backward compatibility.
var (
	OptimismPortalInteropDevFlag   = devfeatures.OptimismPortalInterop
	CannonKonaDevFlag              = devfeatures.CannonKona
	DeployV2DisputeGamesDevFlag    = devfeatures.DeployV2DisputeGames
	OPCMV2DevFlag                  = devfeatures.OPCMV2
	L2CMDevFlag                    = devfeatures.L2CM
	ZKDisputeGameDevFlag           = devfeatures.ZKDisputeGame
	SuperRootGamesMigrationDevFlag = devfeatures.SuperRootGamesMigration
)

// IsDevFeatureEnabled checks if a specific development feature is enabled in a feature bitmap.
func IsDevFeatureEnabled(bitmap, flag common.Hash) bool {
	return devfeatures.IsEnabled(bitmap, flag)
}

// EnableDevFeature enables a specific development feature in a feature bitmap.
func EnableDevFeature(bitmap, flag common.Hash) common.Hash {
	return devfeatures.Enable(bitmap, flag)
}
