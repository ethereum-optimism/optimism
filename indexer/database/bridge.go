package database

import (
	"errors"
	"fmt"
	"math/big"

	"gorm.io/gorm"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"

	"github.com/google/uuid"
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
	GUID                 uuid.UUID `gorm:"primaryKey"`
	InitiatedL1EventGUID uuid.UUID

	// Since we're only currently indexing a single StandardBridge,
	// the message nonce serves as a unique identifier for this
	// deposit. Once this generalizes to more than 1 deployed
	// bridge, we need to include the `CrossDomainMessenger` address
	// such that the (messenger_addr, nonce) is the unique identifier
	// for a bridge msg
	SentMessageNonce U256

	FinalizedL2EventGUID *uuid.UUID

	Tx        Transaction `gorm:"embedded"`
	TokenPair TokenPair   `gorm:"embedded"`
}

type DepositWithTransactionHashes struct {
	Deposit           Deposit     `gorm:"embedded"`
	L1TransactionHash common.Hash `gorm:"serializer:json"`

	FinalizedL2TransactionHash common.Hash `gorm:"serializer:json"`
}

type Withdrawal struct {
	GUID                 uuid.UUID `gorm:"primaryKey"`
	InitiatedL2EventGUID uuid.UUID

	// Since we're only currently indexing a single StandardBridge,
	// the message nonce serves as a unique identifier for this
	// withdrawal. Once this generalizes to more than 1 deployed
	// bridge, we need to include the `CrossDomainMessenger` address
	// such that the (messenger_addr, nonce) is the unique identifier
	// for a bridge msg
	SentMessageNonce U256

	WithdrawalHash       common.Hash `gorm:"serializer:json"`
	ProvenL1EventGUID    *uuid.UUID
	FinalizedL1EventGUID *uuid.UUID

	Tx        Transaction `gorm:"embedded"`
	TokenPair TokenPair   `gorm:"embedded"`
}

type WithdrawalWithTransactionHashes struct {
	Withdrawal        Withdrawal  `gorm:"embedded"`
	L2TransactionHash common.Hash `gorm:"serializer:json"`

	ProvenL1TransactionHash    common.Hash `gorm:"serializer:json"`
	FinalizedL1TransactionHash common.Hash `gorm:"serializer:json"`
}

type BridgeView interface {
	DepositsByAddress(address common.Address) ([]*DepositWithTransactionHashes, error)
	DepositByMessageNonce(*big.Int) (*Deposit, error)
	LatestDepositMessageNonce() (*big.Int, error)

	WithdrawalsByAddress(address common.Address) ([]*WithdrawalWithTransactionHashes, error)
	WithdrawalByMessageNonce(*big.Int) (*Withdrawal, error)
	WithdrawalByHash(common.Hash) (*Withdrawal, error)
	LatestWithdrawalMessageNonce() (*big.Int, error)
}

type BridgeDB interface {
	BridgeView

	StoreDeposits([]*Deposit) error
	MarkFinalizedDepositEvent(uuid.UUID, uuid.UUID) error

	StoreWithdrawals([]*Withdrawal) error
	MarkProvenWithdrawalEvent(uuid.UUID, uuid.UUID) error
	MarkFinalizedWithdrawalEvent(uuid.UUID, uuid.UUID) error
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

func (db *bridgeDB) DepositsByAddress(address common.Address) ([]*DepositWithTransactionHashes, error) {
	depositsQuery := db.gorm.Table("deposits").Select("deposits.*, l1_contract_events.transaction_hash AS l1_transaction_hash, l2_contract_events.transaction_hash AS finalized_l2_transaction_hash")

	initiatedJoinQuery := depositsQuery.Joins("LEFT JOIN l1_contract_events ON deposits.initiated_l1_event_guid = l1_contract_events.guid")
	finalizedJoinQuery := initiatedJoinQuery.Joins("LEFT JOIN l2_contract_events ON deposits.finalized_l2_event_guid = l2_contract_events.guid")

	// add in cursoring options
	filteredQuery := finalizedJoinQuery.Where(&Transaction{FromAddress: address}).Order("deposits.timestamp DESC").Limit(100)

	deposits := make([]*DepositWithTransactionHashes, 100)
	result := filteredQuery.Scan(&deposits)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}

		return nil, result.Error
	}

	return deposits, nil
}

func (db *bridgeDB) DepositByMessageNonce(nonce *big.Int) (*Deposit, error) {
	var deposit Deposit
	result := db.gorm.Where(&Deposit{SentMessageNonce: U256{Int: nonce}}).Take(&deposit)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}

		return nil, result.Error
	}

	return &deposit, nil
}

func (db *bridgeDB) LatestDepositMessageNonce() (*big.Int, error) {
	var deposit Deposit
	result := db.gorm.Order("sent_message_nonce DESC").Take(&deposit)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}

		return nil, result.Error
	}

	return deposit.SentMessageNonce.Int, nil
}

func (db *bridgeDB) MarkFinalizedDepositEvent(guid, finalizationEventGUID uuid.UUID) error {
	var deposit Deposit
	result := db.gorm.Where(&Deposit{GUID: guid}).Take(&deposit)
	if result.Error != nil {
		return result.Error
	}

	deposit.FinalizedL2EventGUID = &finalizationEventGUID
	result = db.gorm.Save(&deposit)
	return result.Error
}

// Withdrawals

func (db *bridgeDB) StoreWithdrawals(withdrawals []*Withdrawal) error {
	result := db.gorm.Create(&withdrawals)
	return result.Error
}

func (db *bridgeDB) MarkProvenWithdrawalEvent(guid, provenL1EventGuid uuid.UUID) error {
	var withdrawal Withdrawal
	result := db.gorm.Where(&Withdrawal{GUID: guid}).Take(&withdrawal)
	if result.Error != nil {
		return result.Error
	}

	withdrawal.ProvenL1EventGUID = &provenL1EventGuid
	result = db.gorm.Save(&withdrawal)
	return result.Error
}

func (db *bridgeDB) MarkFinalizedWithdrawalEvent(guid, finalizedL1EventGuid uuid.UUID) error {
	var withdrawal Withdrawal
	result := db.gorm.Where(&Withdrawal{GUID: guid}).Take(&withdrawal)
	if result.Error != nil {
		return result.Error
	}

	if withdrawal.ProvenL1EventGUID == nil {
		return fmt.Errorf("withdrawal %s marked finalized prior to being proven", guid)
	}

	withdrawal.FinalizedL1EventGUID = &finalizedL1EventGuid
	result = db.gorm.Save(&withdrawal)
	return result.Error
}

func (db *bridgeDB) WithdrawalsByAddress(address common.Address) ([]*WithdrawalWithTransactionHashes, error) {
	withdrawalsQuery := db.gorm.Table("withdrawals").Select("withdrawals.*, l2_contract_events.transaction_hash AS l2_transaction_hash, proven_l1_contract_events.transaction_hash AS proven_l1_transaction_hash, finalized_l1_contract_events.transaction_hash AS finalized_l1_transaction_hash")

	eventsJoinQuery := withdrawalsQuery.Joins("LEFT JOIN l2_contract_events ON withdrawals.initiated_l2_event_guid = l2_contract_events.guid")
	provenJoinQuery := eventsJoinQuery.Joins("LEFT JOIN l1_contract_events AS proven_l1_contract_events ON withdrawals.proven_l1_event_guid = proven_l1_contract_events.guid")
	finalizedJoinQuery := provenJoinQuery.Joins("LEFT JOIN l1_contract_events AS finalized_l1_contract_events ON withdrawals.finalized_l1_event_guid = finalized_l1_contract_events.guid")

	// add in cursoring options
	filteredQuery := finalizedJoinQuery.Where(&Transaction{FromAddress: address}).Order("withdrawals.timestamp DESC").Limit(100)

	withdrawals := make([]*WithdrawalWithTransactionHashes, 100)
	result := filteredQuery.Scan(&withdrawals)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}

		return nil, result.Error
	}

	return withdrawals, nil
}

func (db *bridgeDB) WithdrawalByMessageNonce(nonce *big.Int) (*Withdrawal, error) {
	var withdrawal Withdrawal
	result := db.gorm.Where(&Withdrawal{SentMessageNonce: U256{Int: nonce}}).Take(&withdrawal)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}

		return nil, result.Error
	}

	return &withdrawal, nil
}

func (db *bridgeDB) WithdrawalByHash(hash common.Hash) (*Withdrawal, error) {
	var withdrawal Withdrawal
	result := db.gorm.Where(&Withdrawal{WithdrawalHash: hash}).Take(&withdrawal)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}

		return nil, result.Error
	}

	return &withdrawal, nil
}

func (db *bridgeDB) LatestWithdrawalMessageNonce() (*big.Int, error) {
	var withdrawal Withdrawal
	result := db.gorm.Order("sent_message_nonce DESC").Take(&withdrawal)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}

		return nil, result.Error
	}

	return withdrawal.SentMessageNonce.Int, nil
}
