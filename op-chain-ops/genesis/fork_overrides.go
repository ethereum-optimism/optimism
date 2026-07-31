package genesis

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/ethereum-optimism/optimism/op-core/forks"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

const (
	forkOffsetKeyPrefix = "l2Genesis"
	forkOffsetKeySuffix = "TimeOffset"
)

var forkOffsetKeys = func() map[forks.Name]string {
	configType := reflect.TypeFor[UpgradeScheduleDeployConfig]()
	keys := make(map[forks.Name]string)
	for i := range configType.NumField() {
		jsonName := strings.Split(configType.Field(i).Tag.Get("json"), ",")[0]
		if !strings.HasPrefix(jsonName, forkOffsetKeyPrefix) || !strings.HasSuffix(jsonName, forkOffsetKeySuffix) {
			continue
		}
		name := strings.TrimSuffix(strings.TrimPrefix(jsonName, forkOffsetKeyPrefix), forkOffsetKeySuffix)
		keys[forks.Name(strings.ToLower(name))] = jsonName
	}
	return keys
}()

// ForkOffsetKey returns the deploy-config JSON key for a fork's L2 genesis
// time offset and whether the fork has a corresponding deploy-config field.
func ForkOffsetKey(fork forks.Name) (string, bool) {
	key, ok := forkOffsetKeys[fork]
	return key, ok
}

// ForkOverridesAtGenesis returns deploy-config overrides that activate fork and
// every prior fork at genesis, and explicitly deactivate every later fork with
// an explicit nil offset rather than an omitted key.
//
// Explicitly disabling subsequent forks is required because of how op-deployer
// merges user-provided configs with its default configs.
func ForkOverridesAtGenesis(fork forks.Name) (map[string]*hexutil.Uint64, error) {
	if !forks.IsValid(fork) {
		return nil, fmt.Errorf("invalid fork: %q", fork)
	}
	var sched UpgradeScheduleDeployConfig
	// Bedrock is the implicit genesis state and has no schedulable time offset
	// (ActivateForkAtGenesis panics on it).
	if fork != forks.Bedrock {
		sched.ActivateForkAtGenesis(fork)
	}
	forkList := sched.forks()
	out := make(map[string]*hexutil.Uint64, len(forkList))
	for _, f := range forkList {
		key, ok := ForkOffsetKey(forks.Name(f.Name))
		if !ok {
			return nil, fmt.Errorf("fork %q has no deploy-config time offset", f.Name)
		}
		out[key] = f.L2GenesisTimeOffset
	}
	return out, nil
}
