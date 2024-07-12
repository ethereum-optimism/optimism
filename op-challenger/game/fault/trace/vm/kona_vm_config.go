package vm

import (
	"errors"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/utils"
)

type KonaVmConfig struct {
	Config
}

var _ VmConfig = (*KonaVmConfig)(nil)

func NewKonaVmConfig(vmConfig Config) *KonaVmConfig {
	return &KonaVmConfig{
		vmConfig,
	}
}

func (s *KonaVmConfig) Cfg() Config {
	return s.Config
}

func (s *KonaVmConfig) FillHostCommand(args []string, dataDir string, inputs utils.LocalGameInputs) ([]string, error) {
	if args == nil {
		return nil, errors.New("args is nil")
	}

	args = append(args,
		"--",
		s.Cfg().Server, "--server",
		"--l1-node-address", s.Cfg().L1,
		"--l1-beacon-address", s.Cfg().L1Beacon,
		"--l2-node-address", s.Cfg().L2,
		"--data-dir", dataDir,
		"--l1-head", inputs.L1Head.Hex(),
		"--l2-head", inputs.L2Head.Hex(),
		"--l2-output-root", inputs.L2OutputRoot.Hex(),
		"--l2-claim", inputs.L2Claim.Hex(),
		"--l2-block-number", inputs.L2BlockNumber.Text(10),
	)
	return args, nil
}
