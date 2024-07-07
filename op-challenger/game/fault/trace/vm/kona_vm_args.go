package vm

import (
	"errors"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/utils"
)

type KonaVmArgs struct {
	vmConfig    Config
	localInputs *utils.LocalGameInputs
}

var _ VmArgs = (*KonaVmArgs)(nil)

func NewKonaVmArgs(vmConfig Config, localInputs *utils.LocalGameInputs) *KonaVmArgs {
	return &KonaVmArgs{
		vmConfig:    vmConfig,
		localInputs: localInputs,
	}
}

func (s *KonaVmArgs) Cfg() Config {
	return s.vmConfig
}

func (s *KonaVmArgs) LocalInputs() *utils.LocalGameInputs {
	return s.localInputs
}

func (s *KonaVmArgs) SetLocalInputs(inputs utils.LocalGameInputs) {
	s.localInputs = &inputs
}

func (s *KonaVmArgs) FillHostCommand(args *[]string, dataDir string) error {
	if args == nil {
		return errors.New("args is nil")
	}

	*args = append(*args,
		"--",
		s.Cfg().Server, "--server",
		"--l1-node-address", s.Cfg().L1,
		"--l1-beacon-address", s.Cfg().L1Beacon,
		"--l2-node-address", s.Cfg().L2,
		"--data-dir", dataDir,
		"--l1-head", s.LocalInputs().L1Head.Hex(),
		"--l2-head", s.LocalInputs().L2Head.Hex(),
		"--l2-output-root", s.LocalInputs().L2OutputRoot.Hex(),
		"--l2-claim", s.LocalInputs().L2Claim.Hex(),
		"--l2-block-number", s.LocalInputs().L2BlockNumber.Text(10),
	)
	return nil
}
