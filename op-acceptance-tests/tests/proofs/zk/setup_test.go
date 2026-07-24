package zk

import (
	"os"
	"path/filepath"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

const (
	zkChallengeDuration = 30 * time.Second
	zkProveDuration     = 30 * time.Second
	zkFinalityDelay     = 2 * time.Second

	// zkUnsafeProposalLead places an "unsafe" proposal's timestamp a year beyond the safe head so the
	// chain cannot reach it during the test, even on a CPU-starved CI runner.
	zkUnsafeProposalLead = 365 * 24 * time.Hour
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

// newSystem builds a supernode-backed interop system with the ZK dispute game installed and an
// honest op-challenger playing it, sourcing super roots from the supernode. Tests seed a game; the
// challenger acts on it. The honest proposer is disabled so tests keep manual control over game
// creation; use newProposerSystem to run the kona-sp1-proposer as well.
func newSystem(t devtest.T) *presets.SimpleInterop {
	vkey := loadSuperAggregationVKey(t)
	opts := append(zkPresetOptions(vkey), presets.WithoutHonestProposer())
	return presets.NewSimpleInterop(t, opts...)
}

// newProposerSystem builds the ZK proofs system with the kona-sp1-proposer
// running against the ZK dispute game type (and the honest challenger).
func newProposerSystem(t devtest.T) *presets.SimpleInterop {
	vkey := loadSuperAggregationVKey(t)
	return presets.NewSimpleInterop(t, zkPresetOptions(vkey)...)
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

// zkChallengerAddress derives the honest challenger's address for the given L2 chain.
func zkChallengerAddress(t devtest.T, chainID eth.ChainID) common.Address {
	keys, err := devkeys.NewMnemonicDevKeys(devkeys.TestMnemonic)
	t.Require().NoError(err)
	addr, err := keys.Address(devkeys.ChainOperatorKeys(chainID.ToBig())(devkeys.ChallengerRole))
	t.Require().NoError(err)
	return addr
}
