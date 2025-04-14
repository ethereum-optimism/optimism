package pipeline

import (
	"context"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/state"
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

	if st.L1DevGenesis == nil {
		return errors.New("must seal L1 genesis state before determining start-block")
	}
	thisChainState, err := st.Chain(chainID)
	if err != nil {
		return fmt.Errorf("failed to get chain state: %w", err)
	}

	thisChainState.StartBlock = state.BlockRefJsonFromHeader(st.L1DevGenesis.ToBlock().Header())

	return nil
}
