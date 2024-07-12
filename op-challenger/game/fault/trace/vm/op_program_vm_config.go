package vm

import (
	"errors"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/utils"
)

type OpProgramVmConfig struct {
	Config
}

var _ VmConfig = (*OpProgramVmConfig)(nil)

func NewOpProgramVmConfig(vmConfig Config) *OpProgramVmConfig {
	return &OpProgramVmConfig{
		vmConfig,
	}
}

func (s *OpProgramVmConfig) Cfg() Config {
	return s.Config
}

func (s *OpProgramVmConfig) FillHostCommand(args []string, dataDir string, inputs utils.LocalGameInputs) ([]string, error) {
	if args == nil {
		return nil, errors.New("args is nil")
	}

	args = append(args,
		"--",
		s.Cfg().Server, "--server",
		"--l1", s.Cfg().L1,
		"--l1.beacon", s.Cfg().L1Beacon,
		"--l2", s.Cfg().L2,
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
