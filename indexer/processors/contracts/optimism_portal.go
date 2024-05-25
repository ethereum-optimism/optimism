package contracts

import (
	"errors"
	"math/big"

	"github.com/ethereum-optimism/optimism/indexer/bigint"
	"github.com/ethereum-optimism/optimism/indexer/database"
	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

type OptimismPortalTransactionDepositEvent struct {
	Event     *database.ContractEvent
	DepositTx *types.DepositTx
	Tx        database.Transaction
	GasLimit  *big.Int
}

type OptimismPortalWithdrawalProvenEvent struct {
	*bindings.OptimismPortalWithdrawalProven
	Event *database.ContractEvent
}

type OptimismPortalWithdrawalFinalizedEvent struct {
	*bindings.OptimismPortalWithdrawalFinalized
	Event *database.ContractEvent
}

func OptimismPortalTransactionDepositEvents(contractAddress common.Address, db *database.DB, fromHeight, toHeight *big.Int) ([]OptimismPortalTransactionDepositEvent, error) {
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

		mint := depositTx.Mint
		if mint == nil {
			mint = bigint.Zero
		}

		optimismPortalTxDeposits[i] = OptimismPortalTransactionDepositEvent{
			Event:     &transactionDepositEvents[i].ContractEvent,
			DepositTx: depositTx,
			GasLimit:  new(big.Int).SetUint64(depositTx.Gas),
			Tx: database.Transaction{
				FromAddress: txDeposit.From,
				ToAddress:   txDeposit.To,
				Amount:      mint,
				Data:        depositTx.Data,
				Timestamp:   transactionDepositEvents[i].Timestamp,
			},
		}
	}

	return optimismPortalTxDeposits, nil
}

func OptimismPortalWithdrawalProvenEvents(contractAddress common.Address, db *database.DB, fromHeight, toHeight *big.Int) ([]OptimismPortalWithdrawalProvenEvent, error) {
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

func OptimismPortalWithdrawalFinalizedEvents(contractAddress common.Address, db *database.DB, fromHeight, toHeight *big.Int) ([]OptimismPortalWithdrawalFinalizedEvent, error) {
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
