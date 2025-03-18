package pipeline

import (
	"context"
	"fmt"
	"time"

	"github.com/ethereum-optimism/optimism/op-chain-ops/genesis"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/state"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rpc"
)

func blockRefFromRpc(ctx context.Context, l1Client *rpc.Client, numberArg string) (*state.L1BlockRefJSON, error) {
	var l1BRJ state.L1BlockRefJSON
	if err := l1Client.CallContext(ctx, &l1BRJ, "eth_getBlockByNumber", numberArg, false); err != nil {
		return nil, fmt.Errorf("failed to get L1 block header for block: %w", err)
	}

	return &l1BRJ, nil
}

func SetStartBlockLiveStrategy(ctx context.Context, env *Env, st *state.State, chainID common.Hash) error {
	lgr := env.Logger.New("stage", "set-start-block", "strategy", "live")
	lgr.Info("setting start block", "id", chainID.Hex())

	thisChainState, err := st.Chain(chainID)
	if err != nil {
		return fmt.Errorf("failed to get chain state: %w", err)
	}

	headerBlockRef, err := blockRefFromRpc(ctx, env.L1Client.Client(), "latest")
	if err != nil {
		return fmt.Errorf("failed to get L1 block header: %w", err)
	}

	thisChainState.StartBlock = headerBlockRef

	return nil
}

func SetStartBlockGenesisStrategy(env *Env, st *state.State, chainID common.Hash) error {
	lgr := env.Logger.New("stage", "set-start-block", "strategy", "genesis")
	lgr.Info("setting start block", "id", chainID.Hex())

	thisChainState, err := st.Chain(chainID)
	if err != nil {
		return fmt.Errorf("failed to get chain state: %w", err)
	}

	deployConfig := &genesis.DeployConfig{
		DevL1DeployConfig: genesis.DevL1DeployConfig{
			L1BlockTime:             12,
			L1GenesisBlockTimestamp: hexutil.Uint64(time.Now().Unix()),
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
