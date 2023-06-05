package challenger

import (
	"errors"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
)

var ErrMissingEvent = errors.New("missing event")

// BuildOutputLogFilter creates a filter query for the L2OutputOracle contract.
//
// The `OutputProposed` event is encoded as:
// 0: bytes32 indexed outputRoot,
// 1: uint256 indexed l2OutputIndex,
// 2: uint256 indexed l2BlockNumber,
// 3: uint256 l1Timestamp
func BuildOutputLogFilter(l2ooABI *abi.ABI) (ethereum.FilterQuery, error) {
	// Get the L2OutputOracle contract `OutputProposed` event
	event := l2ooABI.Events["OutputProposed"]

	// Sanity check that the `OutputProposed` event is defined
	if event.ID == (common.Hash{}) {
		return ethereum.FilterQuery{}, ErrMissingEvent
	}

	query := ethereum.FilterQuery{
		Topics: [][]common.Hash{
			{event.ID},
		},
	}

	return query, nil
}
