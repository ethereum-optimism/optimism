package disputegame

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/outputs"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/wait"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching/rpcblock"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/stretchr/testify/require"
)

type OutputGameHelper struct {
	SplitGameHelper
}

func NewOutputGameHelper(t *testing.T, require *require.Assertions, client *ethclient.Client, opts *bind.TransactOpts, privKey *ecdsa.PrivateKey,
	game contracts.FaultDisputeGameContract, factoryAddr common.Address, addr common.Address, correctOutputProvider *outputs.OutputTraceProvider, system DisputeSystem) *OutputGameHelper {
	return &OutputGameHelper{
		SplitGameHelper: SplitGameHelper{
			T:                     t,
			Require:               require,
			Client:                client,
			Opts:                  opts,
			PrivKey:               privKey,
			Game:                  game,
			FactoryAddr:           factoryAddr,
			Addr:                  addr,
			CorrectOutputProvider: correctOutputProvider,
			System:                system,
			DescribePosition: func(pos types.Position, splitDepth types.Depth) string {
				if pos.Depth() > splitDepth {
					return ""
				}
				blockNum, err := correctOutputProvider.ClaimedBlockNumber(pos)
				if err != nil {
					return ""
				}
				return fmt.Sprintf("Block num: %v", blockNum)
			},
			ClaimedL2SequenceNumber: correctOutputProvider.ClaimedBlockNumber,
		},
	}
}

func (g *OutputGameHelper) StartingBlockNum(ctx context.Context) uint64 {
	blockNum, _, err := g.Game.GetGameRange(ctx)
	g.Require.NoError(err, "failed to load starting block number")
	return blockNum
}

func (g *OutputGameHelper) WaitForL2BlockNumberChallenged(ctx context.Context) {
	g.T.Logf("Waiting for game %v to have L2 block number challenged", g.Addr)
	timedCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	err := wait.For(timedCtx, time.Second, func() (bool, error) {
		return g.Game.IsL2BlockNumberChallenged(ctx, rpcblock.Latest)
	})
	g.Require.NoError(err, "L2 block number was not challenged in time")
}
