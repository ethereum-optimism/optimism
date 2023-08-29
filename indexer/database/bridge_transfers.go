package database

import (
	"errors"

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
	LocalTokenAddress  common.Address `gorm:"serializer:bytes"`
	RemoteTokenAddress common.Address `gorm:"serializer:bytes"`
}

type BridgeTransfer struct {
	CrossDomainMessageHash *common.Hash `gorm:"serializer:bytes"`

	Tx        Transaction `gorm:"embedded"`
	TokenPair TokenPair   `gorm:"embedded"`
}

type L1BridgeDeposit struct {
	BridgeTransfer        `gorm:"embedded"`
	TransactionSourceHash common.Hash `gorm:"primaryKey;serializer:bytes"`
}

type L1BridgeDepositWithTransactionHashes struct {
	L1BridgeDeposit L1BridgeDeposit `gorm:"embedded"`

	L1TransactionHash common.Hash `gorm:"serializer:bytes"`
	L2TransactionHash common.Hash `gorm:"serializer:bytes"`
}

type L2BridgeWithdrawal struct {
	BridgeTransfer            `gorm:"embedded"`
	TransactionWithdrawalHash common.Hash `gorm:"primaryKey;serializer:bytes"`
}

type L2BridgeWithdrawalWithTransactionHashes struct {
	L2BridgeWithdrawal L2BridgeWithdrawal `gorm:"embedded"`
	L2TransactionHash  common.Hash        `gorm:"serializer:bytes"`

	ProvenL1TransactionHash    common.Hash `gorm:"serializer:bytes"`
	FinalizedL1TransactionHash common.Hash `gorm:"serializer:bytes"`
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

// L1BridgeDepositWithFilter queries for a bridge deposit with set fields in the `BridgeTransfer` filter
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

// L1BridgeDepositsByAddress retrieves a list of deposits intiated by the specified address,
// coupled with the L1/L2 transaction hashes that complete the bridge transaction.
func (db *bridgeTransfersDB) L1BridgeDepositsByAddress(address common.Address, cursor string, limit int) (*L1BridgeDepositsResponse, error) {
	defaultLimit := 100
	if limit <= 0 {
		limit = defaultLimit
	}

	// TODO join with l1_bridged_tokens and l2_bridged_tokens
	ethAddressString := predeploys.LegacyERC20ETHAddr.String()

	// Coalesce l1 transaction deposits that are simply ETH sends
	ethTransactionDeposits := db.gorm.Model(&L1TransactionDeposit{})
	ethTransactionDeposits = ethTransactionDeposits.Where(Transaction{FromAddress: address}).Where(`data = '0x' AND amount > 0`)
	ethTransactionDeposits = ethTransactionDeposits.Joins("INNER JOIN l1_contract_events ON l1_contract_events.guid = initiated_l1_event_guid")
	ethTransactionDeposits = ethTransactionDeposits.Select(`
from_address, to_address, amount, data, source_hash AS transaction_source_hash,
l2_transaction_hash, l1_contract_events.transaction_hash AS l1_transaction_hash,
l1_transaction_deposits.timestamp, NULL AS cross_domain_message_hash, ? AS local_token_address, ? AS remote_token_address`, ethAddressString, ethAddressString)

	if cursor != "" {
		// Probably need to fix this and compare timestamps
		ethTransactionDeposits = ethTransactionDeposits.Where("source_hash < ?", cursor)
	}

	depositsQuery := db.gorm.Model(&L1BridgeDeposit{})
	depositsQuery = depositsQuery.Joins("INNER JOIN l1_transaction_deposits ON l1_transaction_deposits.source_hash = transaction_source_hash")
	depositsQuery = depositsQuery.Joins("INNER JOIN l1_contract_events ON l1_contract_events.guid = l1_transaction_deposits.initiated_l1_event_guid")
	depositsQuery = depositsQuery.Select(`
l1_bridge_deposits.from_address, l1_bridge_deposits.to_address, l1_bridge_deposits.amount, l1_bridge_deposits.data, transaction_source_hash,
l2_transaction_hash, l1_contract_events.transaction_hash AS l1_transaction_hash,
l1_bridge_deposits.timestamp, cross_domain_message_hash, local_token_address, remote_token_address`)

	if cursor != "" {
		// Probably need to fix this and compare timestamps
		depositsQuery = depositsQuery.Where("source_hash < ?", cursor)
	}

	query := db.gorm.Table("(?) AS deposits", depositsQuery)
	query = query.Joins("UNION (?)", ethTransactionDeposits)
	query = query.Select("*").Order("timestamp DESC").Limit(limit + 1)
	deposits := []L1BridgeDepositWithTransactionHashes{}
	result := query.Debug().Find(&deposits)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, result.Error
	}

	nextCursor := ""
	hasNextPage := false
	if len(deposits) > limit {
		hasNextPage = true
		deposits = deposits[:limit]
		nextCursor = deposits[limit].L1TransactionHash.String()
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

// L2BridgeWithdrawalWithFilter queries for a bridge withdrawal with set fields in the `BridgeTransfer` filter
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

	// TODO join with l1_bridged_tokens and l2_bridged_tokens
	ethAddressString := predeploys.LegacyERC20ETHAddr.String()

	// Coalesce l2 transaction withdrawals that are simply ETH sends
	ethTransactionWithdrawals := db.gorm.Model(&L2TransactionWithdrawal{})
	ethTransactionWithdrawals = ethTransactionWithdrawals.Where(Transaction{FromAddress: address}).Where(`data = '0x' AND amount > 0`)
	ethTransactionWithdrawals = ethTransactionWithdrawals.Joins("INNER JOIN l2_contract_events ON l2_contract_events.guid = l2_transaction_withdrawals.initiated_l2_event_guid")
	ethTransactionWithdrawals = ethTransactionWithdrawals.Joins("LEFT JOIN l1_contract_events AS proven_l1_events ON proven_l1_events.guid = l2_transaction_withdrawals.proven_l1_event_guid")
	ethTransactionWithdrawals = ethTransactionWithdrawals.Joins("LEFT JOIN l1_contract_events AS finalized_l1_events ON finalized_l1_events.guid = l2_transaction_withdrawals.finalized_l1_event_guid")
	ethTransactionWithdrawals = ethTransactionWithdrawals.Select(`
from_address, to_address, amount, data, withdrawal_hash AS transaction_withdrawal_hash,
l2_contract_events.transaction_hash AS l2_transaction_hash, proven_l1_events.transaction_hash AS proven_l1_transaction_hash, finalized_l1_events.transaction_hash AS finalized_l1_transaction_hash,
l2_transaction_withdrawals.timestamp, NULL AS cross_domain_message_hash, ? AS local_token_address, ? AS remote_token_address`, ethAddressString, ethAddressString)

	if cursor != "" {
		// Probably need to fix this and compare timestamps
		ethTransactionWithdrawals = ethTransactionWithdrawals.Where("withdrawal_hash < ?", cursor)
	}

	withdrawalsQuery := db.gorm.Model(&L2BridgeWithdrawal{})
	withdrawalsQuery = withdrawalsQuery.Joins("INNER JOIN l2_transaction_withdrawals ON withdrawal_hash = l2_bridge_withdrawals.transaction_withdrawal_hash")
	withdrawalsQuery = withdrawalsQuery.Joins("INNER JOIN l2_contract_events ON l2_contract_events.guid = l2_transaction_withdrawals.initiated_l2_event_guid")
	withdrawalsQuery = withdrawalsQuery.Joins("LEFT JOIN l1_contract_events AS proven_l1_events ON proven_l1_events.guid = l2_transaction_withdrawals.proven_l1_event_guid")
	withdrawalsQuery = withdrawalsQuery.Joins("LEFT JOIN l1_contract_events AS finalized_l1_events ON finalized_l1_events.guid = l2_transaction_withdrawals.finalized_l1_event_guid")
	withdrawalsQuery = withdrawalsQuery.Select(`
l2_bridge_withdrawals.from_address, l2_bridge_withdrawals.to_address, l2_bridge_withdrawals.amount, l2_bridge_withdrawals.data, transaction_withdrawal_hash,
l2_contract_events.transaction_hash AS l2_transaction_hash, proven_l1_events.transaction_hash AS proven_l1_transaction_hash, finalized_l1_events.transaction_hash AS finalized_l1_transaction_hash,
l2_bridge_withdrawals.timestamp, cross_domain_message_hash, local_token_address, remote_token_address`)

	if cursor != "" {
		// Probably need to fix this and compare timestamps
		withdrawalsQuery = withdrawalsQuery.Where("withdrawal_hash < ?", cursor)
	}

	query := db.gorm.Table("(?) AS withdrawals", withdrawalsQuery)
	query = query.Joins("UNION (?)", ethTransactionWithdrawals)
	query = query.Select("*").Order("timestamp DESC").Limit(limit + 1)
	withdrawals := []L2BridgeWithdrawalWithTransactionHashes{}
	result := query.Scan(&withdrawals)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, result.Error
	}

	nextCursor := ""
	hasNextPage := false
	if len(withdrawals) > limit {
		hasNextPage = true
		withdrawals = withdrawals[:limit]
		nextCursor = withdrawals[limit].L2TransactionHash.String()
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
