package vm

import (
	"errors"
	"strconv"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/utils"
	"github.com/ethereum-optimism/optimism/op-node/chaincfg"
)

type KonaVmConfig struct {
	L1       string
	L1Beacon string
	L2       string
	Server   string // Path to the executable that provides the pre-image oracle server
	Network  string
}

var _ VmConfig = (*KonaVmConfig)(nil)

func NewKonaVmConfig(cfg *Config) *KonaVmConfig {
	return &KonaVmConfig{
		L1:       cfg.L1,
		L1Beacon: cfg.L1Beacon,
		L2:       cfg.L2,
		Server:   cfg.Server,
		Network:  cfg.Network,
	}
}

func (s *KonaVmConfig) FillHostCommand(args []string, dataDir string, inputs utils.LocalGameInputs) ([]string, error) {
	if args == nil {
		return nil, errors.New("args is nil")
	}
	if s.Network == "" {
		return nil, errors.New("Network is not defined")
	}

	chainCfg := chaincfg.ChainByName(s.Network)
	args = append(args,
		s.Server, "--server",
		"--l1-node-address", s.L1,
		"--l1-beacon-address", s.L1Beacon,
		"--l2-node-address", s.L2,
		"--data-dir", dataDir,
		"--l2-chain-id", strconv.FormatUint(chainCfg.ChainID, 10),
		"--l1-head", inputs.L1Head.Hex(),
		"--l2-head", inputs.L2Head.Hex(),
		"--l2-output-root", inputs.L2OutputRoot.Hex(),
		"--l2-claim", inputs.L2Claim.Hex(),
		"--l2-block-number", inputs.L2BlockNumber.Text(10),
	)
	return args, nil
}
