package database

import (
	"database/sql"

	"github.com/ethereum/go-ethereum/common"
	"github.com/jackc/pgtype"
	"gorm.io/gorm"
)

/**
 * Types
 */

type Transaction struct {
	fromAddress common.Address
	toAddress   common.Address
	amount      pgtype.Numeric
	data        []byte
}

type TokenPair struct {
	l1TokenAddress common.Address
	l2TokenAddress common.Address
}

type Deposit struct {
	GUID                 string
	InitiatedL1EventGUID string

	Tx        Transaction `gorm:"embedded"`
	TokenPair TokenPair   `gorm:"embedded"`
}

type DepositWithTransactionHash struct {
	Deposit           *Deposit `gorm:"embedded"`
	L1TransactionHash common.Hash
}

type Withdrawal struct {
	GUID                 string
	InitiatedL2EventGUID string

	WithdrawalHash       common.Hash
	ProvenL1EventGUID    sql.NullString
	FinalizedL1EventGUID sql.NullString

	Tx        Transaction `gorm:"embedded"`
	TokenPair TokenPair   `gorm:"embedded"`
}

type WithdrawalWithTransactionHashes struct {
	Withdrawal        *Withdrawal `gorm:"embedded"`
	L2TransactionHash common.Hash

	ProvenL1TransactionHash    *common.Hash
	FinalizedL1TransactionHash *common.Hash
}

type BridgeView interface {
	DepositsByAddress(address common.Address) ([]*DepositWithTransactionHash, error)
	WithdrawalsByAddress(address common.Address) ([]*WithdrawalWithTransactionHashes, error)
}

type BridgeDB interface {
	BridgeView

	StoreDeposits([]*Deposit) error
	StoreWithdrawals([]*Withdrawal) error
	MarkProvenWithdrawalEvent(string, string) error
	MarkFinalizedWithdrawalEvent(string, string) error
}

/**
 * Implementation
 */

type bridgeDB struct {
	gorm *gorm.DB
}

func newBridgeDB(db *gorm.DB) BridgeDB {
	return &bridgeDB{gorm: db}
}

// Deposits

func (db *bridgeDB) StoreDeposits(deposits []*Deposit) error {
	result := db.gorm.Create(&deposits)
	return result.Error
}

func (db *bridgeDB) DepositsByAddress(address common.Address) ([]*DepositWithTransactionHash, error) {
	// validate this query
	depositsQuery := db.gorm.Table("deposits").Where("from_address = ?", address).Select("deposits.*")
	joinQuery := depositsQuery.Joins("left join l1_contract_events transaction_hash as l1_transaction_hash ON deposit.initiated_l1_event_guid = l1_contract_events.guid")

	deposits := []DepositWithTransactionHash{}

	result := joinQuery.Scan(&deposits)
	if result.Error != nil {
		return nil, result.Error
	}

	depositPtrs := make([]*DepositWithTransactionHash, len(deposits))
	for i, deposit := range deposits {
		depositPtrs[i] = &deposit
	}
	return depositPtrs, nil
}

// Withdrawals

func (db *bridgeDB) StoreWithdrawals(withdrawals []*Withdrawal) error {
	result := db.gorm.Create(&withdrawals)
	return result.Error
}

func (db *bridgeDB) MarkProvenWithdrawalEvent(guid, provenL1EventGuid string) error {
	var withdrawal Withdrawal
	result := db.gorm.First(&withdrawal, "guid = ?", guid)
	if result.Error == nil {
		withdrawal.ProvenL1EventGUID = sql.NullString{String: provenL1EventGuid, Valid: true}
		db.gorm.Save(&withdrawal)
	}

	return result.Error
}

func (db *bridgeDB) MarkFinalizedWithdrawalEvent(guid, finalizedL1EventGuid string) error {
	var withdrawal Withdrawal
	result := db.gorm.First(&withdrawal, "guid = ?", guid)
	if result.Error == nil {
		withdrawal.FinalizedL1EventGUID = sql.NullString{String: finalizedL1EventGuid, Valid: true}
		db.gorm.Save(&withdrawal)
	}

	return result.Error
}

func (db *bridgeDB) WithdrawalsByAddress(address common.Address) ([]*WithdrawalWithTransactionHashes, error) {
	// Implement this query
	return nil, nil
}
