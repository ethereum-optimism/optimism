package processors

import (
	"errors"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"

	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/depset"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

type EventDecoderFn func(*ethTypes.Log, depset.ChainIndexFromID) (*types.ExecutingMessage, error)

func DecodeExecutingMessageLog(l *ethTypes.Log, depSet depset.ChainIndexFromID) (*types.ExecutingMessage, error) {
	if l.Address != params.InteropCrossL2InboxAddress {
		return nil, nil
	}
	if len(l.Topics) != 2 { // topics: event-id and payload-hash
		return nil, nil
	}
	if l.Topics[0] != types.ExecutingMessageEventTopic {
		return nil, nil
	}
	var msg types.Message
	if err := msg.DecodeEvent(l.Topics, l.Data); err != nil {
		return nil, fmt.Errorf("invalid executing message: %w", err)
	}
	logHash := types.PayloadHashToLogHash(msg.PayloadHash, msg.Identifier.Origin)

	var chainIndex types.ChainIndex
	index, err := depSet.ChainIndexFromID(eth.ChainID(msg.Identifier.ChainID))
	if err != nil {
		if errors.Is(err, types.ErrUnknownChain) {
			chainIndex = depset.NotFoundChainIndex
		} else {
			return nil, fmt.Errorf("failed to translate chain ID %s to chain index: %w", msg.Identifier.ChainID, err)
		}
	} else {
		chainIndex = index
	}
	return &types.ExecutingMessage{
		Chain:     chainIndex,
		BlockNum:  msg.Identifier.BlockNumber,
		LogIdx:    msg.Identifier.LogIndex,
		Timestamp: msg.Identifier.Timestamp,
		Hash:      logHash,
	}, nil
}
