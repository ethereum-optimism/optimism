package challenger

import (
	"context"
	"errors"
	"math/big"

	eth "github.com/ethereum-optimism/optimism/op-node/eth"
)

// ValidateOutput checks that a given output is expected via a trusted rollup node rpc.
// It returns: if the output is correct, error
func (c *Challenger) ValidateOutput(ctx context.Context, l2BlockNumber *big.Int, expected eth.Bytes32) (bool, *eth.Bytes32, error) {
	ctx, cancel := context.WithTimeout(ctx, c.networkTimeout)
	defer cancel()
	output, err := c.rollupClient.OutputAtBlock(ctx, l2BlockNumber.Uint64())
	if err != nil {
		c.log.Error("failed to fetch output for l2BlockNumber %d: %w", l2BlockNumber, err)
		return true, nil, err
	}
	if output.Version != supportedL2OutputVersion {
		c.log.Error("unsupported l2 output version: %s", output.Version)
		return true, nil, errors.New("unsupported l2 output version")
	}
	// If the block numbers don't match, we should try to fetch the output again
	if output.BlockRef.Number != l2BlockNumber.Uint64() {
		c.log.Error("invalid blockNumber: next blockNumber is %v, blockNumber of block is %v", l2BlockNumber, output.BlockRef.Number)
		return true, nil, errors.New("invalid blockNumber")
	}
	if output.OutputRoot == expected {
		c.metr.RecordValidOutput(output.BlockRef)
	}
	return output.OutputRoot != expected, &output.OutputRoot, nil
}
