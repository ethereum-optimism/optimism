package inspect

import (
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core"

	"github.com/ethereum-optimism/optimism/op-chain-ops/genesis"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/pipeline"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/state"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/ioutil"
	"github.com/ethereum-optimism/optimism/op-service/jsonutil"
	"github.com/urfave/cli/v2"
)

// L1GenesisCLI is the CLI handler for `op-deployer inspect l1-genesis`.
func L1GenesisCLI(cliCtx *cli.Context) error {
	workdir := cliCtx.String(deployer.WorkdirFlagName)
	if workdir == "" {
		return fmt.Errorf("workdir flag is required")
	}

	outfile := cliCtx.String(OutfileFlagName)
	if outfile == "" {
		return fmt.Errorf("outfile flag is required")
	}

	globalState, err := pipeline.ReadState(workdir)
	if err != nil {
		return fmt.Errorf("failed to read state: %w", err)
	}

	l1Genesis, err := L1Genesis(globalState)
	if err != nil {
		return fmt.Errorf("failed to generate L1 genesis: %w", err)
	}

	if err := jsonutil.WriteJSON(l1Genesis, ioutil.ToStdOutOrFileOrNoop(outfile, 0o666)); err != nil {
		return fmt.Errorf("failed to write L1 genesis: %w", err)
	}

	return nil
}

// L1Genesis reconstructs the full L1 genesis from deployment state.
// It combines the L1StateDump allocs with the genesis template
// rebuilt from AppliedIntent.L1DevGenesisParams, mirroring the logic
// in SealL1DevGenesis.
func L1Genesis(globalState *state.State) (*core.Genesis, error) {
	if globalState.AppliedIntent == nil {
		return nil, fmt.Errorf("chain state is not applied - run op-deployer apply")
	}

	if globalState.L1StateDump == nil {
		return nil, fmt.Errorf("L1 state dump is missing from deployment state")
	}

	intent := globalState.AppliedIntent
	l1DevParams := intent.L1DevGenesisParams
	if l1DevParams == nil {
		l1DevParams = &state.L1DevGenesisParams{}
	}

	bp := &l1DevParams.BlockParams
	timestamp := bp.Timestamp
	if timestamp == 0 {
		timestamp = uint64(time.Now().Unix())
	}
	excessBlobGas := bp.ExcessBlobGas

	// Reconstruct the genesis template using the same logic as SealL1DevGenesis.
	genesisTemplate, err := genesis.NewL1GenesisMinimal(&genesis.DevL1DeployConfigMinimal{
		DevL1DeployConfig: genesis.DevL1DeployConfig{
			L1GenesisBlockTimestamp:     hexutil.Uint64(timestamp),
			L1GenesisBlockGasLimit:      hexutil.Uint64(bp.GasLimit),
			L1GenesisBlockExcessBlobGas: (*hexutil.Uint64)(&excessBlobGas),
		},
		L1ChainID:          eth.ChainIDFromUInt64(intent.L1ChainID),
		L1PragueTimeOffset: l1DevParams.PragueTimeOffset,
		L1OsakaTimeOffset:  l1DevParams.OsakaTimeOffset,
		L1BPO1TimeOffset:   l1DevParams.BPO1TimeOffset,
		BlobScheduleConfig: l1DevParams.BlobSchedule,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create L1 genesis template: %w", err)
	}

	genesisTemplate.Alloc = globalState.L1StateDump.Data.Accounts
	return genesisTemplate, nil
}
