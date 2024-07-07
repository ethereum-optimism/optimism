package vm

import (
	"errors"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/utils"
)

type OpProgramServerArgs struct {
	vmConfig    Config
	localInputs *utils.LocalGameInputs
}

var _ ServerArgs = (*OpProgramServerArgs)(nil)

func NewOpProgramServerArgs(vmConfig Config, localInputs *utils.LocalGameInputs) *OpProgramServerArgs {
	return &OpProgramServerArgs{
		vmConfig:    vmConfig,
		localInputs: localInputs,
	}
}

func (s *OpProgramServerArgs) Cfg() Config {
	return s.vmConfig
}

func (s *OpProgramServerArgs) LocalInputs() *utils.LocalGameInputs {
	return s.localInputs
}

func (s *OpProgramServerArgs) SetLocalInputs(inputs utils.LocalGameInputs) {
	s.localInputs = &inputs
}

func (s *OpProgramServerArgs) FillHostCommand(args *[]string, dataDir string) error {
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
