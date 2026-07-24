package zk

import (
	"os"
	"path/filepath"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

const (
	zkChallengeDuration = 30 * time.Second
	zkProveDuration     = 30 * time.Second
	zkFinalityDelay     = 2 * time.Second
)

func loadSuperAggregationVKey(t devtest.T) common.Hash {
	elfDir := os.Getenv("KONA_SP1_ELF_DIR")
	if elfDir == "" {
		t.Skip("KONA_SP1_ELF_DIR is not set; build the Kona SP1 ELF artifacts before running ZK acceptance tests")
	}

	var vkeys map[string]string
	_, err := toml.DecodeFile(filepath.Join(elfDir, "vkeys.toml"), &vkeys)
	t.Require().NoError(err, "failed to read Kona SP1 vkeys")
	raw, ok := vkeys["super-aggregation"]
	t.Require().True(ok, "vkeys.toml does not contain super-aggregation")
	vkeyBytes, err := hexutil.Decode(raw)
	t.Require().NoError(err, "invalid super-aggregation vkey")
	t.Require().Len(vkeyBytes, common.HashLength, "super-aggregation vkey must encode exactly 32 bytes")
	vkey := common.BytesToHash(vkeyBytes)
	t.Require().NotEqual(common.Hash{}, vkey, "super-aggregation vkey must not be zero")
	return vkey
}

// newSystem builds the ZK proofs system with the honest proposer disabled so
// tests keep manual control over game creation.
func newSystem(t devtest.T) (*presets.SimpleInterop, common.Hash) {
	vkey := loadSuperAggregationVKey(t)
	opts := append(zkPresetOptions(vkey), presets.WithoutHonestProposer())
	return presets.NewSimpleInterop(t, opts...), vkey
}

// newProposerSystem builds the ZK proofs system with the kona-sp1-proposer
// running against the ZK dispute game type.
func newProposerSystem(t devtest.T) (*presets.SimpleInterop, common.Hash) {
	vkey := loadSuperAggregationVKey(t)
	return presets.NewSimpleInterop(t, zkPresetOptions(vkey)...), vkey
}

func zkPresetOptions(vkey common.Hash) []presets.Option {
	zkCfg := sysgo.ZKDisputeGameConfig{
		ProgramVKey:          vkey,
		MaxChallengeDuration: zkChallengeDuration,
		MaxProveDuration:     zkProveDuration,
	}
	return []presets.Option{
		presets.WithZKDisputeGame(zkCfg),
		presets.WithTimeTravelEnabled(),
		presets.WithDisputeGameFinalityDelaySeconds(uint64(zkFinalityDelay / time.Second)),
		presets.WithDeployerOptions(sysgo.WithJovianAtGenesis),
	}
}
