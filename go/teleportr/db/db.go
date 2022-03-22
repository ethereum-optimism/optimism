package db

import (
	"database/sql"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	_ "github.com/lib/pq"
)

var (
	// ErrZeroTimestamp signals that the caller attempted to insert deposits
	// with a timestamp of zero.
	ErrZeroTimestamp = errors.New("timestamp is zero")

	// ErrUnknownDeposit signals that the target deposit could not be found.
	ErrUnknownDeposit = errors.New("unknown deposit")
)

// ConfirmationInfo holds metadata about a tx on either the L1 or L2 chain.
type ConfirmationInfo struct {
	TxnHash        common.Hash
	BlockNumber    uint64
	BlockTimestamp time.Time
}

// Deposit represents an event emitted from the TeleportrDeposit contract on L1,
// along with additional info about the tx that generated the event.
type Deposit struct {
	ID      uint64
	Address common.Address
	Amount  *big.Int

	ConfirmationInfo
}

type Disbursement struct {
	Success bool

	ConfirmationInfo
}

// Teleport represents the combination of an L1 deposit and its disbursement on
// L2. Disburment will be nil if the L2 disbursement has not occurred.
type Teleport struct {
	Deposit

	Disbursement *Disbursement
}

const createDepositsTable = `
CREATE TABLE IF NOT EXISTS deposits (
	id INT8 NOT NULL PRIMARY KEY,
	txn_hash VARCHAR NOT NULL,
	block_number INT8 NOT NULL,
	block_timestamp TIMESTAMPTZ NOT NULL,
	address VARCHAR NOT NULL,
	amount VARCHAR NOT NULL
);
`

const createDepositTxnHashIndex = `
CREATE INDEX ON deposits (txn_hash)
`

const createDepositAddressIndex = `
CREATE INDEX ON deposits (address)
`

const createDisbursementsTable = `
CREATE TABLE IF NOT EXISTS disbursements (
	id INT8 NOT NULL PRIMARY KEY REFERENCES deposits(id),
	txn_hash VARCHAR NOT NULL,
	block_number INT8 NOT NULL,
	block_timestamp TIMESTAMPTZ NOT NULL,
	success BOOL NOT NULL
);
`

const lastProcessedBlockTable = `
CREATE TABLE IF NOT EXISTS last_processed_block (
	id BOOL PRIMARY KEY DEFAULT TRUE,
	value INT8 NOT NULL,
	CONSTRAINT id CHECK (id)
);
`

const pendingTxTable = `
CREATE TABLE IF NOT EXISTS pending_txs (
	txn_hash VARCHAR NOT NULL PRIMARY KEY,
	start_id INT8 NOT NULL,
	end_id INT8 NOT NULL
);
`

var migrations = []string{
	createDepositsTable,
	createDepositTxnHashIndex,
	createDepositAddressIndex,
	createDisbursementsTable,
	lastProcessedBlockTable,
	pendingTxTable,
}

// Config houses the data required to connect to a Postgres backend.
type Config struct {
	// Host is the database hostname.
	Host string
	// Port is the database port.

	Port uint16

	// User is the database user to log in as.
	User string

	// Password is the user's password to authenticate.
	Password string

	// DBName is the name of the database to connect to.
	DBName string

	// EnableSSL enables SLL on the connection if set to true.
	EnableSSL bool
}

// WithDB returns the connection string with a specific database to connect to.
func (c Config) WithDB() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.DBName, c.sslMode(),
	)
}

// WithoutDB returns the connection string without connecting to a specific
// database.
func (c Config) WithoutDB() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.sslMode(),
	)
}

// sslMode retuns "enabled" if EnableSSL is true, otherwise returns "disabled".
func (c Config) sslMode() string {
	if c.EnableSSL {
		return "require"
	}
	return "disable"
}

// Database provides a Go API for accessing Teleportr read/write operations.
type Database struct {
	conn *sql.DB
}

// Open creates a new database connection to the configured Postgres backend and
// applies any migrations.
func Open(cfg Config) (*Database, error) {
	conn, err := sql.Open("postgres", cfg.WithDB())
	if err != nil {
		return nil, err
	}

	return &Database{
		conn: conn,
	}, nil
}

// Migrate applies all existing migrations to the open database.
func (d *Database) Migrate() error {
	for _, migration := range migrations {
		_, err := d.conn.Exec(migration)
		if err != nil {
			return err
		}
	}

	return nil
}

// Close closes the connection to the database.
func (d *Database) Close() error {
	return d.conn.Close()
}

const upsertLastProcessedBlock = `
INSERT INTO last_processed_block (value)
VALUES ($1)
ON CONFLICT (id) DO UPDATE
SET value = $1
`

const upsertDepositStatement = `
INSERT INTO deposits (id, txn_hash, block_number, block_timestamp, address, amount)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (id) DO UPDATE 
SET (txn_hash, block_number, block_timestamp, address, amount) = ($2, $3, $4, $5, $6)
`

// UpsertDeposits inserts a list of deposits into the database, or updats an
// existing deposit in place if the same ID is found.
func (d *Database) UpsertDeposits(
	deposits []Deposit,
	lastProcessedBlock uint64,
) error {

	// Sanity check deposits.
	for _, deposit := range deposits {
		if deposit.BlockTimestamp.IsZero() {
			return ErrZeroTimestamp
		}
	}

	tx, err := d.conn.Begin()
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	for _, deposit := range deposits {
		_, err = tx.Exec(
			upsertDepositStatement,
			deposit.ID,
			deposit.TxnHash.String(),
			deposit.BlockNumber,
			deposit.BlockTimestamp,
			deposit.Address.String(),
			deposit.Amount.String(),
		)
		if err != nil {
			return err
		}
	}

	_, err = tx.Exec(upsertLastProcessedBlock, lastProcessedBlock)
	if err != nil {
		return err
	}

	return tx.Commit()
}

const lastProcessedBlockQuery = `
SELECT value FROM last_processed_block
`

func (d *Database) LastProcessedBlock() (*uint64, error) {
	row := d.conn.QueryRow(lastProcessedBlockQuery)

	var lastProcessedBlock uint64
	err := row.Scan(&lastProcessedBlock)
	if err == sql.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	return &lastProcessedBlock, nil
}

const confirmedDepositsQuery = `
SELECT dep.*
FROM deposits AS dep
LEFT JOIN disbursements AS dis ON dep.id = dis.id
WHERE dis.id IS NULL AND dep.block_number + $1 <= $2 + 1
ORDER BY dep.id ASC
`

// ConfirmedDeposits returns the set of all deposits that have sufficient
// confirmation, but do not have a recorded disbursement.
func (d *Database) ConfirmedDeposits(blockNumber, confirmations uint64) ([]Deposit, error) {
	rows, err := d.conn.Query(confirmedDepositsQuery, confirmations, blockNumber)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var deposits []Deposit
	for rows.Next() {
		var deposit Deposit
		var txnHashStr string
		var addressStr string
		var amountStr string
		err = rows.Scan(
			&deposit.ID,
			&txnHashStr,
			&deposit.BlockNumber,
			&deposit.BlockTimestamp,
			&addressStr,
			&amountStr,
		)
		if err != nil {
			return nil, err
		}
		amount, ok := new(big.Int).SetString(amountStr, 10)
		if !ok {
			return nil, fmt.Errorf("unable to parse amount %v", amount)
		}
		deposit.TxnHash = common.HexToHash(txnHashStr)
		deposit.BlockTimestamp = deposit.BlockTimestamp.Local()
		deposit.Amount = amount
		deposit.Address = common.HexToAddress(addressStr)

		deposits = append(deposits, deposit)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return deposits, nil
}

const latestDisbursementIDQuery = `
SELECT id FROM disbursements
ORDER BY id DESC
LIMIT 1
`

// LatestDisbursementID returns the latest deposit id known to the database that
// has a recorded disbursement.
func (d *Database) LatestDisbursementID() (*uint64, error) {
	row := d.conn.QueryRow(latestDisbursementIDQuery)

	var latestDisbursementID uint64
	err := row.Scan(&latestDisbursementID)
	if err == sql.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	return &latestDisbursementID, nil
}

const markDisbursedStatement = `
INSERT INTO disbursements (id, txn_hash, block_number, block_timestamp, success)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (id) DO UPDATE
SET (txn_hash, block_number, block_timestamp, success) = ($2, $3, $4, $5)
`

// UpsertDisbursement inserts a disbursement, or updates an existing record
// in-place if the ID already exists.
func (d *Database) UpsertDisbursement(
	id uint64,
	txnHash common.Hash,
	blockNumber uint64,
	blockTimestamp time.Time,
	success bool,
) error {
	if blockTimestamp.IsZero() {
		return ErrZeroTimestamp
	}

	result, err := d.conn.Exec(
		markDisbursedStatement,
		id,
		txnHash.String(),
		blockNumber,
		blockTimestamp,
		success,
	)
	if err != nil {
		if strings.Contains(err.Error(), "violates foreign key constraint") {
			return ErrUnknownDeposit
		}
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected != 1 {
		return ErrUnknownDeposit
	}
	return nil
}

const loadTeleportByDepositHashQuery = `
SELECT
dep.id, dep.address, dep.amount, dis.success,
dep.txn_hash, dep.block_number, dep.block_timestamp,
dis.txn_hash, dis.block_number, dis.block_timestamp
FROM deposits AS dep
LEFT JOIN disbursements AS dis
ON dep.id = dis.id
WHERE dep.txn_hash = $1
LIMIT 1
`

func (d *Database) LoadTeleportByDepositHash(
	txHash common.Hash,
) (*Teleport, error) {

	row := d.conn.QueryRow(loadTeleportByDepositHashQuery, txHash.String())
	teleport, err := scanTeleport(row)
	if err == sql.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	return &teleport, nil
}

const loadTeleportsByAddressQuery = `
SELECT
dep.id, dep.address, dep.amount, dis.success,
dep.txn_hash, dep.block_number, dep.block_timestamp,
dis.txn_hash, dis.block_number, dis.block_timestamp
FROM deposits AS dep
LEFT JOIN disbursements AS dis
ON dep.id = dis.id
WHERE dep.address = $1
ORDER BY dep.block_timestamp DESC, dep.id DESC
LIMIT 100
`

func (d *Database) LoadTeleportsByAddress(
	addr common.Address,
) ([]Teleport, error) {

	rows, err := d.conn.Query(loadTeleportsByAddressQuery, addr.String())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var teleports []Teleport
	for rows.Next() {
		teleport, err := scanTeleport(rows)
		if err != nil {
			return nil, err
		}
		teleports = append(teleports, teleport)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return teleports, nil
}

const completedTeleportsQuery = `
SELECT
dep.id, dep.address, dep.amount, dis.success,
dep.txn_hash, dep.block_number, dep.block_timestamp,
dis.txn_hash, dis.block_number, dis.block_timestamp
FROM deposits AS dep, disbursements AS dis
WHERE dep.id = dis.id
ORDER BY id DESC
`

// CompletedTeleports returns the set of all deposits that have also been
// disbursed.
func (d *Database) CompletedTeleports() ([]Teleport, error) {
	rows, err := d.conn.Query(completedTeleportsQuery)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var teleports []Teleport
	for rows.Next() {
		teleport, err := scanTeleport(rows)
		if err != nil {
			return nil, err
		}
		teleports = append(teleports, teleport)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return teleports, nil
}

type Scanner interface {
	Scan(...interface{}) error
}

func scanTeleport(scanner Scanner) (Teleport, error) {
	var teleport Teleport
	var addressStr string
	var amountStr string
	var depTxnHashStr string
	var disTxnHashStr *string
	var disBlockNumber *uint64
	var disBlockTimestamp *time.Time
	var success *bool
	err := scanner.Scan(
		&teleport.ID,
		&addressStr,
		&amountStr,
		&success,
		&depTxnHashStr,
		&teleport.Deposit.BlockNumber,
		&teleport.Deposit.BlockTimestamp,
		&disTxnHashStr,
		&disBlockNumber,
		&disBlockTimestamp,
	)
	if err != nil {
		return Teleport{}, err
	}

	amount, ok := new(big.Int).SetString(amountStr, 10)
	if !ok {
		return Teleport{}, fmt.Errorf("unable to parse amount %v", amount)
	}
	teleport.Address = common.HexToAddress(addressStr)
	teleport.Amount = amount
	teleport.Deposit.TxnHash = common.HexToHash(depTxnHashStr)
	teleport.Deposit.BlockTimestamp = teleport.Deposit.BlockTimestamp.Local()

	hasDisbursement := success != nil &&
		disTxnHashStr != nil &&
		disBlockNumber != nil &&
		disBlockTimestamp != nil

	if hasDisbursement {
		teleport.Disbursement = &Disbursement{
			ConfirmationInfo: ConfirmationInfo{
				TxnHash:        common.HexToHash(*disTxnHashStr),
				BlockNumber:    *disBlockNumber,
				BlockTimestamp: disBlockTimestamp.Local(),
			},
			Success: *success,
		}
	}

	return teleport, nil
}

// PendingTx encapsulates the metadata stored about published disbursement txs.
type PendingTx struct {
	// Txhash is the tx hash of the disbursement tx.
	TxHash common.Hash

	// StartID is the deposit id of the first disbursement, inclusive.
	StartID uint64

	// EndID is the deposit id fo the last disbursement, exclusive.
	EndID uint64
}

const upsertPendingTxStatement = `
INSERT INTO pending_txs (txn_hash, start_id, end_id)
VALUES ($1, $2, $3)
ON CONFLICT (txn_hash) DO UPDATE
SET (start_id, end_id) = ($2, $3)
`

// UpsertPendingTx inserts a disbursement, or updates the entry if the TxHash
// already exists.
func (d *Database) UpsertPendingTx(pendingTx PendingTx) error {
	_, err := d.conn.Exec(
		upsertPendingTxStatement,
		pendingTx.TxHash.String(),
		pendingTx.StartID,
		pendingTx.EndID,
	)
	return err
}

const listPendingTxsQuery = `
SELECT txn_hash, start_id, end_id
FROM pending_txs
ORDER BY start_id DESC, end_id DESC, txn_hash ASC
`

// ListPendingTxs returns all pending txs stored in the database.
func (d *Database) ListPendingTxs() ([]PendingTx, error) {
	rows, err := d.conn.Query(listPendingTxsQuery)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var pendingTxs []PendingTx
	for rows.Next() {
		var pendingTx PendingTx
		var txHashStr string
		err = rows.Scan(
			&txHashStr,
			&pendingTx.StartID,
			&pendingTx.EndID,
		)
		if err != nil {
			return nil, err
		}
		pendingTx.TxHash = common.HexToHash(txHashStr)

		pendingTxs = append(pendingTxs, pendingTx)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return pendingTxs, nil
}

const deletePendingTxsStatement = `
DELETE FROM pending_txs
WHERE start_id = $1 AND end_id = $2
`

// DeletePendingTx removes any pending txs with matching start and end ids. This
// allows the caller to remove any logically-conflicting pending txs from the
// database after successfully processing the outcomes.
func (d *Database) DeletePendingTx(startID, endID uint64) error {
	_, err := d.conn.Exec(
		deletePendingTxsStatement,
		startID,
		endID,
	)
	return err
}
