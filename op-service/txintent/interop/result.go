package txintent

import (
	"context"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/plan"
	"github.com/ethereum-optimism/optimism/op-service/txintent"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"

	suptypes "github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

var _ txintent.Result = (*InteropOutput)(nil)

type InteropOutput struct {
	Entries []suptypes.Message
}

func (i *InteropOutput) Init() txintent.Result {
	return &InteropOutput{}
}

// FromReceipt creates Messages from receipt and block included, to prepare validating messages
func (i *InteropOutput) FromReceipt(ctx context.Context, rec *types.Receipt, includedIn eth.BlockRef, chainID eth.ChainID) error {
	for _, logEvent := range rec.Logs {
		payload := suptypes.LogToMessagePayload(logEvent)
		id := suptypes.Identifier{
			Origin:      logEvent.Address,
			BlockNumber: logEvent.BlockNumber,
			LogIndex:    uint32(logEvent.Index),
			Timestamp:   includedIn.Time,
			ChainID:     chainID,
		}
		payloadHash := crypto.Keccak256Hash(payload)
		i.Entries = append(i.Entries, suptypes.Message{
			Identifier:  id,
			PayloadHash: payloadHash,
		})
	}
	return nil
}

// ExecuteIndexed returns a lambda to transform InteropOutput to a new ExecTrigger
func ExecuteIndexed(events *plan.Lazy[*InteropOutput], index int) func(ctx context.Context) (*ExecTrigger, error) {
	return func(ctx context.Context) (*ExecTrigger, error) {
		if x := len(events.Value().Entries); x <= index {
			return nil, fmt.Errorf("invalid index: %d, only have %d events", index, x)
		}
		return &ExecTrigger{
			Executor: common.Address{},
			Msg:      events.Value().Entries[index],
		}, nil
	}
}

// RelayIndexed returns a lambda to transform InteropOutput to a new RelayTrigger
func RelayIndexed(executor common.Address, events *plan.Lazy[*InteropOutput], receipt *plan.Lazy[*types.Receipt], index int) func(ctx context.Context) (*RelayTrigger, error) {
	return func(ctx context.Context) (*RelayTrigger, error) {
		if x := len(events.Value().Entries); x <= index {
			return nil, fmt.Errorf("invalid entry index: %d, only have %d events", index, x)
		}
		if x := len(receipt.Value().Logs); x <= index {
			return nil, fmt.Errorf("invalid log index: %d, only have %d events", index, x)
		}
		msg := events.Value().Entries[index]
		payload := suptypes.LogToMessagePayload(receipt.Value().Logs[index])
		payloadHash := crypto.Keccak256Hash(payload)
		if msg.PayloadHash != payloadHash {
			return nil, fmt.Errorf("payload hash does not match, want %s but got %s", msg.PayloadHash.Hex(), payloadHash.Hex())
		}
		return &RelayTrigger{
			ExecTrigger: ExecTrigger{
				Executor: executor,
				Msg:      msg,
			},
			Payload: payload,
		}, nil
	}
}
