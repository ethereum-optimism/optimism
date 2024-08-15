package vm

import (
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/utils"
)

type OpProgramServerExecutor struct {
}

var _ OracleServerExecutor = (*OpProgramServerExecutor)(nil)

func NewOpProgramServerExecutor() *OpProgramServerExecutor {
	return &OpProgramServerExecutor{}
}

func (s *OpProgramServerExecutor) OracleCommand(cfg Config, dataDir string, inputs utils.LocalGameInputs) ([]string, error) {
	args := []string{
		cfg.Server, "--server",
		"--l1", cfg.L1,
		"--l1.beacon", cfg.L1Beacon,
		"--l2", cfg.L2,
		"--datadir", dataDir,
		"--l1.head", inputs.L1Head.Hex(),
		"--l2.head", inputs.L2Head.Hex(),
		"--l2.outputroot", inputs.L2OutputRoot.Hex(),
		"--l2.claim", inputs.L2Claim.Hex(),
		"--l2.blocknumber", inputs.L2BlockNumber.Text(10),
	}
	if cfg.Network != "" {
		args = append(args, "--network", cfg.Network)
	}
	if cfg.RollupConfigPath != "" {
		args = append(args, "--rollup.config", cfg.RollupConfigPath)
	}
	if cfg.L2GenesisPath != "" {
		args = append(args, "--l2.genesis", cfg.L2GenesisPath)
	}
	return args, nil
}
