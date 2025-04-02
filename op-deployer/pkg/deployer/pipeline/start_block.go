package pipeline

import (
	"context"
	"fmt"
	"time"

	"github.com/ethereum-optimism/optimism/op-chain-ops/genesis"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/state"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

func SetStartBlockLiveStrategy(ctx context.Context, intent *state.Intent, env *Env, st *state.State, chainID common.Hash) error {
	lgr := env.Logger.New("stage", "set-start-block", "strategy", "live")
	lgr.Info("setting start block", "id", chainID.Hex())

	thisIntent, err := intent.Chain(chainID)
	if err != nil {
		return fmt.Errorf("failed to get chain intent: %w", err)
	}

	thisChainState, err := st.Chain(chainID)
	if err != nil {
		return fmt.Errorf("failed to get chain state: %w", err)
	}

	l1Client := env.L1Client.Client()

	var headerBlockRef *state.L1BlockRefJSON
	if thisIntent.L1StartBlockHash != nil {
		var l1BRJ state.L1BlockRefJSON
		if err := l1Client.CallContext(ctx, &l1BRJ, "eth_getBlockByHash", thisIntent.L1StartBlockHash.Hex(), false); err != nil {
			return fmt.Errorf("failed to get L1 block header for block: %w", err)
		}
		headerBlockRef = &l1BRJ
	} else {
		var l1BRJ state.L1BlockRefJSON
		if err := l1Client.CallContext(ctx, &l1BRJ, "eth_getBlockByNumber", "latest", false); err != nil {
			return fmt.Errorf("failed to get L1 block header for block: %w", err)
		}
		headerBlockRef = &l1BRJ
	}
	thisChainState.StartBlock = headerBlockRef

	return nil
}

func SetStartBlockGenesisStrategy(env *Env, intent *state.Intent, st *state.State, chainID common.Hash) error {
	lgr := env.Logger.New("stage", "set-start-block", "strategy", "genesis")
	lgr.Info("setting start block", "id", chainID.Hex())

	thisChainState, err := st.Chain(chainID)
	if err != nil {
		return fmt.Errorf("failed to get chain state: %w", err)
	}

	var timestamp uint64
	if intent.L1StartTimestamp != nil {
		timestamp = *intent.L1StartTimestamp
	} else {
		timestamp = uint64(time.Now().Unix())
	}

	deployConfig := &genesis.DeployConfig{
		DevL1DeployConfig: genesis.DevL1DeployConfig{
			L1BlockTime:             12,
			L1GenesisBlockTimestamp: hexutil.Uint64(timestamp),
		},
		L2InitializationConfig: genesis.L2InitializationConfig{
			L2CoreDeployConfig: genesis.L2CoreDeployConfig{
				L1ChainID: 900,
			},
			DevDeployConfig: genesis.DevDeployConfig{
				FundDevAccounts: true,
			},
		},
	}

	devGenesis, err := genesis.BuildL1DeveloperGenesis(deployConfig, st.L1StateDump.Data, &genesis.L1Deployments{})
	if err != nil {
		return fmt.Errorf("failed to build L1 developer genesis: %w", err)
	}
	thisChainState.StartBlock = state.BlockRefJsonFromHeader(devGenesis.ToBlock().Header())

	return nil
}
