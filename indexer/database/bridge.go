package database

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"gorm.io/gorm"
)

/**
 * Types
 */

type Transaction struct {
	FromAddress common.Address `gorm:"serializer:json"`
	ToAddress   common.Address `gorm:"serializer:json"`
	Amount      U256
	Data        hexutil.Bytes `gorm:"serializer:json"`
	Timestamp   uint64
}

type TokenPair struct {
	L1TokenAddress common.Address `gorm:"serializer:json"`
	L2TokenAddress common.Address `gorm:"serializer:json"`
}

type Deposit struct {
	GUID                 string `gorm:"primaryKey"`
	InitiatedL1EventGUID string

	Tx        Transaction `gorm:"embedded"`
	TokenPair TokenPair   `gorm:"embedded"`
}

type DepositWithTransactionHash struct {
	Deposit           Deposit     `gorm:"embedded"`
	L1TransactionHash common.Hash `gorm:"serializer:json"`
}

type Withdrawal struct {
	GUID                 string `gorm:"primaryKey"`
	InitiatedL2EventGUID string

	WithdrawalHash       common.Hash `gorm:"serializer:json"`
	ProvenL1EventGUID    *string
	FinalizedL1EventGUID *string

	Tx        Transaction `gorm:"embedded"`
	TokenPair TokenPair   `gorm:"embedded"`
}

type WithdrawalWithTransactionHashes struct {
	Withdrawal        Withdrawal  `gorm:"embedded"`
	L2TransactionHash common.Hash `gorm:"serializer:json"`

	ProvenL1TransactionHash    *common.Hash `gorm:"serializer:json"`
	FinalizedL1TransactionHash *common.Hash `gorm:"serializer:json"`
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
	depositsQuery := db.gorm.Table("deposits").Select("deposits.*, l1_contract_events.transaction_hash AS l1_transaction_hash")
	eventsJoinQuery := depositsQuery.Joins("LEFT JOIN l1_contract_events ON deposits.initiated_l1_event_guid = l1_contract_events.guid")

	// add in cursoring options
	filteredQuery := eventsJoinQuery.Where(&Transaction{FromAddress: address}).Order("deposits.timestamp DESC").Limit(100)

	deposits := make([]*DepositWithTransactionHash, 100)
	result := filteredQuery.Scan(&deposits)
	if result.Error != nil {
		return nil, result.Error
	}

	return deposits, nil
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
		withdrawal.ProvenL1EventGUID = &provenL1EventGuid
		db.gorm.Save(&withdrawal)
	}

	return result.Error
}

func (db *bridgeDB) MarkFinalizedWithdrawalEvent(guid, finalizedL1EventGuid string) error {
	var withdrawal Withdrawal
	result := db.gorm.First(&withdrawal, "guid = ?", guid)
	if result.Error == nil {
		withdrawal.FinalizedL1EventGUID = &finalizedL1EventGuid
		db.gorm.Save(&withdrawal)
	}

	return result.Error
}

func (db *bridgeDB) WithdrawalsByAddress(address common.Address) ([]*WithdrawalWithTransactionHashes, error) {
	withdrawalsQuery := db.gorm.Debug().Table("withdrawals").Select("withdrawals.*, l2_contract_events.transaction_hash AS l2_transaction_hash, proven_l1_contract_events.transaction_hash AS proven_l1_transaction_hash, finalized_l1_contract_events.transaction_hash AS finalized_l1_transaction_hash")

	eventsJoinQuery := withdrawalsQuery.Joins("LEFT JOIN l2_contract_events ON withdrawals.initiated_l2_event_guid = l2_contract_events.guid")
	provenJoinQuery := eventsJoinQuery.Joins("LEFT JOIN l1_contract_events AS proven_l1_contract_events ON withdrawals.proven_l1_event_guid = proven_l1_contract_events.guid")
	finalizedJoinQuery := provenJoinQuery.Joins("LEFT JOIN l1_contract_events AS finalized_l1_contract_events ON withdrawals.finalized_l1_event_guid = finalized_l1_contract_events.guid")

	// add in cursoring options
	filteredQuery := finalizedJoinQuery.Where(&Transaction{FromAddress: address}).Order("withdrawals.timestamp DESC").Limit(100)

	withdrawals := make([]*WithdrawalWithTransactionHashes, 100)
	result := filteredQuery.Scan(&withdrawals)
	if result.Error != nil {
		return nil, result.Error
	}

	return withdrawals, nil
}
