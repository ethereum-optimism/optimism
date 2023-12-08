package database

import (
	"errors"
	"fmt"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/ethereum-optimism/optimism/op-bindings/predeploys"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
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

	L1BlockHash       common.Hash `gorm:"serializer:bytes"`
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
	L2BlockHash        common.Hash        `gorm:"serializer:bytes"`

	ProvenL1TransactionHash    common.Hash `gorm:"serializer:bytes"`
	FinalizedL1TransactionHash common.Hash `gorm:"serializer:bytes"`
}

type BridgeTransfersView interface {
	L1BridgeDeposit(common.Hash) (*L1BridgeDeposit, error)
	L1TxDepositSum() (float64, error)
	L1BridgeDepositWithFilter(BridgeTransfer) (*L1BridgeDeposit, error)
	L1BridgeDepositsByAddress(common.Address, string, int) (*L1BridgeDepositsResponse, error)

	L2BridgeWithdrawal(common.Hash) (*L2BridgeWithdrawal, error)
	L2BridgeWithdrawalSum(filter WithdrawFilter) (float64, error)
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
	log  log.Logger
	gorm *gorm.DB
}

func newBridgeTransfersDB(log log.Logger, db *gorm.DB) BridgeTransfersDB {
	return &bridgeTransfersDB{log: log.New("table", "bridge_transfers"), gorm: db}
}

/**
 * Tokens Bridged (Deposited) from L1
 */

func (db *bridgeTransfersDB) StoreL1BridgeDeposits(deposits []L1BridgeDeposit) error {
	deduped := db.gorm.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "transaction_source_hash"}}, DoNothing: true})
	result := deduped.Create(&deposits)
	if result.Error == nil && int(result.RowsAffected) < len(deposits) {
		db.log.Warn("ignored L1 bridge transfer duplicates", "duplicates", len(deposits)-int(result.RowsAffected))
	}

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

// L1TxDepositSum ... returns the sum of all l1 tx deposit mints in gwei
func (db *bridgeTransfersDB) L1TxDepositSum() (float64, error) {
	var sum float64
	result := db.gorm.Model(&L1TransactionDeposit{}).Select("SUM(amount)").Scan(&sum)
	if result.Error != nil {
		return 0, result.Error
	}

	return sum, nil
}

// L1BridgeDepositsByAddress retrieves a list of deposits initiated by the specified address,
// coupled with the L1/L2 transaction hashes that complete the bridge transaction.
func (db *bridgeTransfersDB) L1BridgeDepositsByAddress(address common.Address, cursor string, limit int) (*L1BridgeDepositsResponse, error) {
	if limit <= 0 {
		return nil, fmt.Errorf("limit must be greater than 0")
	}

	cursorClause := ""
	if cursor != "" {
		sourceHash := common.HexToHash(cursor)
		txDeposit := new(L1TransactionDeposit)
		result := db.gorm.Model(&L1TransactionDeposit{}).Where(&L1TransactionDeposit{SourceHash: sourceHash}).Take(txDeposit)
		if result.Error != nil || errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("unable to find transaction with supplied cursor source hash %s: %w", sourceHash, result.Error)
		}
		cursorClause = fmt.Sprintf("l1_transaction_deposits.timestamp <= %d", txDeposit.Tx.Timestamp)
	}

	ethAddressString := predeploys.LegacyERC20ETHAddr.String()

	// Coalesce l1 transaction deposits that are simply ETH sends
	ethTransactionDeposits := db.gorm.Model(&L1TransactionDeposit{})
	ethTransactionDeposits = ethTransactionDeposits.Where(&Transaction{FromAddress: address}).Where("amount > 0")
	ethTransactionDeposits = ethTransactionDeposits.Joins("INNER JOIN l1_contract_events ON l1_contract_events.guid = initiated_l1_event_guid")
	ethTransactionDeposits = ethTransactionDeposits.Select(`
from_address, to_address, amount, data, source_hash AS transaction_source_hash,
l2_transaction_hash, l1_contract_events.transaction_hash AS l1_transaction_hash, l1_contract_events.block_hash as l1_block_hash,
l1_transaction_deposits.timestamp, NULL AS cross_domain_message_hash, ? AS local_token_address, ? AS remote_token_address`, ethAddressString, ethAddressString)
	ethTransactionDeposits = ethTransactionDeposits.Order("timestamp DESC").Limit(limit + 1)
	if cursorClause != "" {
		ethTransactionDeposits = ethTransactionDeposits.Where(cursorClause)
	}

	depositsQuery := db.gorm.Model(&L1BridgeDeposit{})
	depositsQuery = depositsQuery.Where(&Transaction{FromAddress: address})
	depositsQuery = depositsQuery.Joins("INNER JOIN l1_transaction_deposits ON l1_transaction_deposits.source_hash = transaction_source_hash")
	depositsQuery = depositsQuery.Joins("INNER JOIN l1_contract_events ON l1_contract_events.guid = l1_transaction_deposits.initiated_l1_event_guid")
	depositsQuery = depositsQuery.Select(`
l1_bridge_deposits.from_address, l1_bridge_deposits.to_address, l1_bridge_deposits.amount, l1_bridge_deposits.data, transaction_source_hash,
l2_transaction_hash, l1_contract_events.transaction_hash AS l1_transaction_hash, l1_contract_events.block_hash as l1_block_hash,
l1_bridge_deposits.timestamp, cross_domain_message_hash, local_token_address, remote_token_address`)
	depositsQuery = depositsQuery.Order("timestamp DESC").Limit(limit + 1)
	if cursorClause != "" {
		depositsQuery = depositsQuery.Where(cursorClause)
	}

	query := db.gorm.Table("(?) AS deposits", depositsQuery)
	query = query.Joins("UNION (?)", ethTransactionDeposits)
	query = query.Select("*").Order("timestamp DESC").Limit(limit + 1)
	deposits := []L1BridgeDepositWithTransactionHashes{}
	result := query.Find(&deposits)
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
		nextCursor = deposits[limit].L1BridgeDeposit.TransactionSourceHash.String()
		deposits = deposits[:limit]
	}

	response := &L1BridgeDepositsResponse{Deposits: deposits, Cursor: nextCursor, HasNextPage: hasNextPage}
	return response, nil
}

/**
 * Tokens Bridged (Withdrawn) from L2
 */

func (db *bridgeTransfersDB) StoreL2BridgeWithdrawals(withdrawals []L2BridgeWithdrawal) error {
	deduped := db.gorm.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "transaction_withdrawal_hash"}}, DoNothing: true})
	result := deduped.Create(&withdrawals)
	if result.Error == nil && int(result.RowsAffected) < len(withdrawals) {
		db.log.Warn("ignored L2 bridge transfer duplicates", "duplicates", len(withdrawals)-int(result.RowsAffected))
	}

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

type WithdrawFilter uint8

const (
	All WithdrawFilter = iota // Same as "initialized"
	Proven
	Finalized
)

func (db *bridgeTransfersDB) L2BridgeWithdrawalSum(filter WithdrawFilter) (float64, error) {
	// Determine where filter
	var clause string
	switch filter {
	case All:
		clause = ""

	case Finalized:
		clause = "finalized_l1_event_guid IS NOT NULL"

	case Proven:
		clause = "proven_l1_event_guid IS NOT NULL"

	default:
		return 0, fmt.Errorf("unknown filter argument: %d", filter)
	}

	// NOTE - Scanning to float64 reduces precision versus scanning to big.Int since amount is a uint256
	// This is ok though given all bridges will never exceed max float64 (10^308 || 1.7E+308) in wei value locked
	// since that would require 10^308 / 10^18 = 10^290 ETH locked in the bridge
	var sum float64
	result := db.gorm.Model(&L2TransactionWithdrawal{}).Where(clause).Select("SUM(amount)").Scan(&sum)
	if result.Error != nil && strings.Contains(result.Error.Error(), "converting NULL to float64 is unsupported") {
		// no rows found
		return 0, nil
	} else if result.Error != nil {
		return 0, result.Error
	} else {
		return sum, nil
	}
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

// L2BridgeDepositsByAddress retrieves a list of deposits initiated by the specified address, coupled with the L1/L2 transaction hashes
// that complete the bridge transaction. The hashes that correspond with the Bedrock multi-step withdrawal process are also surfaced
func (db *bridgeTransfersDB) L2BridgeWithdrawalsByAddress(address common.Address, cursor string, limit int) (*L2BridgeWithdrawalsResponse, error) {
	if limit <= 0 {
		return nil, fmt.Errorf("limit must be greater than 0")
	}

	// (1) Generate cursor clause provided a cursor tx hash
	cursorClause := ""
	if cursor != "" {
		withdrawalHash := common.HexToHash(cursor)
		var txWithdrawal L2TransactionWithdrawal
		result := db.gorm.Model(&L2TransactionWithdrawal{}).Where(&L2TransactionWithdrawal{WithdrawalHash: withdrawalHash}).Take(&txWithdrawal)
		if result.Error != nil || errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("unable to find transaction with supplied cursor withdrawal hash %s: %w", withdrawalHash, result.Error)
		}
		cursorClause = fmt.Sprintf("l2_transaction_withdrawals.timestamp <= %d", txWithdrawal.Tx.Timestamp)
	}

	// (2) Generate query for fetching ETH withdrawal data
	// This query is a UNION (A | B) of two sub-queries:
	//   - (A) ETH sends from L2 to L1
	//   - (B) Bridge withdrawals from L2 to L1

	// TODO join with l1_bridged_tokens and l2_bridged_tokens
	ethAddressString := predeploys.LegacyERC20ETHAddr.String()

	// Coalesce l2 transaction withdrawals that are simply ETH sends
	ethTransactionWithdrawals := db.gorm.Model(&L2TransactionWithdrawal{})
	ethTransactionWithdrawals = ethTransactionWithdrawals.Where(&Transaction{FromAddress: address}).Where("amount > 0")
	ethTransactionWithdrawals = ethTransactionWithdrawals.Joins("INNER JOIN l2_contract_events ON l2_contract_events.guid = l2_transaction_withdrawals.initiated_l2_event_guid")
	ethTransactionWithdrawals = ethTransactionWithdrawals.Joins("LEFT JOIN l1_contract_events AS proven_l1_events ON proven_l1_events.guid = l2_transaction_withdrawals.proven_l1_event_guid")
	ethTransactionWithdrawals = ethTransactionWithdrawals.Joins("LEFT JOIN l1_contract_events AS finalized_l1_events ON finalized_l1_events.guid = l2_transaction_withdrawals.finalized_l1_event_guid")
	ethTransactionWithdrawals = ethTransactionWithdrawals.Select(`
from_address, to_address, amount, data, withdrawal_hash AS transaction_withdrawal_hash,
l2_contract_events.transaction_hash AS l2_transaction_hash, l2_contract_events.block_hash as l2_block_hash, proven_l1_events.transaction_hash AS proven_l1_transaction_hash, finalized_l1_events.transaction_hash AS finalized_l1_transaction_hash,
l2_transaction_withdrawals.timestamp, NULL AS cross_domain_message_hash, ? AS local_token_address, ? AS remote_token_address`, ethAddressString, ethAddressString)
	ethTransactionWithdrawals = ethTransactionWithdrawals.Order("timestamp DESC").Limit(limit + 1)
	if cursorClause != "" {
		ethTransactionWithdrawals = ethTransactionWithdrawals.Where(cursorClause)
	}

	withdrawalsQuery := db.gorm.Model(&L2BridgeWithdrawal{})
	withdrawalsQuery = withdrawalsQuery.Where(&Transaction{FromAddress: address})
	withdrawalsQuery = withdrawalsQuery.Joins("INNER JOIN l2_transaction_withdrawals ON withdrawal_hash = l2_bridge_withdrawals.transaction_withdrawal_hash")
	withdrawalsQuery = withdrawalsQuery.Joins("INNER JOIN l2_contract_events ON l2_contract_events.guid = l2_transaction_withdrawals.initiated_l2_event_guid")
	withdrawalsQuery = withdrawalsQuery.Joins("LEFT JOIN l1_contract_events AS proven_l1_events ON proven_l1_events.guid = l2_transaction_withdrawals.proven_l1_event_guid")
	withdrawalsQuery = withdrawalsQuery.Joins("LEFT JOIN l1_contract_events AS finalized_l1_events ON finalized_l1_events.guid = l2_transaction_withdrawals.finalized_l1_event_guid")
	withdrawalsQuery = withdrawalsQuery.Select(`
l2_bridge_withdrawals.from_address, l2_bridge_withdrawals.to_address, l2_bridge_withdrawals.amount, l2_bridge_withdrawals.data, transaction_withdrawal_hash,
l2_contract_events.transaction_hash AS l2_transaction_hash, l2_contract_events.block_hash as l2_block_hash, proven_l1_events.transaction_hash AS proven_l1_transaction_hash, finalized_l1_events.transaction_hash AS finalized_l1_transaction_hash,
l2_bridge_withdrawals.timestamp, cross_domain_message_hash, local_token_address, remote_token_address`)
	withdrawalsQuery = withdrawalsQuery.Order("timestamp DESC").Limit(limit + 1)
	if cursorClause != "" {
		withdrawalsQuery = withdrawalsQuery.Where(cursorClause)
	}

	query := db.gorm.Table("(?) AS withdrawals", withdrawalsQuery)
	query = query.Joins("UNION (?)", ethTransactionWithdrawals)
	query = query.Select("*").Order("timestamp DESC").Limit(limit + 1)
	withdrawals := []L2BridgeWithdrawalWithTransactionHashes{}

	// (3) Execute query and process results
	result := query.Find(&withdrawals)

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
		nextCursor = withdrawals[limit].L2BridgeWithdrawal.TransactionWithdrawalHash.String()
		withdrawals = withdrawals[:limit]
	}

	response := &L2BridgeWithdrawalsResponse{Withdrawals: withdrawals, Cursor: nextCursor, HasNextPage: hasNextPage}
	return response, nil
}
