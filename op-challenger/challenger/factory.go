package challenger

import (
	"errors"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
)

var ErrMissingFactoryEvent = errors.New("missing factory event")

// BuildDisputeGameLogFilter creates a filter query for the DisputeGameFactory contract.
//
// The `DisputeGameCreated` event is encoded as:
// 0: address indexed disputeProxy,
// 1: GameType indexed gameType,
// 2: Claim indexed rootClaim,
func BuildDisputeGameLogFilter(contract *abi.ABI) (ethereum.FilterQuery, error) {
	event := contract.Events["DisputeGameCreated"]

	if event.ID == (common.Hash{}) {
		return ethereum.FilterQuery{}, ErrMissingFactoryEvent
	}

	query := ethereum.FilterQuery{
		Topics: [][]common.Hash{
			{event.ID},
		},
	}

	return query, nil
}
