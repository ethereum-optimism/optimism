package challenger

import (
	"context"
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/core/types"

	"github.com/ethereum-optimism/optimism/op-node/eth"
)

var (
	// supportedL2OutputVersion is the version of the L2 output that the challenger supports.
	supportedL2OutputVersion = eth.Bytes32{}
	// ErrInvalidBlockNumber is returned when the block number of the output does not match the expected block number.
	ErrInvalidBlockNumber = errors.New("invalid block number")
	// ErrUnsupportedL2OOVersion is returned when the output version is not supported.
	ErrUnsupportedL2OOVersion = errors.New("unsupported l2oo version")
	// ErrInvalidOutputLogTopic is returned when the output log topic is invalid.
	ErrInvalidOutputLogTopic = errors.New("invalid output log topic")
	// ErrInvalidOutputTopicLength is returned when the output log topic length is invalid.
	ErrInvalidOutputTopicLength = errors.New("invalid output log topic length")
)

// OutputProposal is a proposal for an output root
// in the L2OutputOracle for a given L2 block number.
type OutputProposal struct {
	L2BlockNumber *big.Int
	OutputRoot    eth.Bytes32
}

// ParseOutputLog parses a log from the L2OutputOracle contract.
func (c *Challenger) ParseOutputLog(log *types.Log) (*OutputProposal, error) {
	// Check the length of log topics
	if len(log.Topics) != 4 {
		return nil, ErrInvalidOutputTopicLength
	}
	// Validate the first topic is the output log topic
	if log.Topics[0] != c.l2ooABI.Events["OutputProposed"].ID {
		return nil, ErrInvalidOutputLogTopic
	}
	l2BlockNumber := new(big.Int).SetBytes(log.Topics[3][:])
	expected := log.Topics[1]
	return &OutputProposal{
		L2BlockNumber: l2BlockNumber,
		OutputRoot:    eth.Bytes32(expected),
	}, nil
}

// ValidateOutput checks that a given output is expected via a trusted rollup node rpc.
// It returns: if the output is correct, the fetched output, error
func (c *Challenger) ValidateOutput(ctx context.Context, proposal OutputProposal) (bool, eth.Bytes32, error) {
	// Fetch the output from the rollup node
	ctx, cancel := context.WithTimeout(ctx, c.networkTimeout)
	defer cancel()
	output, err := c.rollupClient.OutputAtBlock(ctx, proposal.L2BlockNumber.Uint64())
	if err != nil {
		c.log.Error("Failed to fetch output", "blockNum", proposal.L2BlockNumber, "err", err)
		return false, eth.Bytes32{}, err
	}

	// Compare the output root to the expected output root
	equalRoots, err := c.compareOutputRoots(output, proposal)
	if err != nil {
		return false, eth.Bytes32{}, err
	}

	return equalRoots, output.OutputRoot, nil
}

// compareOutputRoots compares the output root of the given block number to the expected output root.
func (c *Challenger) compareOutputRoots(received *eth.OutputResponse, expected OutputProposal) (bool, error) {
	if received.Version != supportedL2OutputVersion {
		c.log.Error("Unsupported l2 output version", "version", received.Version)
		return false, ErrUnsupportedL2OOVersion
	}
	if received.BlockRef.Number != expected.L2BlockNumber.Uint64() {
		c.log.Error("Invalid blockNumber", "expected", expected.L2BlockNumber, "actual", received.BlockRef.Number)
		return false, ErrInvalidBlockNumber
	}
	return received.OutputRoot == expected.OutputRoot, nil
}
