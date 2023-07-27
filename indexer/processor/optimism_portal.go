package processor

import (
	"context"
	"math/big"

	"github.com/ethereum-optimism/optimism/indexer/database"
	"github.com/ethereum-optimism/optimism/op-bindings/bindings"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

type OptimismPortalWithdrawalProvenEvent struct {
	*bindings.OptimismPortalWithdrawalProven

	RawEvent *database.ContractEvent
}

type OptimismPortalProvenWithdrawal struct {
	OutputRoot    [32]byte
	Timestamp     *big.Int
	L2OutputIndex *big.Int
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
