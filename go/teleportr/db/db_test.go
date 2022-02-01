package db_test

import (
	"database/sql"
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/go/teleportr/db"
	"github.com/ethereum/go-ethereum/common"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

var (
	testTimestamp = time.Unix(time.Now().Unix(), 0)
)

func newDatabase(t *testing.T) *db.Database {
	dbName := uuid.NewString()
	cfg := db.Config{
		Host:     "0.0.0.0",
		Port:     5432,
		User:     "postgres",
		Password: "password",
		DBName:   dbName,
	}

	conn, err := sql.Open("postgres", cfg.WithoutDB())
	require.Nil(t, err)

	_, err = conn.Exec(fmt.Sprintf("CREATE DATABASE \"%s\";", dbName))
	require.Nil(t, err)

	err = conn.Close()
	require.Nil(t, err)

	db, err := db.Open(cfg)
	require.Nil(t, err)

	err = db.Migrate()
	require.Nil(t, err)

	return db
}

// TestOpenClose asserts that we are able to open and close the database
// connection.
func TestOpenClose(t *testing.T) {
	t.Parallel()

	d := newDatabase(t)
	err := d.Close()
	require.Nil(t, err)
}

// TestUpsert empty deposits asserts that it is safe to call UpsertDeposits with
// an empty list.
func TestUpsertEmptyDeposits(t *testing.T) {
	t.Parallel()

	d := newDatabase(t)
	defer d.Close()

	err := d.UpsertDeposits(nil)
	require.Nil(t, err)

	err = d.UpsertDeposits([]db.Deposit{})
	require.Nil(t, err)
}

// TestUpsertDepositWithZeroTimestampFails asserts that trying to insert a
// deposit with a zero-timestamp fails.
func TestUpsertDepositWithZeroTimestampFails(t *testing.T) {
	t.Parallel()

	d := newDatabase(t)
	defer d.Close()

	err := d.UpsertDeposits([]db.Deposit{{}})
	require.Equal(t, db.ErrZeroTimestamp, err)
}

// TestLatestDeposit asserts that the LatestDeposit method properly returns the
// highest block number in the databse, or nil if no items are present.
func TestLatestDeposit(t *testing.T) {
	t.Parallel()

	d := newDatabase(t)
	defer d.Close()

	// Query should return nil on empty databse.
	latestDeposit, err := d.LatestDeposit()
	require.Nil(t, err)
	require.Equal(t, (*int64)(nil), latestDeposit)

	// Update table to have a single element.
	expLatestDeposit := int64(1)
	err = d.UpsertDeposits([]db.Deposit{{
		ID:             1,
		TxnHash:        common.HexToHash("0xf1"),
		BlockNumber:    expLatestDeposit,
		BlockTimestamp: testTimestamp,
		Address:        common.HexToAddress("0xa1"),
		Amount:         big.NewInt(1),
	}})
	require.Nil(t, err)

	// Query should return block number of only deposit.
	latestDeposit, err = d.LatestDeposit()
	require.Nil(t, err)
	require.Equal(t, &expLatestDeposit, latestDeposit)

	// Update table to have two distinct block numbers.
	expLatestDeposit = 2
	err = d.UpsertDeposits([]db.Deposit{{
		ID:             2,
		TxnHash:        common.HexToHash("0xf2"),
		BlockNumber:    expLatestDeposit,
		BlockTimestamp: testTimestamp,
		Address:        common.HexToAddress("0xa2"),
		Amount:         big.NewInt(2),
	}})
	require.Nil(t, err)

	// Query should return the highest of the two block numbers.
	latestDeposit, err = d.LatestDeposit()
	require.Nil(t, err)
	require.Equal(t, &expLatestDeposit, latestDeposit)
}

// TestUpsertDeposits asserts that UpsertDeposits properly overwrites an
// existing entry with the same ID.
func TestUpsertDeposits(t *testing.T) {
	t.Parallel()

	d := newDatabase(t)
	defer d.Close()

	deposit1 := db.Deposit{
		ID:             1,
		TxnHash:        common.HexToHash("0xff01"),
		BlockNumber:    1,
		BlockTimestamp: testTimestamp,
		Address:        common.HexToAddress("0xaa01"),
		Amount:         big.NewInt(1),
	}

	err := d.UpsertDeposits([]db.Deposit{deposit1})
	require.Nil(t, err)

	deposits, err := d.ConfirmedDeposits(1, 1)
	require.Nil(t, err)
	require.Equal(t, deposits, []db.Deposit{deposit1})

	deposit2 := db.Deposit{
		ID:             1,
		TxnHash:        common.HexToHash("0xff02"),
		BlockNumber:    2,
		BlockTimestamp: testTimestamp,
		Address:        common.HexToAddress("0xaa02"),
		Amount:         big.NewInt(2),
	}

	err = d.UpsertDeposits([]db.Deposit{deposit2})
	require.Nil(t, err)

	deposits, err = d.ConfirmedDeposits(2, 1)
	require.Nil(t, err)
	require.Equal(t, deposits, []db.Deposit{deposit2})
}

// TestConfirmedDeposits asserts that ConfirmedDeposits properly returns the set
// of deposits that have sufficient confirmation, but do not have a recorded
// disbursement.
func TestConfirmedDeposits(t *testing.T) {
	t.Parallel()

	d := newDatabase(t)
	defer d.Close()

	deposits, err := d.ConfirmedDeposits(1e9, 1)
	require.Nil(t, err)
	require.Equal(t, int(0), len(deposits))

	deposit1 := db.Deposit{
		ID:             1,
		TxnHash:        common.HexToHash("0xff01"),
		BlockNumber:    1,
		BlockTimestamp: testTimestamp,
		Address:        common.HexToAddress("0xaa01"),
		Amount:         big.NewInt(1),
	}
	deposit2 := db.Deposit{
		ID:             2,
		TxnHash:        common.HexToHash("0xff21"),
		BlockNumber:    2,
		BlockTimestamp: testTimestamp,
		Address:        common.HexToAddress("0xaa21"),
		Amount:         big.NewInt(2),
	}
	deposit3 := db.Deposit{
		ID:             3,
		TxnHash:        common.HexToHash("0xff22"),
		BlockNumber:    2,
		BlockTimestamp: testTimestamp,
		Address:        common.HexToAddress("0xaa22"),
		Amount:         big.NewInt(2),
	}

	err = d.UpsertDeposits([]db.Deposit{
		deposit1, deposit2, deposit3,
	})
	require.Nil(t, err)

	// First deposit only has 1 conf, should not be found using 2 confs at block
	// 1.
	deposits, err = d.ConfirmedDeposits(1, 2)
	require.Nil(t, err)
	require.Equal(t, int(0), len(deposits))

	// First deposit should be returned when querying for 1 conf at block 1.
	deposits, err = d.ConfirmedDeposits(1, 1)
	require.Nil(t, err)
	require.Equal(t, []db.Deposit{deposit1}, deposits)

	// All deposits should be returned when querying for 1 conf at block 2.
	deposits, err = d.ConfirmedDeposits(2, 1)
	require.Nil(t, err)
	require.Equal(t, []db.Deposit{deposit1, deposit2, deposit3}, deposits)

	err = d.UpsertDisbursement(deposit1.ID, common.HexToHash("0xdd01"), 1, testTimestamp)
	require.Nil(t, err)

	deposits, err = d.ConfirmedDeposits(2, 1)
	require.Nil(t, err)
	require.Equal(t, []db.Deposit{deposit2, deposit3}, deposits)
}

// TestUpsertDisbursement asserts that UpsertDisbursement properly inserts new
// disbursements or overwrites existing ones.
func TestUpsertDisbursement(t *testing.T) {
	t.Parallel()

	d := newDatabase(t)
	defer d.Close()

	address := common.HexToAddress("0xaa01")
	amount := big.NewInt(1)
	depTxnHash := common.HexToHash("0xdd01")
	depBlockNumber := int64(1)
	disTxnHash := common.HexToHash("0xee02")
	disBlockNumber := int64(2)

	// Calling UpsertDisbursement with the zero timestamp should fail.
	err := d.UpsertDisbursement(0, common.HexToHash("0xdd00"), 0, time.Time{})
	require.Equal(t, db.ErrZeroTimestamp, err)

	// Calling UpsertDisbursement with an unknown id should fail.
	err = d.UpsertDisbursement(0, common.HexToHash("0xdd00"), 0, testTimestamp)
	require.Equal(t, db.ErrUnknownDeposit, err)

	// Now, insert a real deposit that we will disburse.
	err = d.UpsertDeposits([]db.Deposit{
		{
			ID:             1,
			TxnHash:        depTxnHash,
			BlockNumber:    depBlockNumber,
			BlockTimestamp: testTimestamp,
			Address:        address,
			Amount:         amount,
		},
	})
	require.Nil(t, err)

	// Mark the deposit as disbursed with some temporary info.
	err = d.UpsertDisbursement(1, common.HexToHash("0xee00"), 1, testTimestamp)
	require.Nil(t, err)

	// Overwrite the disbursement info with the final values.
	err = d.UpsertDisbursement(1, disTxnHash, disBlockNumber, testTimestamp)
	require.Nil(t, err)

	expTeleports := []db.CompletedTeleport{
		{
			ID:      1,
			Address: address,
			Amount:  amount,
			Deposit: db.ConfirmationInfo{
				TxnHash:        depTxnHash,
				BlockNumber:    depBlockNumber,
				BlockTimestamp: testTimestamp,
			},
			Disbursement: db.ConfirmationInfo{
				TxnHash:        disTxnHash,
				BlockNumber:    disBlockNumber,
				BlockTimestamp: testTimestamp,
			},
		},
	}

	// Assert that the deposit now shows up in the CompletedTeleports method
	// with both the L1 and L2 confirmation info.
	teleports, err := d.CompletedTeleports()
	require.Nil(t, err)
	require.Equal(t, expTeleports, teleports)
}
