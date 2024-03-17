package mon

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-service/eth"
)

type OutputRollupClient interface {
	OutputAtBlock(ctx context.Context, blockNum uint64) (*eth.OutputResponse, error)
	SafeHeadAtL1Block(ctx context.Context, blockNum uint64) (*eth.SafeHeadResponse, error)
}

type OutputMetrics interface {
	RecordOutputFetchTime(float64)
}

type outputValidator struct {
	log     log.Logger
	metrics OutputMetrics
	client  OutputRollupClient
}

func newOutputValidator(logger log.Logger, metrics OutputMetrics, client OutputRollupClient) *outputValidator {
	return &outputValidator{
		log:     logger,
		metrics: metrics,
		client:  client,
	}
}

// CheckRootAgreement validates the specified root claim against the output at the given block number.
func (o *outputValidator) CheckRootAgreement(ctx context.Context, l1HeadNum uint64, l2BlockNum uint64, rootClaim common.Hash) (bool, common.Hash, error) {
	output, err := o.client.OutputAtBlock(ctx, l2BlockNum)
	if err != nil {
		// string match as the error comes from the remote server so we can't use Errors.Is sadly.
		if strings.Contains(err.Error(), "not found") {
			// Output root doesn't exist, so we must disagree with it.
			return false, common.Hash{}, nil
		}
		return false, common.Hash{}, fmt.Errorf("failed to get output at block: %w", err)
	}
	o.metrics.RecordOutputFetchTime(float64(time.Now().Unix()))
	expected := common.Hash(output.OutputRoot)
	rootMatches := rootClaim == expected
	if !rootMatches {
		return false, expected, nil
	}

	// If the root matches, also check that l2 block is safe at the L1 head
	safeHead, err := o.client.SafeHeadAtL1Block(ctx, l1HeadNum)
	if err != nil {
		o.log.Warn("Unable to verify proposed block was safe", "l1HeadNum", l1HeadNum, "l2BlockNum", l2BlockNum, "err", err)
		// If safe head data isn't available, assume the output root was safe
		// Avoids making the dispute mon dependent on safe head db being available
		//
		return true, expected, nil
	}
	isSafe := safeHead.SafeHead.Number >= l2BlockNum
	return isSafe, expected, nil
}
