package vm

import (
	"errors"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/utils"
)

type KonaServerArgs struct {
	vmConfig    Config
	localInputs *utils.LocalGameInputs
}

var _ ServerArgs = (*KonaServerArgs)(nil)

func NewKonaServerArgs(vmConfig Config, localInputs *utils.LocalGameInputs) *KonaServerArgs {
	return &KonaServerArgs{
		vmConfig:    vmConfig,
		localInputs: localInputs,
	}
}

func (s *KonaServerArgs) Cfg() Config {
	return s.vmConfig
}

func (s *KonaServerArgs) LocalInputs() *utils.LocalGameInputs {
	return s.localInputs
}

func (s *KonaServerArgs) SetLocalInputs(inputs utils.LocalGameInputs) {
	s.localInputs = &inputs
}

func (s *KonaServerArgs) FillHostCommand(args *[]string, dataDir string) error {
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
