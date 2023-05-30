package challenger

import (
	"context"
	"errors"
	"math/big"

	"github.com/ethereum-optimism/optimism/op-node/eth"
)

var (
	// supportedL2OutputVersion is the version of the L2 output that the challenger supports.
	supportedL2OutputVersion = eth.Bytes32{}
	// ErrInvalidBlockNumber is returned when the block number of the output does not match the expected block number.
	ErrInvalidBlockNumber = errors.New("invalid block number")
	// ErrUnsupportedL2OOVersion is returned when the output version is not supported.
	ErrUnsupportedL2OOVersion = errors.New("unsupported l2oo version")
)

// ValidateOutput checks that a given output is expected via a trusted rollup node rpc.
// It returns: if the output is correct, the fetched output, error
func (c *Challenger) ValidateOutput(ctx context.Context, l2BlockNumber *big.Int, expected eth.Bytes32) (bool, *eth.Bytes32, error) {
	// Fetch the output from the rollup node
	ctx, cancel := context.WithTimeout(ctx, c.networkTimeout)
	defer cancel()
	output, err := c.rollupClient.OutputAtBlock(ctx, l2BlockNumber.Uint64())
	if err != nil {
		c.log.Error("Failed to fetch output", "blockNum", l2BlockNumber, "err", err)
		return false, nil, err
	}

	// Compare the output root to the expected output root
	equalRoots, err := c.compareOutputRoots(output, expected, l2BlockNumber)
	if err != nil {
		return false, nil, err
	}

	return equalRoots, &output.OutputRoot, nil
}

// compareOutputRoots compares the output root of the given block number to the expected output root.
func (c *Challenger) compareOutputRoots(received *eth.OutputResponse, expected eth.Bytes32, blockNumber *big.Int) (bool, error) {
	if received.Version != supportedL2OutputVersion {
		c.log.Error("Unsupported l2 output version", "version", received.Version)
		return false, ErrUnsupportedL2OOVersion
	}
	if received.BlockRef.Number != blockNumber.Uint64() {
		c.log.Error("Invalid blockNumber", "expected", blockNumber, "actual", received.BlockRef.Number)
		return false, ErrInvalidBlockNumber
	}
	return received.OutputRoot == expected, nil
}
