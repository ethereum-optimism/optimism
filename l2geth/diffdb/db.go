package diffdb

import (
	"github.com/ethereum/go-ethereum/common"
	_ "github.com/mattn/go-sqlite3"

	"database/sql"
	"math/big"
)

type Key struct {
	Key     common.Hash
	Mutated bool
}

type Diff map[common.Address][]Key

/// A DiffDb is a thin wrapper around an Sqlite3 connection.
///
/// Its purpose is to store and fetch the storage keys corresponding to an address that was
/// touched in a block.
type DiffDb struct {
	db    *sql.DB
	tx    *sql.Tx
	stmt  *sql.Stmt
	cache uint64
	// We have a db-wide counter for the number of db calls made which we reset
	// whenever it hits `cache`.
	numCalls uint64
}

/// This key is used to mark that an account's state has been modified (e.g. nonce or balance)
/// and that an account proof is required.
var accountKey = common.HexToHash("0xDEADBEEFDEADBEEFDEADBEEFDEADBEEFDEADBEEFDEADBEEFDEADBEEFDEADBEEF")

var insertStatement = `
INSERT INTO diffs
    (block, address, key, mutated)
    VALUES
    ($1, $2, $3, $4)
ON CONFLICT DO NOTHING
`
var createStmt = `
CREATE TABLE IF NOT EXISTS diffs (
    block INTEGER,
    address STRING,
    key STRING,
    mutated BOOL,
    PRIMARY KEY (block, address, key)
)
`
var selectStmt = `
SELECT * from diffs WHERE block = $1
`

/// Inserts a new row to the sqlite with the provided diff data.
func (diff *DiffDb) SetDiffKey(block *big.Int, address common.Address, key common.Hash, mutated bool) error {
	// add 1 more insertion to the transaction
	_, err := diff.stmt.Exec(block.Uint64(), address, key, mutated)
	if err != nil {
		return err
	}

	// increment number of calls
	diff.numCalls += 1

	// if we had enough calls, commit it
	if diff.numCalls >= diff.cache {
		if err := diff.ForceCommit(); err != nil {
			return err
		}
	}

	return nil
}

/// Inserts a new row to the sqlite indicating that the account was modified in that block
/// at a pre-set key
func (diff *DiffDb) SetDiffAccount(block *big.Int, address common.Address) error {
	return diff.SetDiffKey(block, address, accountKey, true)
}

/// Commits a pending diffdb transaction
func (diff *DiffDb) ForceCommit() error {
	if err := diff.tx.Commit(); err != nil {
		return err
	}
	return diff.resetTx()
}

/// Gets all the rows for the matching block and converts them to a Diff map.
func (diff *DiffDb) GetDiff(blockNum *big.Int) (Diff, error) {
	// make the query
	rows, err := diff.db.Query(selectStmt, blockNum.Uint64())
	if err != nil {
		return nil, err
	}

	// initialize our data
	res := make(Diff)
	var block uint64
	var address common.Address
	var key common.Hash
	var mutated bool
	for rows.Next() {
		// deserialize the line
		err = rows.Scan(&block, &address, &key, &mutated)
		if err != nil {
			return nil, err
		}
		// add the data to the map
		res[address] = append(res[address], Key{key, mutated})
	}

	return res, rows.Err()
}

// Initializes the transaction which we will be using to commit data to the db
func (diff *DiffDb) resetTx() error {
	// reset the number of calls made
	diff.numCalls = 0

	// start a new tx
	tx, err := diff.db.Begin()
	if err != nil {
		return err
	}
	diff.tx = tx

	// the tx is about inserts
	stmt, err := diff.tx.Prepare(insertStatement)
	if err != nil {
		return err
	}
	diff.stmt = stmt

	return nil
}

func (diff *DiffDb) Close() error {
	return diff.db.Close()
}

/// Instantiates a new DiffDb using sqlite at `path`, with `cache` insertions
/// done in a transaction before it gets committed to the database.
func NewDiffDb(path string, cache uint64) (*DiffDb, error) {
	// get a handle
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, err
	}

	// create the table if it does not exist
	_, err = db.Exec(createStmt)
	if err != nil {
		return nil, err
	}

	diffdb := &DiffDb{db: db, cache: cache}

	// initialize the transaction
	if err := diffdb.resetTx(); err != nil {
		return nil, err
	}
	return diffdb, nil
}
