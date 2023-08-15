package processor

import (
	"errors"
	"math/big"

	"github.com/ethereum-optimism/optimism/indexer/database"
	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

type OptimismPortalTransactionDepositEvent struct {
	*bindings.OptimismPortalTransactionDeposited
	DepositTx *types.DepositTx
	Event     *database.ContractEvent
}

type OptimismPortalWithdrawalProvenEvent struct {
	*bindings.OptimismPortalWithdrawalProven
	Event *database.ContractEvent
}

type OptimismPortalWithdrawalFinalizedEvent struct {
	*bindings.OptimismPortalWithdrawalFinalized
	Event *database.ContractEvent
}

type OptimismPortalProvenWithdrawal struct {
	OutputRoot    [32]byte
	Timestamp     *big.Int
	L2OutputIndex *big.Int
}

func OptimismPortalTransactionDepositEvents(events *ProcessedContractEvents) ([]OptimismPortalTransactionDepositEvent, error) {
	optimismPortalAbi, err := bindings.OptimismPortalMetaData.GetAbi()
	if err != nil {
		return nil, err
	}

	transactionDepositedEventAbi := optimismPortalAbi.Events["TransactionDeposited"]
	if transactionDepositedEventAbi.ID != derive.DepositEventABIHash {
		return nil, errors.New("op-node deposit event abi hash & optimism portal tx deposit mismatch")
	}

	processedTxDepositedEvents := events.eventsBySignature[derive.DepositEventABIHash]
	txDeposits := make([]OptimismPortalTransactionDepositEvent, len(processedTxDepositedEvents))
	for i, txDepositEvent := range processedTxDepositedEvents {
		log := txDepositEvent.RLPLog

		depositTx, err := derive.UnmarshalDepositLogEvent(log)
		if err != nil {
			return nil, err
		}

		var txDeposit bindings.OptimismPortalTransactionDeposited
		txDeposit.Raw = *log
		err = UnpackLog(&txDeposit, log, transactionDepositedEventAbi.Name, optimismPortalAbi)
		if err != nil {
			return nil, err
		}

		txDeposits[i] = OptimismPortalTransactionDepositEvent{
			OptimismPortalTransactionDeposited: &txDeposit,
			DepositTx:                          depositTx,
			Event:                              txDepositEvent,
		}
	}

	return txDeposits, nil
}

func OptimismPortalTransactionDepositEvents2(contractAddress common.Address, db *database.DB, fromHeight, toHeight *big.Int) ([]OptimismPortalTransactionDepositEvent, error) {
	optimismPortalAbi, err := bindings.OptimismPortalMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	transactionDepositedEventAbi := optimismPortalAbi.Events["TransactionDeposited"]
	if transactionDepositedEventAbi.ID != derive.DepositEventABIHash {
		return nil, errors.New("op-node DepositEventABIHash & optimism portal TransactionDeposited ID mismatch")
	}

	contractEventFilter := database.ContractEvent{ContractAddress: contractAddress, EventSignature: transactionDepositedEventAbi.ID}
	transactionDepositEvents, err := db.ContractEvents.L1ContractEventsWithFilter(contractEventFilter, fromHeight, toHeight)
	if err != nil {
		return nil, err
	}

	optimismPortalTxDeposits := make([]OptimismPortalTransactionDepositEvent, len(transactionDepositEvents))
	for i := range transactionDepositEvents {
		depositTx, err := derive.UnmarshalDepositLogEvent(transactionDepositEvents[i].RLPLog)
		if err != nil {
			return nil, err
		}

		txDeposit := bindings.OptimismPortalTransactionDeposited{Raw: *transactionDepositEvents[i].RLPLog}
		err = UnpackLog(&txDeposit, transactionDepositEvents[i].RLPLog, transactionDepositedEventAbi.Name, optimismPortalAbi)
		if err != nil {
			return nil, err
		}

		optimismPortalTxDeposits[i] = OptimismPortalTransactionDepositEvent{
			OptimismPortalTransactionDeposited: &txDeposit,
			DepositTx:                          depositTx,
			Event:                              &transactionDepositEvents[i].ContractEvent,
		}

	}

	return optimismPortalTxDeposits, nil
}

func OptimismPortalWithdrawalProvenEvents(events *ProcessedContractEvents) ([]OptimismPortalWithdrawalProvenEvent, error) {
	optimismPortalAbi, err := bindings.OptimismPortalMetaData.GetAbi()
	if err != nil {
		return nil, err
	}

	eventName := "WithdrawalProven"
	processedWithdrawalProvenEvents := events.eventsBySignature[optimismPortalAbi.Events[eventName].ID]
	provenEvents := make([]OptimismPortalWithdrawalProvenEvent, len(processedWithdrawalProvenEvents))
	for i, provenEvent := range processedWithdrawalProvenEvents {
		log := provenEvent.RLPLog

		var withdrawalProven bindings.OptimismPortalWithdrawalProven
		withdrawalProven.Raw = *log
		err := UnpackLog(&withdrawalProven, log, eventName, optimismPortalAbi)
		if err != nil {
			return nil, err
		}

		provenEvents[i] = OptimismPortalWithdrawalProvenEvent{
			OptimismPortalWithdrawalProven: &withdrawalProven,
			Event:                          provenEvent,
		}
	}

	return provenEvents, nil
}

func OptimismPortalWithdrawalProvenEvents2(contractAddress common.Address, db *database.DB, fromHeight, toHeight *big.Int) ([]OptimismPortalWithdrawalProvenEvent, error) {
	optimismPortalAbi, err := bindings.OptimismPortalMetaData.GetAbi()
	if err != nil {
		return nil, err
	}

	withdrawalProvenEventAbi := optimismPortalAbi.Events["WithdrawalProven"]
	contractEventFilter := database.ContractEvent{ContractAddress: contractAddress, EventSignature: withdrawalProvenEventAbi.ID}
	withdrawalProvenEvents, err := db.ContractEvents.L1ContractEventsWithFilter(contractEventFilter, fromHeight, toHeight)
	if err != nil {
		return nil, err
	}

	provenWithdrawals := make([]OptimismPortalWithdrawalProvenEvent, len(withdrawalProvenEvents))
	for i := range withdrawalProvenEvents {
		withdrawalProven := bindings.OptimismPortalWithdrawalProven{Raw: *withdrawalProvenEvents[i].RLPLog}
		err := UnpackLog(&withdrawalProven, withdrawalProvenEvents[i].RLPLog, withdrawalProvenEventAbi.Name, optimismPortalAbi)
		if err != nil {
			return nil, err
		}
		provenWithdrawals[i] = OptimismPortalWithdrawalProvenEvent{
			OptimismPortalWithdrawalProven: &withdrawalProven,
			Event:                          &withdrawalProvenEvents[i].ContractEvent,
		}
	}

	return provenWithdrawals, nil
}

func OptimismPortalWithdrawalFinalizedEvents(events *ProcessedContractEvents) ([]OptimismPortalWithdrawalFinalizedEvent, error) {
	optimismPortalAbi, err := bindings.OptimismPortalMetaData.GetAbi()
	if err != nil {
		return nil, err
	}

	eventName := "WithdrawalFinalized"
	processedWithdrawalFinalizedEvents := events.eventsBySignature[optimismPortalAbi.Events[eventName].ID]
	finalizedEvents := make([]OptimismPortalWithdrawalFinalizedEvent, len(processedWithdrawalFinalizedEvents))
	for i, finalizedEvent := range processedWithdrawalFinalizedEvents {
		log := finalizedEvent.RLPLog

		var withdrawalFinalized bindings.OptimismPortalWithdrawalFinalized
		err := UnpackLog(&withdrawalFinalized, log, eventName, optimismPortalAbi)
		if err != nil {
			return nil, err
		}

		finalizedEvents[i] = OptimismPortalWithdrawalFinalizedEvent{
			OptimismPortalWithdrawalFinalized: &withdrawalFinalized,
			Event:                             finalizedEvent,
		}
	}

	return finalizedEvents, nil
}

func OptimismPortalWithdrawalFinalizedEvents2(contractAddress common.Address, db *database.DB, fromHeight, toHeight *big.Int) ([]OptimismPortalWithdrawalFinalizedEvent, error) {
	optimismPortalAbi, err := bindings.OptimismPortalMetaData.GetAbi()
	if err != nil {
		return nil, err
	}

	withdrawalFinalizedEventAbi := optimismPortalAbi.Events["WithdrawalFinalized"]
	contractEventFilter := database.ContractEvent{ContractAddress: contractAddress, EventSignature: withdrawalFinalizedEventAbi.ID}
	withdrawalFinalizedEvents, err := db.ContractEvents.L1ContractEventsWithFilter(contractEventFilter, fromHeight, toHeight)
	if err != nil {
		return nil, err
	}

	finalizedWithdrawals := make([]OptimismPortalWithdrawalFinalizedEvent, len(withdrawalFinalizedEvents))
	for i := range withdrawalFinalizedEvents {
		withdrawalFinalized := bindings.OptimismPortalWithdrawalFinalized{Raw: *withdrawalFinalizedEvents[i].RLPLog}
		err := UnpackLog(&withdrawalFinalized, withdrawalFinalizedEvents[i].RLPLog, withdrawalFinalizedEventAbi.Name, optimismPortalAbi)
		if err != nil {
			return nil, err
		}
		finalizedWithdrawals[i] = OptimismPortalWithdrawalFinalizedEvent{
			OptimismPortalWithdrawalFinalized: &withdrawalFinalized,
			Event:                             &withdrawalFinalizedEvents[i].ContractEvent,
		}
	}

	return finalizedWithdrawals, nil
}
