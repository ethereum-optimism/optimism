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

// Deposit represents an event emitted from the TeleportrDeposit contract on L1,
// along with additional info about the tx that generated the event.
type Deposit struct {
	ID             int64
	TxnHash        common.Hash
	BlockNumber    int64
	BlockTimestamp time.Time
	Address        common.Address
	Amount         *big.Int
}

// ConfirmationInfo holds metadata about a tx on either the L1 or L2 chain.
type ConfirmationInfo struct {
	TxnHash        common.Hash
	BlockNumber    int64
	BlockTimestamp time.Time
}

// CompletedTeleport represents an L1 deposit that has been disbursed on L2. The
// struct also hold info about the L1 and L2 txns involved.
type CompletedTeleport struct {
	ID           int64
	Address      common.Address
	Amount       *big.Int
	Deposit      ConfirmationInfo
	Disbursement ConfirmationInfo
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

const createDisbursementsTable = `
CREATE TABLE IF NOT EXISTS disbursements (
	id INT8 NOT NULL PRIMARY KEY REFERENCES deposits(id),
	txn_hash VARCHAR NOT NULL,
	block_number INT8 NOT NULL,
	block_timestamp TIMESTAMPTZ NOT NULL
);
`

var migrations = []string{
	createDepositsTable,
	createDisbursementsTable,
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
		return "enable"
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

const upsertDepositStatement = `
INSERT INTO deposits (id, txn_hash, block_number, block_timestamp, address, amount)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (id) DO UPDATE 
SET (txn_hash, block_number, block_timestamp, address, amount) = ($2, $3, $4, $5, $6)
`

// UpsertDeposits inserts a list of deposits into the database, or updats an
// existing deposit in place if the same ID is found.
func (d *Database) UpsertDeposits(deposits []Deposit) error {
	if len(deposits) == 0 {
		return nil
	}

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
	defer tx.Rollback()

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

	return tx.Commit()
}

const latestDepositQuery = `
SELECT block_number FROM deposits
ORDER BY block_number DESC
LIMIT 1
`

// LatestDeposit returns the block number of the latest deposit known to the
// database.
func (d *Database) LatestDeposit() (*int64, error) {
	row := d.conn.QueryRow(latestDepositQuery)

	var latestTransfer int64
	err := row.Scan(&latestTransfer)
	if err == sql.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	return &latestTransfer, nil
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
func (d *Database) ConfirmedDeposits(blockNumber, confirmations int64) ([]Deposit, error) {
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

const markDisbursedStatement = `
INSERT INTO disbursements (id, txn_hash, block_number, block_timestamp)
VALUES ($1, $2, $3, $4)
ON CONFLICT (id) DO UPDATE
SET (txn_hash, block_number, block_timestamp) = ($2, $3, $4)
`

// UpsertDisbursement inserts a disbursement, or updates an existing record
// in-place if the ID already exists.
func (d *Database) UpsertDisbursement(
	id int64,
	txnHash common.Hash,
	blockNumber int64,
	blockTimestamp time.Time,
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

const completedTeleportsQuery = `
SELECT
dep.id, dep.address, dep.amount,
dep.txn_hash, dep.block_number, dep.block_timestamp,
dis.txn_hash, dis.block_number, dis.block_timestamp
FROM deposits AS dep, disbursements AS dis
WHERE dep.id = dis.id
ORDER BY id DESC
`

// CompletedTeleports returns the set of all deposits that have also been
// disbursed.
func (d *Database) CompletedTeleports() ([]CompletedTeleport, error) {
	rows, err := d.conn.Query(completedTeleportsQuery)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var teleports []CompletedTeleport
	for rows.Next() {
		var teleport CompletedTeleport
		var addressStr string
		var amountStr string
		var depTxnHashStr string
		var disTxnHashStr string
		err = rows.Scan(
			&teleport.ID,
			&addressStr,
			&amountStr,
			&depTxnHashStr,
			&teleport.Deposit.BlockNumber,
			&teleport.Deposit.BlockTimestamp,
			&disTxnHashStr,
			&teleport.Disbursement.BlockNumber,
			&teleport.Disbursement.BlockTimestamp,
		)
		if err != nil {
			return nil, err
		}
		amount, ok := new(big.Int).SetString(amountStr, 10)
		if !ok {
			return nil, fmt.Errorf("unable to parse amount %v", amount)
		}
		teleport.Address = common.HexToAddress(addressStr)
		teleport.Amount = amount
		teleport.Deposit.TxnHash = common.HexToHash(depTxnHashStr)
		teleport.Deposit.BlockTimestamp = teleport.Deposit.BlockTimestamp.Local()
		teleport.Disbursement.TxnHash = common.HexToHash(disTxnHashStr)
		teleport.Disbursement.BlockTimestamp = teleport.Disbursement.BlockTimestamp.Local()

		teleports = append(teleports, teleport)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return teleports, nil
}
