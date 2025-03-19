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

type InitTrigger struct {
	Emitter    common.Address // address of the EventLogger contract
	Topics     []common.Hash
	OpaqueData []byte
}

func (v *InitTrigger) To() (*common.Address, error) {
	return &v.Emitter, nil
}

func (v *InitTrigger) Data() ([]byte, error) {
	// TODO format call
	return nil, nil
}

type ExecTrigger struct {
	Executor common.Address // address of the EventLogger contract
	Msg      suptypes.Message
}

func (v *ExecTrigger) To() (*common.Address, error) {
	return &v.Executor, nil
}

func (v *ExecTrigger) Data() ([]byte, error) {
	// TODO format call to CrossL2Inbox
	return nil, nil
}

type MultiTrigger struct {
	Calls []txintent.Call
}

func (v *MultiTrigger) Data() ([]byte, error) {
	// TODO format multi-call
	return nil, nil
}

type InteropOutput struct {
	Entries []suptypes.Message
}

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

func executeIndexed(events *plan.Lazy[*InteropOutput], index int) func(ctx context.Context) (*ExecTrigger, error) {
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
