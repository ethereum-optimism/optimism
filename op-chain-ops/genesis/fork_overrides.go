package genesis

import (
	"strings"

	"github.com/ethereum-optimism/optimism/op-core/forks"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

// ForkOverridesAtGenesis returns deploy-config overrides that activate fork and
// every prior fork at genesis, and explicitly deactivate every later fork with
// an explicit nil offset rather than an omitted key.
//
// Explicitly disabling subsequent forks is required because of how op-deployer
// merges user-provided configs with its default configs.
func ForkOverridesAtGenesis(fork forks.Name) map[string]*hexutil.Uint64 {
	var sched UpgradeScheduleDeployConfig
	// Bedrock is the implicit genesis state and has no schedulable time offset
	// (ActivateForkAtGenesis panics on it).
	if fork != forks.Bedrock {
		sched.ActivateForkAtGenesis(fork)
	}
	forkList := sched.forks()
	out := make(map[string]*hexutil.Uint64, len(forkList))
	for _, f := range forkList {
		name := string(f.Name)
		key := "l2Genesis" + strings.ToUpper(name[:1]) + name[1:] + "TimeOffset"
		out[key] = f.L2GenesisTimeOffset
	}
	return out
}
