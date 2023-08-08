package processor

import (
	"context"
	"errors"
	"math/big"

	"github.com/ethereum-optimism/optimism/indexer/database"
	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

type OptimismPortalTransactionDepositEvent struct {
	*bindings.OptimismPortalTransactionDeposited
	DepositTx *types.DepositTx
	RawEvent  *database.ContractEvent
}

type OptimismPortalWithdrawalProvenEvent struct {
	*bindings.OptimismPortalWithdrawalProven
	RawEvent *database.ContractEvent
}

type OptimismPortalWithdrawalFinalizedEvent struct {
	*bindings.OptimismPortalWithdrawalFinalized
	RawEvent *database.ContractEvent
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
		log := events.eventLog[txDepositEvent.GUID]

		depositTx, err := derive.UnmarshalDepositLogEvent(log)
		if err != nil {
			return nil, err
		}

		var txDeposit bindings.OptimismPortalTransactionDeposited
		err = UnpackLog(&txDeposit, log, eventName, optimismPortalAbi)
		if err != nil {
			return nil, err
		}

		txDeposits[i] = OptimismPortalTransactionDepositEvent{&txDeposit, depositTx, txDepositEvent}
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
		log := events.eventLog[provenEvent.GUID]

		var withdrawalProven bindings.OptimismPortalWithdrawalProven
		err := UnpackLog(&withdrawalProven, log, eventName, optimismPortalAbi)
		if err != nil {
			return nil, err
		}

		provenEvents[i] = OptimismPortalWithdrawalProvenEvent{&withdrawalProven, provenEvent}
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
		log := events.eventLog[finalizedEvent.GUID]

		var withdrawalFinalized bindings.OptimismPortalWithdrawalFinalized
		err := UnpackLog(&withdrawalFinalized, log, eventName, optimismPortalAbi)
		if err != nil {
			return nil, err
		}

		finalizedEvents[i] = OptimismPortalWithdrawalFinalizedEvent{&withdrawalFinalized, finalizedEvent}
	}

	return finalizedEvents, nil
}

func OptimismPortalQueryProvenWithdrawal(ethClient *ethclient.Client, portalAddress common.Address, withdrawalHash common.Hash) (OptimismPortalProvenWithdrawal, error) {
	var provenWithdrawal OptimismPortalProvenWithdrawal

	optimismPortalAbi, err := bindings.OptimismPortalMetaData.GetAbi()
	if err != nil {
		return provenWithdrawal, err
	}

	name := "provenWithdrawals"
	txData, err := optimismPortalAbi.Pack(name, withdrawalHash)
	if err != nil {
		return provenWithdrawal, err
	}

	callMsg := ethereum.CallMsg{To: &portalAddress, Data: txData}
	data, err := ethClient.CallContract(context.Background(), callMsg, nil)
	if err != nil {
		return provenWithdrawal, err
	}

	err = optimismPortalAbi.UnpackIntoInterface(&provenWithdrawal, name, data)
	if err != nil {
		return provenWithdrawal, err
	}

	return provenWithdrawal, nil
}
