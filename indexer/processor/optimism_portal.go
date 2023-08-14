package processor

import (
	"errors"
	"math/big"

	"github.com/ethereum-optimism/optimism/indexer/database"
	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"

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

	eventName := "TransactionDeposited"
	if optimismPortalAbi.Events[eventName].ID != derive.DepositEventABIHash {
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
		err = UnpackLog(&txDeposit, log, eventName, optimismPortalAbi)
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
