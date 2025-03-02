package extract

import (
	"context"
	"errors"
	"fmt"
	"strings"

	monTypes "github.com/ethereum-optimism/optimism/op-dispute-mon/mon/types"
	"github.com/ethereum-optimism/optimism/op-service/clock"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching/rpcblock"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
)

var (
	ErrRollupRpcRequired = errors.New("rollup rpc required")
)

type OutputRollupClient interface {
	OutputAtBlock(ctx context.Context, blockNum uint64) (*eth.OutputResponse, error)
	SafeHeadAtL1Block(ctx context.Context, blockNum uint64) (*eth.SafeHeadResponse, error)
}

type OutputMetrics interface {
	RecordOutputFetchTime(float64)
}

type OutputAgreementEnricher struct {
	log     log.Logger
	metrics OutputMetrics
	client  OutputRollupClient
	clock   clock.Clock
}

func NewOutputAgreementEnricher(logger log.Logger, metrics OutputMetrics, client OutputRollupClient, cl clock.Clock) *OutputAgreementEnricher {
	return &OutputAgreementEnricher{
		log:     logger,
		metrics: metrics,
		client:  client,
		clock:   cl,
	}
}

// Enrich validates the specified root claim against the output at the given block number.
func (o *OutputAgreementEnricher) Enrich(ctx context.Context, block rpcblock.Block, caller GameCaller, game *monTypes.EnrichedGameData) error {
	if !game.UsesOutputRoots() {
		return nil
	}
	if o.client == nil {
		return fmt.Errorf("%w but required for game type %v", ErrRollupRpcRequired, game.GameType)
	}
	output, err := o.client.OutputAtBlock(ctx, game.L2BlockNumber)
	if err != nil {
		// string match as the error comes from the remote server so we can't use Errors.Is sadly.
		if strings.Contains(err.Error(), "not found") {
			// Output root doesn't exist, so we must disagree with it.
			game.AgreeWithClaim = false
			return nil
		}
		return fmt.Errorf("failed to get output at block: %w", err)
	}
	o.metrics.RecordOutputFetchTime(float64(o.clock.Now().Unix()))
	game.ExpectedRootClaim = common.Hash(output.OutputRoot)
	rootMatches := game.RootClaim == game.ExpectedRootClaim
	if !rootMatches {
		game.AgreeWithClaim = false
		return nil
	}

	// If the root matches, also check that l2 block is safe at the L1 head
	safeHead, err := o.client.SafeHeadAtL1Block(ctx, game.L1HeadNum)
	if err != nil {
		o.log.Warn("Unable to verify proposed block was safe", "l1HeadNum", game.L1HeadNum, "l2BlockNum", game.L2BlockNumber, "err", err)
		// If safe head data isn't available, assume the output root was safe
		// Avoids making the dispute mon dependent on safe head db being available
		game.AgreeWithClaim = true
		return nil
	}
	game.AgreeWithClaim = safeHead.SafeHead.Number >= game.L2BlockNumber
	return nil
}
