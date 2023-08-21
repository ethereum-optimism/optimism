package database

import (
	"errors"
	"fmt"

	"gorm.io/gorm"

	"github.com/ethereum-optimism/optimism/op-bindings/predeploys"
	"github.com/ethereum/go-ethereum/common"
)

var (
	ETHTokenPair = TokenPair{LocalTokenAddress: predeploys.LegacyERC20ETHAddr, RemoteTokenAddress: predeploys.LegacyERC20ETHAddr}
)

/**
 * Types
 */

type TokenPair struct {
	LocalTokenAddress  common.Address `gorm:"serializer:json"`
	RemoteTokenAddress common.Address `gorm:"serializer:json"`
}

type BridgeTransfer struct {
	CrossDomainMessageHash *common.Hash `gorm:"serializer:json"`

	Tx        Transaction `gorm:"embedded"`
	TokenPair TokenPair   `gorm:"embedded"`
}

type L1BridgeDeposit struct {
	BridgeTransfer        `gorm:"embedded"`
	TransactionSourceHash common.Hash `gorm:"primaryKey;serializer:json"`
}

type L1BridgeDepositWithTransactionHashes struct {
	L1BridgeDeposit L1BridgeDeposit `gorm:"embedded"`

	L1TransactionHash common.Hash `gorm:"serializer:json"`
	L2TransactionHash common.Hash `gorm:"serializer:json"`
}

type L2BridgeWithdrawal struct {
	BridgeTransfer            `gorm:"embedded"`
	TransactionWithdrawalHash common.Hash `gorm:"primaryKey;serializer:json"`
}

type L2BridgeWithdrawalWithTransactionHashes struct {
	L2BridgeWithdrawal L2BridgeWithdrawal `gorm:"embedded"`
	L2TransactionHash  common.Hash        `gorm:"serializer:json"`

	ProvenL1TransactionHash    common.Hash `gorm:"serializer:json"`
	FinalizedL1TransactionHash common.Hash `gorm:"serializer:json"`
}

type BridgeTransfersView interface {
	L1BridgeDeposit(common.Hash) (*L1BridgeDeposit, error)
	L1BridgeDepositWithFilter(BridgeTransfer) (*L1BridgeDeposit, error)
	L1BridgeDepositsByAddress(common.Address, string, int) (*L1BridgeDepositsResponse, error)

	L2BridgeWithdrawal(common.Hash) (*L2BridgeWithdrawal, error)
	L2BridgeWithdrawalWithFilter(BridgeTransfer) (*L2BridgeWithdrawal, error)
	L2BridgeWithdrawalsByAddress(common.Address, string, int) (*L2BridgeWithdrawalsResponse, error)
}

type BridgeTransfersDB interface {
	BridgeTransfersView

	StoreL1BridgeDeposits([]L1BridgeDeposit) error
	StoreL2BridgeWithdrawals([]L2BridgeWithdrawal) error
}

/**
 * Implementation
 */

type bridgeTransfersDB struct {
	gorm *gorm.DB
}

func newBridgeTransfersDB(db *gorm.DB) BridgeTransfersDB {
	return &bridgeTransfersDB{gorm: db}
}

/**
 * Tokens Bridged (Deposited) from L1
 */

func (db *bridgeTransfersDB) StoreL1BridgeDeposits(deposits []L1BridgeDeposit) error {
	result := db.gorm.Create(&deposits)
	return result.Error
}

func (db *bridgeTransfersDB) L1BridgeDeposit(txSourceHash common.Hash) (*L1BridgeDeposit, error) {
	var deposit L1BridgeDeposit
	result := db.gorm.Where(&L1BridgeDeposit{TransactionSourceHash: txSourceHash}).Take(&deposit)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, result.Error
	}

	return &deposit, nil
}

// L1BridgeDepositByCrossDomainMessengerNonce retrieves tokens deposited, specified by the associated `L1CrossDomainMessenger` nonce.
// All tokens bridged via the StandardBridge flows through the L1CrossDomainMessenger
func (db *bridgeTransfersDB) L1BridgeDepositWithFilter(filter BridgeTransfer) (*L1BridgeDeposit, error) {
	var deposit L1BridgeDeposit
	result := db.gorm.Where(&filter).Take(&deposit)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, result.Error
	}

	return &deposit, nil
}

type L1BridgeDepositsResponse struct {
	Deposits    []L1BridgeDepositWithTransactionHashes
	Cursor      string
	HasNextPage bool
}

// L1BridgeDepositsByAddress retrieves a list of deposits intiated by the specified address, coupled with the L1/L2 transaction
// hashes that complete the bridge transaction.
func (db *bridgeTransfersDB) L1BridgeDepositsByAddress(address common.Address, cursor string, limit int) (*L1BridgeDepositsResponse, error) {
	defaultLimit := 100
	if limit <= 0 {
		limit = defaultLimit
	}

	depositsQuery := db.gorm.Table("l1_bridge_deposits").Select(`
l1_bridge_deposits.*,
l1_contract_events.transaction_hash AS l1_transaction_hash,
l1_transaction_deposits.l2_transaction_hash`)

	// TODO join with l1_tokens and l2_tokens
	depositsQuery = depositsQuery.Joins("INNER JOIN l1_transaction_deposits ON l1_bridge_deposits.transaction_source_hash = l1_transaction_deposits.source_hash")
	depositsQuery = depositsQuery.Joins("INNER JOIN l1_contract_events ON l1_transaction_deposits.initiated_l1_event_guid = l1_contract_events.guid")

	if cursor != "" {
		depositsQuery = depositsQuery.Where("l1_bridge_deposits.transaction_source_hash < ?", cursor)
	}

	filteredQuery := depositsQuery.Where(&Transaction{FromAddress: address}).Order("l1_bridge_deposits.transaction_source_hash DESC").Limit(limit + 1)

	deposits := []L1BridgeDepositWithTransactionHashes{}
	result := filteredQuery.Scan(&deposits)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, result.Error
	}

	hasNextPage := false
	if len(deposits) > limit {
		hasNextPage = true
		deposits = deposits[:limit]
	}

	nextCursor := ""
	if hasNextPage {
		nextCursor = deposits[len(deposits)-1].L1TransactionHash.String()
	}

	response := &L1BridgeDepositsResponse{
		Deposits:    deposits,
		Cursor:      nextCursor,
		HasNextPage: hasNextPage,
	}

	return response, nil
}

/**
 * Tokens Bridged (Withdrawn) from L2
 */

func (db *bridgeTransfersDB) StoreL2BridgeWithdrawals(withdrawals []L2BridgeWithdrawal) error {
	result := db.gorm.Create(&withdrawals)
	return result.Error
}

func (db *bridgeTransfersDB) L2BridgeWithdrawal(txWithdrawalHash common.Hash) (*L2BridgeWithdrawal, error) {
	var withdrawal L2BridgeWithdrawal
	result := db.gorm.Where(&L2BridgeWithdrawal{TransactionWithdrawalHash: txWithdrawalHash}).Take(&withdrawal)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, result.Error
	}

	return &withdrawal, nil
}

// L2BridgeWithdrawalByCrossDomainMessengerNonce retrieves tokens withdrawn, specified by the associated `L2CrossDomainMessenger` nonce.
// All tokens bridged via the StandardBridge flows through the L2CrossDomainMessenger
func (db *bridgeTransfersDB) L2BridgeWithdrawalWithFilter(filter BridgeTransfer) (*L2BridgeWithdrawal, error) {
	var withdrawal L2BridgeWithdrawal
	result := db.gorm.Where(filter).Take(&withdrawal)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, result.Error
	}

	return &withdrawal, nil
}

type L2BridgeWithdrawalsResponse struct {
	Withdrawals []L2BridgeWithdrawalWithTransactionHashes
	Cursor      string
	HasNextPage bool
}

// L2BridgeDepositsByAddress retrieves a list of deposits intiated by the specified address, coupled with the L1/L2 transaction hashes
// that complete the bridge transaction. The hashes that correspond to with the Bedrock multistep withdrawal process are also surfaced
func (db *bridgeTransfersDB) L2BridgeWithdrawalsByAddress(address common.Address, cursor string, limit int) (*L2BridgeWithdrawalsResponse, error) {
	defaultLimit := 100
	if limit <= 0 {
		limit = defaultLimit
	}

	withdrawalsQuery := db.gorm.Table("l2_bridge_withdrawals").Select(`
l2_bridge_withdrawals.*,
l2_contract_events.transaction_hash AS l2_transaction_hash,
proven_l1_contract_events.transaction_hash AS proven_l1_transaction_hash,
finalized_l1_contract_events.transaction_hash AS finalized_l1_transaction_hash`)

	withdrawalsQuery = withdrawalsQuery.Joins("INNER JOIN l2_transaction_withdrawals ON l2_bridge_withdrawals.transaction_withdrawal_hash = l2_transaction_withdrawals.withdrawal_hash")
	withdrawalsQuery = withdrawalsQuery.Joins("INNER JOIN l2_contract_events ON l2_transaction_withdrawals.initiated_l2_event_guid = l2_contract_events.guid")
	withdrawalsQuery = withdrawalsQuery.Joins("LEFT JOIN l1_contract_events AS proven_l1_contract_events ON l2_transaction_withdrawals.proven_l1_event_guid = proven_l1_contract_events.guid")
	withdrawalsQuery = withdrawalsQuery.Joins("LEFT JOIN l1_contract_events AS finalized_l1_contract_events ON l2_transaction_withdrawals.finalized_l1_event_guid = finalized_l1_contract_events.guid")

	if cursor != "" {
		withdrawalsQuery = withdrawalsQuery.Where("l2_bridge_withdrawals.id < ?", cursor)
	}

	filteredQuery := withdrawalsQuery.Where(&Transaction{FromAddress: address}).Order("l2_bridge_withdrawals.timestamp DESC").Limit(limit + 1)

	withdrawals := []L2BridgeWithdrawalWithTransactionHashes{}
	result := filteredQuery.Scan(&withdrawals)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, result.Error
	}

	hasNextPage := false
	if len(withdrawals) > limit {
		hasNextPage = true
		withdrawals = withdrawals[:limit]
	}

	nextCursor := ""
	if hasNextPage {
		nextCursor = fmt.Sprintf("%d", withdrawals[len(withdrawals)-1].L2TransactionHash)
	}

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, result.Error
	}
	response := &L2BridgeWithdrawalsResponse{
		Withdrawals: withdrawals,
		Cursor:      nextCursor,
		HasNextPage: hasNextPage,
	}

	return response, nil
}
