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

// TestUpsertEmptyDeposits empty deposits asserts that it is safe to call
// UpsertDeposits with an empty list.
func TestUpsertEmptyDeposits(t *testing.T) {
	t.Parallel()

	d := newDatabase(t)
	defer d.Close()

	err := d.UpsertDeposits(nil, 0)
	require.Nil(t, err)

	err = d.UpsertDeposits([]db.Deposit{}, 0)
	require.Nil(t, err)
}

// TestUpsertDepositWithZeroTimestampFails asserts that trying to insert a
// deposit with a zero-timestamp fails.
func TestUpsertDepositWithZeroTimestampFails(t *testing.T) {
	t.Parallel()

	d := newDatabase(t)
	defer d.Close()

	err := d.UpsertDeposits([]db.Deposit{{}}, 0)
	require.Equal(t, db.ErrZeroTimestamp, err)
}

// TestUpsertDeposits asserts that UpsertDeposits properly overwrites an
// existing entry with the same ID.
func TestUpsertDeposits(t *testing.T) {
	t.Parallel()

	d := newDatabase(t)
	defer d.Close()

	deposit1 := db.Deposit{
		ID:      1,
		Address: common.HexToAddress("0xaa01"),
		Amount:  big.NewInt(1),
		ConfirmationInfo: db.ConfirmationInfo{
			TxnHash:        common.HexToHash("0xff01"),
			BlockNumber:    1,
			BlockTimestamp: testTimestamp,
		},
	}

	err := d.UpsertDeposits([]db.Deposit{deposit1}, 0)
	require.Nil(t, err)

	deposits, err := d.ConfirmedDeposits(1, 1)
	require.Nil(t, err)
	require.Equal(t, deposits, []db.Deposit{deposit1})

	deposit2 := db.Deposit{
		ID:      1,
		Address: common.HexToAddress("0xaa02"),
		Amount:  big.NewInt(2),
		ConfirmationInfo: db.ConfirmationInfo{
			TxnHash:        common.HexToHash("0xff02"),
			BlockNumber:    2,
			BlockTimestamp: testTimestamp,
		},
	}

	err = d.UpsertDeposits([]db.Deposit{deposit2}, 0)
	require.Nil(t, err)

	deposits, err = d.ConfirmedDeposits(2, 1)
	require.Nil(t, err)
	require.Equal(t, deposits, []db.Deposit{deposit2})
}

// TestUpsertDepositsRecordsLastProcessedBlock asserts that calling
// UpsertDeposits properly records the last processed block.
func TestUpsertDepositsRecordsLastProcessedBlock(t *testing.T) {
	t.Parallel()

	d := newDatabase(t)
	defer d.Close()

	uint64Ptr := func(x uint64) *uint64 {
		return &x
	}

	// Should be empty initially.
	lastProcessedBlock, err := d.LastProcessedBlock()
	require.Nil(t, err)
	require.Nil(t, lastProcessedBlock)

	// Insert nil deposits through block 1.
	err = d.UpsertDeposits(nil, 1)
	require.Nil(t, err)

	// Check that LastProcessedBlock returns 1.
	lastProcessedBlock, err = d.LastProcessedBlock()
	require.Nil(t, err)
	require.Equal(t, uint64Ptr(1), lastProcessedBlock)

	// Insert empty deposits through block 2.
	err = d.UpsertDeposits([]db.Deposit{}, 2)
	require.Nil(t, err)

	// Check that LastProcessedBlock returns 2.
	lastProcessedBlock, err = d.LastProcessedBlock()
	require.Nil(t, err)
	require.Equal(t, uint64Ptr(2), lastProcessedBlock)

	// Insert real deposit in block 3 with last processed at 4.
	deposit := db.Deposit{
		ID:      0,
		Address: common.HexToAddress("0xaa03"),
		Amount:  big.NewInt(3),
		ConfirmationInfo: db.ConfirmationInfo{
			TxnHash:        common.HexToHash("0xff03"),
			BlockNumber:    3,
			BlockTimestamp: testTimestamp,
		},
	}
	err = d.UpsertDeposits([]db.Deposit{deposit}, 4)
	require.Nil(t, err)

	// Check that LastProcessedBlock returns 2.
	lastProcessedBlock, err = d.LastProcessedBlock()
	require.Nil(t, err)
	require.Equal(t, uint64Ptr(4), lastProcessedBlock)
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
		ID:      1,
		Address: common.HexToAddress("0xaa01"),
		Amount:  big.NewInt(1),
		ConfirmationInfo: db.ConfirmationInfo{
			TxnHash:        common.HexToHash("0xff01"),
			BlockNumber:    1,
			BlockTimestamp: testTimestamp,
		},
	}
	deposit2 := db.Deposit{
		ID:      2,
		Address: common.HexToAddress("0xaa21"),
		Amount:  big.NewInt(2),
		ConfirmationInfo: db.ConfirmationInfo{
			TxnHash:        common.HexToHash("0xff21"),
			BlockNumber:    2,
			BlockTimestamp: testTimestamp,
		},
	}
	deposit3 := db.Deposit{
		ID:      3,
		Address: common.HexToAddress("0xaa22"),
		Amount:  big.NewInt(2),
		ConfirmationInfo: db.ConfirmationInfo{
			TxnHash:        common.HexToHash("0xff22"),
			BlockNumber:    2,
			BlockTimestamp: testTimestamp,
		},
	}

	err = d.UpsertDeposits([]db.Deposit{
		deposit1, deposit2, deposit3,
	}, 0)
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

	err = d.UpsertDisbursement(deposit1.ID, common.HexToHash("0xdd01"), 1, testTimestamp, true)
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
	depBlockNumber := uint64(1)
	disTxnHash := common.HexToHash("0xee02")
	disBlockNumber := uint64(2)

	// Calling UpsertDisbursement with the zero timestamp should fail.
	err := d.UpsertDisbursement(0, common.HexToHash("0xdd00"), 0, time.Time{}, true)
	require.Equal(t, db.ErrZeroTimestamp, err)

	// Calling UpsertDisbursement with an unknown id should fail.
	err = d.UpsertDisbursement(0, common.HexToHash("0xdd00"), 0, testTimestamp, true)
	require.Equal(t, db.ErrUnknownDeposit, err)

	// Now, insert a real deposit that we will disburse.
	err = d.UpsertDeposits([]db.Deposit{
		{
			ID:      1,
			Address: address,
			Amount:  amount,
			ConfirmationInfo: db.ConfirmationInfo{
				TxnHash:        depTxnHash,
				BlockNumber:    depBlockNumber,
				BlockTimestamp: testTimestamp,
			},
		},
	}, 0)
	require.Nil(t, err)

	// Mark the deposit as disbursed with some temporary info.
	tempDisTxnHash := common.HexToHash("0xee00")
	tempDisBlockNumber := uint64(1)
	err = d.UpsertDisbursement(
		1, tempDisTxnHash, tempDisBlockNumber, testTimestamp, false,
	)
	require.Nil(t, err)

	expTeleports := []db.Teleport{
		{
			Deposit: db.Deposit{
				ID:      1,
				Address: address,
				Amount:  amount,
				ConfirmationInfo: db.ConfirmationInfo{
					TxnHash:        depTxnHash,
					BlockNumber:    depBlockNumber,
					BlockTimestamp: testTimestamp,
				},
			},
			Disbursement: &db.Disbursement{
				Success: false,
				ConfirmationInfo: db.ConfirmationInfo{
					TxnHash:        tempDisTxnHash,
					BlockNumber:    tempDisBlockNumber,
					BlockTimestamp: testTimestamp,
				},
			},
		},
	}

	// Assert that the deposit shows up in the CompletedTeleports method with
	// both the L1 and temp L2 confirmation info.
	teleports, err := d.CompletedTeleports()
	require.Nil(t, err)
	require.Equal(t, expTeleports, teleports)

	// Overwrite the disbursement info with the final values.
	err = d.UpsertDisbursement(1, disTxnHash, disBlockNumber, testTimestamp, true)
	require.Nil(t, err)

	expTeleports = []db.Teleport{
		{
			Deposit: db.Deposit{
				ID:      1,
				Address: address,
				Amount:  amount,
				ConfirmationInfo: db.ConfirmationInfo{
					TxnHash:        depTxnHash,
					BlockNumber:    depBlockNumber,
					BlockTimestamp: testTimestamp,
				},
			},
			Disbursement: &db.Disbursement{
				Success: true,
				ConfirmationInfo: db.ConfirmationInfo{
					TxnHash:        disTxnHash,
					BlockNumber:    disBlockNumber,
					BlockTimestamp: testTimestamp,
				},
			},
		},
	}

	// Assert that the deposit now shows up in the CompletedTeleports method
	// with both the L1 and L2 confirmation info.
	teleports, err = d.CompletedTeleports()
	require.Nil(t, err)
	require.Equal(t, expTeleports, teleports)
}

// TestUpsertPendingTxs asserts that UpsertPendingTx properly records a pending
// tx, and that it appears in ListPendingTxs on subsequent calls.
func TestUpsertPendingTxs(t *testing.T) {
	t.Parallel()

	d := newDatabase(t)
	defer d.Close()

	// Should be empty at first.
	pendingTxs, err := d.ListPendingTxs()
	require.Nil(t, err)
	require.Nil(t, pendingTxs)

	// Add first pending tx.
	pendingTx1 := db.PendingTx{
		TxHash:  common.HexToHash("0x11"),
		StartID: 0,
		EndID:   1,
	}
	err = d.UpsertPendingTx(pendingTx1)
	require.Nil(t, err)

	pendingTxs, err = d.ListPendingTxs()
	require.Nil(t, err)
	require.Equal(t, []db.PendingTx{pendingTx1}, pendingTxs)

	// Add second pending tx.
	pendingTx2 := db.PendingTx{
		TxHash:  common.HexToHash("0x22"),
		StartID: 0,
		EndID:   1,
	}
	err = d.UpsertPendingTx(pendingTx2)
	require.Nil(t, err)

	pendingTxs, err = d.ListPendingTxs()
	require.Nil(t, err)
	require.Equal(t, []db.PendingTx{pendingTx1, pendingTx2}, pendingTxs)

	// Readd duplciate pending tx.
	err = d.UpsertPendingTx(pendingTx2)
	require.Nil(t, err)

	pendingTxs, err = d.ListPendingTxs()
	require.Nil(t, err)
	require.Equal(t, []db.PendingTx{pendingTx1, pendingTx2}, pendingTxs)

	// Add third pending tx.
	pendingTx3 := db.PendingTx{
		TxHash:  common.HexToHash("0x33"),
		StartID: 1,
		EndID:   2,
	}
	err = d.UpsertPendingTx(pendingTx3)
	require.Nil(t, err)

	pendingTxs, err = d.ListPendingTxs()
	require.Nil(t, err)
	require.Equal(t, []db.PendingTx{pendingTx3, pendingTx1, pendingTx2}, pendingTxs)
}

// TestDeletePendingTx asserts that DeletePendingTx properly cleans up the
// pending_txs table when provided with various start/end ids.
func TestDeletePendingTx(t *testing.T) {
	t.Parallel()

	d := newDatabase(t)
	defer d.Close()

	pendingTx1 := db.PendingTx{
		TxHash:  common.HexToHash("0x11"),
		StartID: 0,
		EndID:   1,
	}
	pendingTx2 := db.PendingTx{
		TxHash:  common.HexToHash("0x22"),
		StartID: 0,
		EndID:   1,
	}
	pendingTx3 := db.PendingTx{
		TxHash:  common.HexToHash("0x33"),
		StartID: 1,
		EndID:   2,
	}

	err := d.UpsertPendingTx(pendingTx1)
	require.Nil(t, err)
	err = d.UpsertPendingTx(pendingTx2)
	require.Nil(t, err)
	err = d.UpsertPendingTx(pendingTx3)
	require.Nil(t, err)

	pendingTxs, err := d.ListPendingTxs()
	require.Nil(t, err)
	require.Equal(t, []db.PendingTx{pendingTx3, pendingTx1, pendingTx2}, pendingTxs)

	// Delete with indexes that do not match any start/end, no effect.
	err = d.DeletePendingTx(3, 4)
	require.Nil(t, err)
	pendingTxs, err = d.ListPendingTxs()
	require.Nil(t, err)
	require.Equal(t, []db.PendingTx{pendingTx3, pendingTx1, pendingTx2}, pendingTxs)

	// Delete with indexes that matches start but no end, no effect.
	err = d.DeletePendingTx(1, 3)
	require.Nil(t, err)
	pendingTxs, err = d.ListPendingTxs()
	require.Nil(t, err)
	require.Equal(t, []db.PendingTx{pendingTx3, pendingTx1, pendingTx2}, pendingTxs)

	// Delete with indexes that matches end but no start, no effect.
	err = d.DeletePendingTx(0, 2)
	require.Nil(t, err)
	pendingTxs, err = d.ListPendingTxs()
	require.Nil(t, err)
	require.Equal(t, []db.PendingTx{pendingTx3, pendingTx1, pendingTx2}, pendingTxs)

	// Delete with indexes that matches start and end, should remove both.
	err = d.DeletePendingTx(0, 1)
	require.Nil(t, err)
	pendingTxs, err = d.ListPendingTxs()
	require.Nil(t, err)
	require.Equal(t, []db.PendingTx{pendingTx3}, pendingTxs)

	// Delete with indexes that matches start and end, no empty.
	err = d.DeletePendingTx(1, 2)
	require.Nil(t, err)
	pendingTxs, err = d.ListPendingTxs()
	require.Nil(t, err)
	require.Nil(t, pendingTxs)
}

// TestLoadTeleports asserts that LoadTeleportByDepositHash and
// LoadTeleportsByAddress are able to query for a spcific deposit in various
// stages through the teleport process.
func TestLoadTeleports(t *testing.T) {
	t.Parallel()

	d := newDatabase(t)
	defer d.Close()

	address := common.HexToAddress("0x01")
	amount := big.NewInt(1000)
	depTxnHash := common.HexToHash("0x0d01")
	depBlockNumber := uint64(1)
	disTxnHash := common.HexToHash("0x0e01")
	disBlockNumber := uint64(2)

	// Insert deposit.
	deposit1 := db.Deposit{
		ID:      1,
		Address: address,
		Amount:  amount,
		ConfirmationInfo: db.ConfirmationInfo{
			TxnHash:        depTxnHash,
			BlockNumber:    depBlockNumber,
			BlockTimestamp: testTimestamp,
		},
	}

	err := d.UpsertDeposits([]db.Deposit{deposit1}, 0)
	require.Nil(t, err)

	// The same, undisbursed teleport should be retruned by hash and address.
	expTeleport := db.Teleport{
		Deposit:      deposit1,
		Disbursement: nil,
	}

	teleport, err := d.LoadTeleportByDepositHash(depTxnHash)
	require.Nil(t, err)
	require.NotNil(t, teleport)
	require.Equal(t, expTeleport, *teleport)

	teleports, err := d.LoadTeleportsByAddress(address)
	require.Nil(t, err)
	require.Equal(t, []db.Teleport{expTeleport}, teleports)

	// Insert a disbursement for the above deposit.
	err = d.UpsertDisbursement(
		1, disTxnHash, disBlockNumber, testTimestamp, true,
	)
	require.Nil(t, err)

	// The now-complete teleport should be returned from both queries.
	expTeleport = db.Teleport{
		Deposit: deposit1,
		Disbursement: &db.Disbursement{
			Success: true,
			ConfirmationInfo: db.ConfirmationInfo{
				TxnHash:        disTxnHash,
				BlockNumber:    disBlockNumber,
				BlockTimestamp: testTimestamp,
			},
		},
	}

	teleport, err = d.LoadTeleportByDepositHash(depTxnHash)
	require.Nil(t, err)
	require.NotNil(t, teleport)
	require.Equal(t, expTeleport, *teleport)

	teleports, err = d.LoadTeleportsByAddress(address)
	require.Nil(t, err)
	require.Equal(t, []db.Teleport{expTeleport}, teleports)
}
