package database

import (
	"errors"
	"fmt"
	"math/big"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
)

/**
 * Types
 */

type Transaction struct {
	FromAddress common.Address `gorm:"serializer:bytes"`
	ToAddress   common.Address `gorm:"serializer:bytes"`
	Amount      *big.Int       `gorm:"serializer:u256"`
	Data        Bytes          `gorm:"serializer:bytes"`
	Timestamp   uint64
}

type L1TransactionDeposit struct {
	SourceHash           common.Hash `gorm:"serializer:bytes;primaryKey"`
	L2TransactionHash    common.Hash `gorm:"serializer:bytes"`
	InitiatedL1EventGUID uuid.UUID

	Tx       Transaction `gorm:"embedded"`
	GasLimit *big.Int    `gorm:"serializer:u256"`
}

type L2TransactionWithdrawal struct {
	WithdrawalHash       common.Hash `gorm:"serializer:bytes;primaryKey"`
	Nonce                *big.Int    `gorm:"serializer:u256"`
	InitiatedL2EventGUID uuid.UUID

	ProvenL1EventGUID    *uuid.UUID
	FinalizedL1EventGUID *uuid.UUID
	Succeeded            *bool

	Tx       Transaction `gorm:"embedded"`
	GasLimit *big.Int    `gorm:"serializer:u256"`
}

type BridgeTransactionsView interface {
	L1TransactionDeposit(common.Hash) (*L1TransactionDeposit, error)
	L1LatestBlockHeader() (*L1BlockHeader, error)

	L2TransactionWithdrawal(common.Hash) (*L2TransactionWithdrawal, error)
	L2LatestBlockHeader() (*L2BlockHeader, error)
}

type BridgeTransactionsDB interface {
	BridgeTransactionsView

	StoreL1TransactionDeposits([]L1TransactionDeposit) error

	StoreL2TransactionWithdrawals([]L2TransactionWithdrawal) error
	MarkL2TransactionWithdrawalProvenEvent(common.Hash, uuid.UUID) error
	MarkL2TransactionWithdrawalFinalizedEvent(common.Hash, uuid.UUID, bool) error
}

/**
 * Implementation
 */

type bridgeTransactionsDB struct {
	log  log.Logger
	gorm *gorm.DB
}

func newBridgeTransactionsDB(log log.Logger, db *gorm.DB) BridgeTransactionsDB {
	return &bridgeTransactionsDB{log: log.New("table", "bridge_transactions"), gorm: db}
}

/**
 * Transactions deposited from L1
 */

func (db *bridgeTransactionsDB) StoreL1TransactionDeposits(deposits []L1TransactionDeposit) error {
	deduped := db.gorm.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "source_hash"}}, DoNothing: true})
	result := deduped.Create(&deposits)
	if result.Error == nil && int(result.RowsAffected) < len(deposits) {
		db.log.Warn("ignored L1 tx deposit duplicates", "duplicates", len(deposits)-int(result.RowsAffected))
	}

	return result.Error
}

func (db *bridgeTransactionsDB) L1TransactionDeposit(sourceHash common.Hash) (*L1TransactionDeposit, error) {
	var deposit L1TransactionDeposit
	result := db.gorm.Where(&L1TransactionDeposit{SourceHash: sourceHash}).Take(&deposit)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, result.Error
	}

	return &deposit, nil
}

func (db *bridgeTransactionsDB) L1LatestBlockHeader() (*L1BlockHeader, error) {
	// Markers for an indexed bridge event
	// L1: Latest Transaction Deposit, Latest Proven/Finalized Withdrawal
	l1DepositQuery := db.gorm.Table("l1_transaction_deposits").Order("l1_transaction_deposits.timestamp DESC").Limit(1)
	l1DepositQuery = l1DepositQuery.Joins("INNER JOIN l1_contract_events ON l1_contract_events.guid = l1_transaction_deposits.initiated_l1_event_guid")
	l1DepositQuery = l1DepositQuery.Select("l1_contract_events.*")

	l1ProvenQuery := db.gorm.Table("l2_transaction_withdrawals")
	l1ProvenQuery = l1ProvenQuery.Joins("INNER JOIN l1_contract_events ON l1_contract_events.guid = l2_transaction_withdrawals.proven_l1_event_guid")
	l1ProvenQuery = l1ProvenQuery.Order("l1_contract_events.timestamp DESC").Select("l1_contract_events.*").Limit(1)

	l1FinalizedQuery := db.gorm.Table("l2_transaction_withdrawals")
	l1FinalizedQuery = l1FinalizedQuery.Joins("INNER JOIN l1_contract_events ON l1_contract_events.guid = l2_transaction_withdrawals.proven_l1_event_guid")
	l1FinalizedQuery = l1FinalizedQuery.Order("l1_contract_events.timestamp DESC").Select("l1_contract_events.*").Limit(1)

	l1Query := db.gorm.Table("((?) UNION (?) UNION (?)) AS latest_bridge_events", l1DepositQuery.Limit(1), l1ProvenQuery, l1FinalizedQuery)
	l1Query = l1Query.Joins("INNER JOIN l1_block_headers ON l1_block_headers.hash = latest_bridge_events.block_hash")
	l1Query = l1Query.Order("latest_bridge_events.timestamp DESC").Select("l1_block_headers.*")

	var l1Header L1BlockHeader
	result := l1Query.Take(&l1Header)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, result.Error
	}

	return &l1Header, nil
}

/**
 * Transactions withdrawn from L2
 */

func (db *bridgeTransactionsDB) StoreL2TransactionWithdrawals(withdrawals []L2TransactionWithdrawal) error {
	deduped := db.gorm.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "withdrawal_hash"}}, DoNothing: true})
	result := deduped.Create(&withdrawals)
	if result.Error == nil && int(result.RowsAffected) < len(withdrawals) {
		db.log.Warn("ignored L2 tx withdrawal duplicates", "duplicates", len(withdrawals)-int(result.RowsAffected))
	}

	return result.Error
}

func (db *bridgeTransactionsDB) L2TransactionWithdrawal(withdrawalHash common.Hash) (*L2TransactionWithdrawal, error) {
	var withdrawal L2TransactionWithdrawal
	result := db.gorm.Where(&L2TransactionWithdrawal{WithdrawalHash: withdrawalHash}).Take(&withdrawal)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, result.Error
	}

	return &withdrawal, nil
}

// MarkL2TransactionWithdrawalProvenEvent links a withdrawn transaction with associated Prove action on L1.
func (db *bridgeTransactionsDB) MarkL2TransactionWithdrawalProvenEvent(withdrawalHash common.Hash, provenL1EventGuid uuid.UUID) error {
	withdrawal, err := db.L2TransactionWithdrawal(withdrawalHash)
	if err != nil {
		return err
	} else if withdrawal == nil {
		return fmt.Errorf("transaction withdrawal hash %s not found", withdrawalHash)
	}

	if withdrawal.ProvenL1EventGUID != nil && withdrawal.ProvenL1EventGUID.ID() == provenL1EventGuid.ID() {
		return nil
	} else if withdrawal.ProvenL1EventGUID != nil {
		return fmt.Errorf("proven withdrawal %s re-proven with a different event %s", withdrawalHash, provenL1EventGuid)
	}

	withdrawal.ProvenL1EventGUID = &provenL1EventGuid
	result := db.gorm.Save(&withdrawal)
	return result.Error
}

// MarkL2TransactionWithdrawalProvenEvent links a withdrawn transaction in its finalized state
func (db *bridgeTransactionsDB) MarkL2TransactionWithdrawalFinalizedEvent(withdrawalHash common.Hash, finalizedL1EventGuid uuid.UUID, succeeded bool) error {
	withdrawal, err := db.L2TransactionWithdrawal(withdrawalHash)
	if err != nil {
		return err
	} else if withdrawal == nil {
		return fmt.Errorf("transaction withdrawal hash %s not found", withdrawalHash)
	} else if withdrawal.ProvenL1EventGUID == nil {
		return fmt.Errorf("cannot mark unproven withdrawal hash %s as finalized", withdrawal.WithdrawalHash)
	}

	if withdrawal.FinalizedL1EventGUID != nil && withdrawal.FinalizedL1EventGUID.ID() == finalizedL1EventGuid.ID() {
		return nil
	} else if withdrawal.FinalizedL1EventGUID != nil {
		return fmt.Errorf("finalized withdrawal %s re-finalized with a different event %s", withdrawalHash, finalizedL1EventGuid)
	}

	withdrawal.FinalizedL1EventGUID = &finalizedL1EventGuid
	withdrawal.Succeeded = &succeeded
	result := db.gorm.Save(&withdrawal)
	return result.Error
}

func (db *bridgeTransactionsDB) L2LatestBlockHeader() (*L2BlockHeader, error) {
	// L2: Latest Withdrawal, Latest L2 Header of indexed deposit epoch
	var latestWithdrawalHeader, latestL2DepositHeader *L2BlockHeader

	var withdrawHeader L2BlockHeader
	withdrawalQuery := db.gorm.Table("l2_transaction_withdrawals").Order("timestamp DESC").Limit(1)
	withdrawalQuery = withdrawalQuery.Joins("INNER JOIN l2_contract_events ON l2_contract_events.guid = l2_transaction_withdrawals.initiated_l2_event_guid")
	withdrawalQuery = withdrawalQuery.Joins("INNER JOIN l2_block_headers ON l2_block_headers.hash = l2_contract_events.block_hash")
	result := withdrawalQuery.Select("l2_block_headers.*").Take(&withdrawHeader)
	if result.Error != nil && !errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, result.Error
	} else if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
		latestWithdrawalHeader = &withdrawHeader
	}

	// Check for any deposits that may have been included after the latest withdrawal. However, since the bridge
	// processor only inserts entries when the corresponding epoch has been indexed on both L1 and L2, we can
	// simply look for the latest L2 block with at <= time of the latest L1 deposit.
	var l1Deposit L1TransactionDeposit
	result = db.gorm.Table("l1_transaction_deposits").Order("timestamp DESC").Limit(1).Take(&l1Deposit)
	if result.Error != nil && !errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, result.Error
	} else if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
		var l2DepositHeader L2BlockHeader
		result := db.gorm.Table("l2_block_headers").Order("timestamp DESC").Limit(1).Where("timestamp <= ?", l1Deposit.Tx.Timestamp).Take(&l2DepositHeader)
		if result.Error != nil && !errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, result.Error
		} else if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
			latestL2DepositHeader = &l2DepositHeader
		}
	}

	// compare
	if latestWithdrawalHeader == nil {
		return latestL2DepositHeader, nil
	} else if latestL2DepositHeader == nil {
		return latestWithdrawalHeader, nil
	}

	if latestWithdrawalHeader.Timestamp >= latestL2DepositHeader.Timestamp {
		return latestWithdrawalHeader, nil
	} else {
		return latestL2DepositHeader, nil
	}
}
