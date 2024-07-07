package vm

import (
	"errors"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/utils"
)

type OpProgramVmArgs struct {
	vmConfig    Config
	localInputs *utils.LocalGameInputs
}

var _ VmArgs = (*OpProgramVmArgs)(nil)

func NewOpProgramVmArgs(vmConfig Config, localInputs *utils.LocalGameInputs) *OpProgramVmArgs {
	return &OpProgramVmArgs{
		vmConfig:    vmConfig,
		localInputs: localInputs,
	}
}

func (s *OpProgramVmArgs) Cfg() Config {
	return s.vmConfig
}

func (s *OpProgramVmArgs) LocalInputs() *utils.LocalGameInputs {
	return s.localInputs
}

func (s *OpProgramVmArgs) SetLocalInputs(inputs utils.LocalGameInputs) {
	s.localInputs = &inputs
}

func (s *OpProgramVmArgs) FillHostCommand(args *[]string, dataDir string) error {
	if args == nil {
		return errors.New("args is nil")
	}

	*args = append(*args,
		"--",
		s.Cfg().Server, "--server",
		"--l1", s.Cfg().L1,
		"--l1.beacon", s.Cfg().L1Beacon,
		"--l2", s.Cfg().L2,
		"--datadir", dataDir,
		"--l1.head", s.LocalInputs().L1Head.Hex(),
		"--l2.head", s.LocalInputs().L2Head.Hex(),
		"--l2.outputroot", s.LocalInputs().L2OutputRoot.Hex(),
		"--l2.claim", s.LocalInputs().L2Claim.Hex(),
		"--l2.blocknumber", s.LocalInputs().L2BlockNumber.Text(10),
	)
	return nil
}
