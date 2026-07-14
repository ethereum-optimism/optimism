package proofs

import (
	"context"
	"math/big"
	"testing"
	"time"

	op_e2e "github.com/ethereum-optimism/optimism/op-e2e"

	"github.com/ethereum-optimism/optimism/op-e2e/config"

	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-e2e/bindings"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/geth"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/wait"
	"github.com/ethereum-optimism/optimism/op-e2e/system/e2esys"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestL2OutputSubmitterFaultProofs(t *testing.T) {
	op_e2e.InitParallel(t)
	cfg := e2esys.DefaultSystemConfig(t, e2esys.WithAllocType(config.AllocTypeMTCannon))
	cfg.NonFinalizedProposals = true // speed up the time till we see output proposals

	sys, err := cfg.Start(t)
	require.Nil(t, err, "Error starting up system")

	l1Client := sys.NodeClient("l1")

	rollupClient := sys.RollupClient("sequencer")

	disputeGameFactory, err := bindings.NewDisputeGameFactoryCaller(cfg.L1Deployments.DisputeGameFactoryProxy, l1Client)
	require.Nil(t, err)

	initialGameCount, err := disputeGameFactory.GameCount(&bind.CallOpts{})
	require.Nil(t, err)

	l2Verif := sys.NodeClient("verifier")
	const targetBlockNumber = uint64(6)
	_, err = geth.WaitForBlock(new(big.Int).SetUint64(targetBlockNumber), l2Verif)
	require.Nil(t, err)

	timeoutCh := time.After(15 * time.Second)
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	for {
		latestGameCount, err := disputeGameFactory.GameCount(&bind.CallOpts{})
		require.Nil(t, err)

		if latestGameCount.Cmp(initialGameCount) > 0 {
			latestGames, err := disputeGameFactory.FindLatestGames(
				&bind.CallOpts{},
				uint32(gameTypes.SuperPermissionedGameType),
				new(big.Int).Sub(latestGameCount, common.Big1),
				common.Big1,
			)
			require.NoError(t, err)
			require.Len(t, latestGames, 1)
			latestGame := latestGames[0]
			superRoot, err := eth.UnmarshalSuperRoot(latestGame.ExtraData)
			require.NoError(t, err)
			superV1, ok := superRoot.(*eth.SuperV1)
			require.True(t, ok)
			gameBlockNumber, err := sys.RollupConfig.TargetBlockNumber(superV1.Timestamp)
			require.NoError(t, err)
			// A new game may still predate the target block, so keep polling until a proposal covers it.
			if gameBlockNumber >= targetBlockNumber {
				require.GreaterOrEqual(t, gameBlockNumber, targetBlockNumber)
				require.Len(t, superV1.Chains, 1)
				require.Equal(t, eth.ChainIDFromBig(sys.RollupConfig.L2ChainID), superV1.Chains[0].ChainID)
				require.Equal(t, eth.SuperRoot(superV1), eth.Bytes32(latestGame.RootClaim))

				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer cancel()
				l2Output, err := wait.ForOutputAtBlock(ctx, rollupClient, gameBlockNumber)
				require.NoError(t, err)
				require.Equal(t, l2Output.OutputRoot, superV1.Chains[0].Output)
				break
			}
		}

		select {
		case <-timeoutCh:
			t.Fatalf("no SuperPermissioned game proposed for L2 block %d or later", targetBlockNumber)
		case <-ticker.C:
		}
	}
}
