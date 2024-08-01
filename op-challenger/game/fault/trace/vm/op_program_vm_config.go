package vm

import (
	"errors"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/utils"
)

type OpProgramVmConfig struct {
	L1               string
	L1Beacon         string
	L2               string
	Server           string // Path to the executable that provides the pre-image oracle server
	Network          string
	RollupConfigPath string
	L2GenesisPath    string
}

var _ VmConfig = (*OpProgramVmConfig)(nil)

func NewOpProgramVmConfig(cfg *Config) *OpProgramVmConfig {
	return &OpProgramVmConfig{
		L1:               cfg.L1,
		L1Beacon:         cfg.L1Beacon,
		L2:               cfg.L2,
		Server:           cfg.Server,
		Network:          cfg.Network,
		RollupConfigPath: cfg.RollupConfigPath,
		L2GenesisPath:    cfg.L2GenesisPath,
	}
}

func (s *OpProgramVmConfig) FillHostCommand(args []string, dataDir string, inputs utils.LocalGameInputs) ([]string, error) {
	if args == nil {
		return nil, errors.New("args is nil")
	}

	args = append(args,
		"--",
		s.Server, "--server",
		"--l1", s.L1,
		"--l1.beacon", s.L1Beacon,
		"--l2", s.L2,
		"--datadir", dataDir,
		"--l1.head", inputs.L1Head.Hex(),
		"--l2.head", inputs.L2Head.Hex(),
		"--l2.outputroot", inputs.L2OutputRoot.Hex(),
		"--l2.claim", inputs.L2Claim.Hex(),
		"--l2.blocknumber", inputs.L2BlockNumber.Text(10),
	)
	if s.Network != "" {
		args = append(args, "--network", s.Network)
	}
	if s.RollupConfigPath != "" {
		args = append(args, "--rollup.config", s.RollupConfigPath)
	}
	if s.L2GenesisPath != "" {
		args = append(args, "--l2.genesis", s.L2GenesisPath)
	}
	return args, nil
}
