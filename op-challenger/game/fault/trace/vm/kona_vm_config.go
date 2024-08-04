package vm

import (
	"errors"
	"strconv"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/utils"
	"github.com/ethereum-optimism/optimism/op-node/chaincfg"
)

type KonaVmConfig struct {
}

var _ OracleServerExecutor = (*KonaVmConfig)(nil)

func NewKonaVmConfig() *KonaVmConfig {
	return &KonaVmConfig{}
}

func (s *KonaVmConfig) OracleCommand(cfg Config, dataDir string, inputs utils.LocalGameInputs) ([]string, error) {
	if cfg.Network == "" {
		return nil, errors.New("network is not defined")
	}

	chainCfg := chaincfg.ChainByName(cfg.Network)
	return []string{
		cfg.Server, "--server",
		"--l1-node-address", cfg.L1,
		"--l1-beacon-address", cfg.L1Beacon,
		"--l2-node-address", cfg.L2,
		"--data-dir", dataDir,
		"--l2-chain-id", strconv.FormatUint(chainCfg.ChainID, 10),
		"--l1-head", inputs.L1Head.Hex(),
		"--l2-head", inputs.L2Head.Hex(),
		"--l2-output-root", inputs.L2OutputRoot.Hex(),
		"--l2-claim", inputs.L2Claim.Hex(),
		"--l2-block-number", inputs.L2BlockNumber.Text(10),
	}, nil
}
