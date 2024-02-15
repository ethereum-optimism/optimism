package source

import (
	"context"
	"errors"
	"fmt"
	"math"
	"path/filepath"

	"github.com/ethereum-optimism/optimism/op-challenger/config"
	opnode "github.com/ethereum-optimism/optimism/op-node"
	"github.com/ethereum-optimism/optimism/op-program/client/driver"
	"github.com/ethereum-optimism/optimism/op-program/host"
	fpp_config "github.com/ethereum-optimism/optimism/op-program/host/config"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
)

const subdirName = "output_preimages"

type fppRunner struct {
	logger log.Logger
	cfg    *config.Config
	runFPP func(context.Context, log.Logger, *fpp_config.Config) error
}

func newFPPRunner(logger log.Logger, cfg *config.Config) *fppRunner {
	return &fppRunner{
		logger: logger,
		cfg:    cfg,
		runFPP: host.FaultProofProgram,
	}
}

// RunProgram runs op-program natively to determine if the output root is valid or not.
// Returns the maximum L2 block number that is supported by data on L1 and whether the output root is valid.
func (r *fppRunner) RunProgram(ctx context.Context, l1Head common.Hash, l2Start eth.BlockID, l2StartOutputRoot common.Hash, l2Claim common.Hash, l2ClaimBlockNum uint64) (uint64, bool, error) {
	logger := r.logger.New("claim", l2Claim, "claimBlock", l2ClaimBlockNum, "l1Head", l1Head, "startBlock", l2Start)
	logger.Info("Checking output root validity")
	fppConfig, err := createFPPConfig(logger, r.cfg, l1Head, l2Start.Hash, l2StartOutputRoot, l2Claim, l2ClaimBlockNum)
	if err != nil {
		return 0, false, fmt.Errorf("invalid op-program config: %w", err)
	}
	err = r.runFPP(ctx, logger, fppConfig)
	if errors.Is(err, driver.ErrClaimNotValid) {
		// Output root is invalid
		// TODO(client-pod#416): Determine the safe head derivation stopped at and return it
		return math.MaxUint64, false, nil
	} else if err != nil {
		// Failed to determine validity
		return 0, false, fmt.Errorf("failed to check claim validity: %w", err)
	}
	// Output root is valid, no need to restrict our output root range
	return math.MaxUint64, true, nil
}

func createFPPConfig(logger log.Logger, cfg *config.Config, l1Head common.Hash, l2Head common.Hash, l2OutputRoot common.Hash, l2Claim common.Hash, l2ClaimBlockNumber uint64) (*fpp_config.Config, error) {
	rollupCfg, err := opnode.NewRollupConfig(logger, cfg.CannonNetwork, cfg.CannonRollupConfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load rollup config: %w", err)
	}
	l2Genesis, isCustomChainConfig, err := fpp_config.LoadL2Genesis(cfg.CannonNetwork, cfg.CannonL2GenesisPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load l2 genesis: %w", err)
	}
	fppCfg := fpp_config.NewConfig(rollupCfg, l2Genesis, l1Head, l2Head, l2OutputRoot, l2Claim, l2ClaimBlockNumber)
	fppCfg.DataDir = filepath.Join(cfg.Datadir, subdirName)
	fppCfg.IsCustomChainConfig = isCustomChainConfig
	fppCfg.L1URL = cfg.L1EthRpc
	fppCfg.L2URL = cfg.CannonL2
	fppCfg.L1BeaconURL = cfg.L1Beacon
	return fppCfg, nil
}
