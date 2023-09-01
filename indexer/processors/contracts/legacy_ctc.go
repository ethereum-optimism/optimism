package contracts

import (
	"math/big"

	"github.com/ethereum-optimism/optimism/indexer/database"
	legacy_bindings "github.com/ethereum-optimism/optimism/op-bindings/legacy-bindings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

type LegacyCTCDepositEvent struct {
	Event    *database.ContractEvent
	Tx       database.Transaction
	TxHash   common.Hash
	GasLimit *big.Int
}

func LegacyCTCDepositEvents(contractAddress common.Address, db *database.DB, fromHeight, toHeight *big.Int) ([]LegacyCTCDepositEvent, error) {
	ctcAbi, err := legacy_bindings.CanonicalTransactionChainMetaData.GetAbi()
	if err != nil {
		return nil, err
	}

	transactionEnqueuedEventAbi := ctcAbi.Events["TransactionEnqueued"]
	contractEventFilter := database.ContractEvent{ContractAddress: contractAddress, EventSignature: transactionEnqueuedEventAbi.ID}
	events, err := db.ContractEvents.L1ContractEventsWithFilter(contractEventFilter, fromHeight, toHeight)
	if err != nil {
		return nil, err
	}

	ctcTxDeposits := make([]LegacyCTCDepositEvent, len(events))
	for i := range events {
		txEnqueued := legacy_bindings.CanonicalTransactionChainTransactionEnqueued{Raw: *events[i].RLPLog}
		err = UnpackLog(&txEnqueued, events[i].RLPLog, transactionEnqueuedEventAbi.Name, ctcAbi)
		if err != nil {
			return nil, err
		}

		zeroAmt := big.NewInt(0)
		ctcTxDeposits[i] = LegacyCTCDepositEvent{
			Event:    &events[i].ContractEvent,
			GasLimit: txEnqueued.GasLimit,
			TxHash:   types.NewTransaction(0, txEnqueued.Target, zeroAmt, txEnqueued.GasLimit.Uint64(), nil, txEnqueued.Data).Hash(),
			Tx: database.Transaction{
				FromAddress: txEnqueued.L1TxOrigin,
				ToAddress:   txEnqueued.Target,
				Amount:      zeroAmt,
				Data:        txEnqueued.Data,
				Timestamp:   events[i].Timestamp,
			},
		}
	}

	return ctcTxDeposits, nil
}
