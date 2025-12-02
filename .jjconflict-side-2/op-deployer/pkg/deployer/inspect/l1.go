package inspect

import (
	"fmt"

	"github.com/ethereum-optimism/optimism/op-chain-ops/addresses"

	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/pipeline"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/state"

	"github.com/ethereum-optimism/optimism/op-service/ioutil"
	"github.com/ethereum-optimism/optimism/op-service/jsonutil"
	"github.com/ethereum/go-ethereum/common"
	"github.com/urfave/cli/v2"
)

func L1CLI(cliCtx *cli.Context) error {
	cfg, err := readConfig(cliCtx)
	if err != nil {
		return err
	}

	globalState, err := pipeline.ReadState(cfg.Workdir)
	if err != nil {
		return fmt.Errorf("failed to read intent: %w", err)
	}

	l1Contracts, err := L1(globalState, cfg.ChainID)
	if err != nil {
		return fmt.Errorf("failed to generate l1Contracts: %w", err)
	}

	if err := jsonutil.WriteJSON(l1Contracts, ioutil.ToStdOutOrFileOrNoop(cfg.Outfile, 0o666)); err != nil {
		return fmt.Errorf("failed to write L1 contract addresses: %w", err)
	}

	return nil
}

func L1(globalState *state.State, chainID common.Hash) (*addresses.L1Contracts, error) {
	chainState, err := globalState.Chain(chainID)
	if err != nil {
		return nil, fmt.Errorf("failed to get chain state for ID %s: %w", chainID.String(), err)
	}

	l1Contracts := addresses.L1Contracts{
		SuperchainContracts:      *globalState.SuperchainDeployment,
		ImplementationsContracts: *globalState.ImplementationsDeployment,
		OpChainContracts:         chainState.OpChainContracts,
	}

	return &l1Contracts, nil
}
