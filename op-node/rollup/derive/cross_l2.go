package derive

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	superchain "github.com/ethereum-optimism/optimism/op-superchain"
)

var (
	crossL2InboxAddr = common.Address{}
)

type CrossL2TxValidity uint8

const (
	CrossL2TxDrop = iota
	CrossL2TxAccept
	CrossL2TxUndecided
)

type CrossL2 struct {
	log       log.Logger
	rollupCfg *rollup.Config
	backend   superchain.Backend
}

func NewCrossL2(log log.Logger, cfg *rollup.Config, backend superchain.Backend) *CrossL2 {
	return &CrossL2{log, cfg, backend}
}

func (inbox *CrossL2) FilterAttributes(ctx context.Context, attrs *eth.PayloadAttributes) (*eth.PayloadAttributes, bool, error) {
	if !inbox.rollupCfg.IsInterop(uint64(attrs.Timestamp)) {
		return attrs, false, nil
	}

	var deposits []eth.Data
	for i, data := range attrs.Transactions {
		if data[0] == types.DepositTxType {
			// skip over deposits and track them in-case these attributes are spoiled:w
			deposits = attrs.Transactions[:i+1]
			continue
		}

		validity := inbox.checkTxBytes(ctx, data)
		if validity == CrossL2TxDrop {
			inbox.log.Info("converting payload to deposits-only due to an invalid cross-l2 tx")
			attrs.Transactions = deposits
			return attrs, true, nil
		}

		// The only way a tx can be undecided with security derived from L1 is if:
		//  - Transient RPC failures with the backend
		//  - L2 Peer information wasn't added in configuration setup
		//  - Initiating message hasn't been deemed finalized or safe relative to L2
		//
		// In these scenarios, we can choose to let the attributes pass through and progress the
		// pending safe head, relying on the engine_queue to re-org when progressing the safe head
		// but instead we'll raise a temporary error and wait since this failure mode is transient
		if validity == CrossL2TxUndecided {
			return nil, false, NewTemporaryError(fmt.Errorf(""))
		}

		// CrossL2TxAccepted
	}

	return attrs, false, nil
}

// Check for basic validity of cross-l2 executing messages. Intended only batch validation
func (inbox *CrossL2) checkTxBytes(ctx context.Context, txBytes hexutil.Bytes) CrossL2TxValidity {
	var tx types.Transaction
	if err := rlp.DecodeBytes(txBytes, &tx); err != nil {
		inbox.log.Warn("unable to decode tx bytes")
		return CrossL2TxDrop
	}

	// Skip over non-inbox transactions
	if tx.To() == nil || *tx.To() != crossL2InboxAddr {
		return CrossL2TxAccept
	}

	_, id, payload, err := superchain.ParseInboxExecuteMessageTxData(tx.Data())
	if err != nil {
		inbox.log.Warn("unable to decode inbox tx data", "tx_hash", tx.Hash())
		return CrossL2TxDrop
	}

	// Check validity with the backend
	safetyLabel, err := inbox.backend.MessageSafety(ctx, id, payload)
	if err != nil {
		inbox.log.Warn("failed to check inbox tx against message backend", "err", err)
		return CrossL2TxUndecided
	}

	if safetyLabel == superchain.MessageInvalid {
		return CrossL2TxDrop
	} else if safetyLabel == superchain.MessageUnknown {
		return CrossL2TxUndecided
	}

	return CrossL2TxAccept
}
