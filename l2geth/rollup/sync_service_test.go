package rollup

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/l2geth/common"
	"github.com/ethereum-optimism/optimism/l2geth/consensus/ethash"
	"github.com/ethereum-optimism/optimism/l2geth/core"
	"github.com/ethereum-optimism/optimism/l2geth/core/rawdb"
	"github.com/ethereum-optimism/optimism/l2geth/core/types"
	"github.com/ethereum-optimism/optimism/l2geth/core/vm"
	"github.com/ethereum-optimism/optimism/l2geth/crypto"
	"github.com/ethereum-optimism/optimism/l2geth/eth/gasprice"
	"github.com/ethereum-optimism/optimism/l2geth/ethdb"
	"github.com/ethereum-optimism/optimism/l2geth/event"
	"github.com/ethereum-optimism/optimism/l2geth/params"
	"github.com/ethereum-optimism/optimism/l2geth/rollup/rcfg"
)

// Test that the timestamps are updated correctly.
// This impacts execution, for `block.timestamp`
func TestSyncServiceTimestampUpdate(t *testing.T) {
	service, txCh, _, err := newTestSyncService(false, nil)
	if err != nil {
		t.Fatal(err)
	}

	// Get the timestamp from the sync service
	// It should be initialized to 0
	ts := service.GetLatestL1Timestamp()
	if ts != 0 {
		t.Fatalf("Unexpected timestamp: %d", ts)
	}

	// Create a mock transaction and assert that its timestamp
	// a value. This tests the case that the timestamp does
	// not get malleated when it is set to a non zero value
	timestamp := uint64(1)
	tx1 := setMockTxL1Timestamp(mockTx(), timestamp)
	if tx1.GetMeta().L1Timestamp != timestamp {
		t.Fatalf("Expecting mock timestamp to be %d", timestamp)
	}
	if tx1.GetMeta().QueueOrigin != types.QueueOriginSequencer {
		t.Fatalf("Expecting mock queue origin to be queue origin sequencer")
	}

	go func() {
		err = service.applyTransactionToTip(tx1)
	}()
	event1 := <-txCh

	// Ensure that the timestamp isn't malleated
	if event1.Txs[0].GetMeta().L1Timestamp != timestamp {
		t.Fatalf("Timestamp was malleated: %d", event1.Txs[0].GetMeta().L1Timestamp)
	}
	// Ensure that the timestamp in the sync service was updated
	if service.GetLatestL1Timestamp() != timestamp {
		t.Fatal("timestamp updated in sync service")
	}

	// Now test the case for when a transaction is malleated.
	// If the timestamp is 0, then it should be malleated and set
	// equal to whatever the latestL1Timestamp is
	tx2 := mockTx()
	if tx2.GetMeta().L1Timestamp != 0 {
		t.Fatal("Expecting mock timestamp to be 0")
	}
	go func() {
		err = service.applyTransactionToTip(tx2)
	}()
	event2 := <-txCh

	// Ensure that the sync service timestamp is updated
	if service.GetLatestL1Timestamp() == 0 {
		t.Fatal("timestamp not updated")
	}
	// Ensure that the timestamp is malleated to be equal to what the sync
	// service has as the latest timestamp
	if event2.Txs[0].GetMeta().L1Timestamp != service.GetLatestL1Timestamp() {
		t.Fatal("unexpected timestamp update")
	}

	// L1ToL2 transactions should have their timestamp malleated
	// Be sure to set the timestamp to a non zero value so that
	// its specifically testing the fact its a deposit tx
	tx3 := setMockQueueOrigin(setMockTxL1Timestamp(mockTx(), 100), types.QueueOriginL1ToL2)
	// Get a reference to the timestamp before transaction execution
	ts3 := service.GetLatestL1Timestamp()

	go func() {
		err = service.applyTransactionToTip(tx3)
	}()
	event3 := <-txCh

	if event3.Txs[0].GetMeta().L1Timestamp != ts3 {
		t.Fatal("bad malleation")
	}
	// Ensure that the timestamp didn't change
	if ts3 != service.GetLatestL1Timestamp() {
		t.Fatal("timestamp updated when it shouldn't have")
	}
}

// Test that the L1 blocknumber is updated correctly
func TestSyncServiceL1BlockNumberUpdate(t *testing.T) {
	service, txCh, _, err := newTestSyncService(false, nil)
	if err != nil {
		t.Fatal(err)
	}

	// Get the L1 blocknumber from the sync service
	// It should be initialized to 0
	bn := service.GetLatestL1BlockNumber()
	if bn != 0 {
		t.Fatalf("Unexpected timestamp: %d", bn)
	}

	tx1 := setMockTxL1BlockNumber(mockTx(), new(big.Int).SetUint64(1))
	go func() {
		err = service.applyTransactionToTip(tx1)
	}()
	event1 := <-txCh

	// Ensure that the L1 blocknumber was not
	// malleated
	if event1.Txs[0].L1BlockNumber().Uint64() != 1 {
		t.Fatal("wrong l1 blocknumber")
	}

	// Ensure that the latest L1 blocknumber was
	// updated
	if service.GetLatestL1BlockNumber() != 1 {
		t.Fatal("sync service latest l1 blocknumber not updated")
	}

	// Ensure that a tx without a L1 blocknumber gets one
	// assigned
	tx2 := setMockTxL1BlockNumber(mockTx(), nil)
	if tx2.L1BlockNumber() != nil {
		t.Fatal("non nil l1 blocknumber")
	}
	go func() {
		err = service.applyTransactionToTip(tx2)
	}()
	event2 := <-txCh

	if event2.Txs[0].L1BlockNumber() == nil {
		t.Fatal("tx not assigned an l1 blocknumber")
	}
	if event2.Txs[0].L1BlockNumber().Uint64() != service.GetLatestL1BlockNumber() {
		t.Fatal("tx assigned incorrect l1 blocknumber")
	}

	// Ensure that the latest L1 blocknumber doesn't go backwards
	latest := service.GetLatestL1BlockNumber()
	tx3 := setMockTxL1BlockNumber(mockTx(), new(big.Int).SetUint64(latest-1))
	go func() {
		err = service.applyTransactionToTip(tx3)
	}()
	event3 := <-txCh
	if service.GetLatestL1BlockNumber() != latest {
		t.Fatal("block number went backwards")
	}

	if event3.Txs[0].L1BlockNumber().Uint64() != latest-1 {
		t.Fatal("l1 block number was malleated")
	}
}

// Test that the `RollupTransaction` ends up in the transaction cache
// after the transaction enqueued event is emitted. Set `false` as
// the argument to start as a sequencer
func TestSyncServiceTransactionEnqueued(t *testing.T) {
	service, txCh, _, err := newTestSyncService(false, nil)
	if err != nil {
		t.Fatal(err)
	}

	// The timestamp is in the rollup transaction
	timestamp := uint64(24)
	// The target is the `to` field on the transaction
	target := common.HexToAddress("0x04668ec2f57cc15c381b461b9fedab5d451c8f7f")
	// The layer one transaction origin is in the txmeta on the transaction
	l1TxOrigin := common.HexToAddress("0xEA674fdDe714fd979de3EdF0F56AA9716B898ec8")
	// The gasLimit is the `gasLimit` on the transaction
	gasLimit := uint64(66)
	// The data is the `data` on the transaction
	data := []byte{0x02, 0x92}
	// The L1 blocknumber for the transaction's evm context
	l1BlockNumber := big.NewInt(100)
	// The queue index of the L1 to L2 transaction
	queueIndex := uint64(0)
	// The index in the ctc
	index := uint64(0)

	tx := types.NewTransaction(0, target, big.NewInt(0), gasLimit, big.NewInt(0), data)
	txMeta := types.NewTransactionMeta(
		l1BlockNumber,
		timestamp,
		&l1TxOrigin,
		types.QueueOriginL1ToL2,
		&index,
		&queueIndex,
		nil,
	)
	tx.SetTransactionMeta(txMeta)

	setupMockClient(service, map[string]interface{}{
		"GetEnqueue": []*types.Transaction{
			tx,
		},
	})

	// Run an iteration of the eloop
	err = nil
	go func() {
		err = service.syncQueueToTip()
	}()
	// Wait for the tx to be confirmed into the chain and then
	// make sure it is the transactions that was set up with in the mockclient
	event := <-txCh
	if err != nil {
		t.Fatal("sequencing failed", err)
	}
	if len(event.Txs) != 1 {
		t.Fatal("Unexpected number of transactions")
	}
	confirmed := event.Txs[0]

	if !reflect.DeepEqual(tx, confirmed) {
		t.Fatal("different txs")
	}
}

func TestTransactionToTipNoIndex(t *testing.T) {
	service, txCh, _, err := newTestSyncService(false, nil)
	if err != nil {
		t.Fatal(err)
	}

	// Get a reference to the current next index to compare with the index that
	// is set to the transaction that is ingested
	nextIndex := service.GetNextIndex()

	timestamp := uint64(24)
	target := common.HexToAddress("0x04668ec2f57cc15c381b461b9fedab5d451c8f7f")
	l1TxOrigin := common.HexToAddress("0xEA674fdDe714fd979de3EdF0F56AA9716B898ec8")
	gasLimit := uint64(66)
	data := []byte{0x02, 0x92}
	l1BlockNumber := big.NewInt(100)

	tx := types.NewTransaction(0, target, big.NewInt(0), gasLimit, big.NewInt(0), data)
	meta := types.NewTransactionMeta(
		l1BlockNumber,
		timestamp,
		&l1TxOrigin,
		types.QueueOriginL1ToL2,
		nil, // The index is `nil`, expect it to be set afterwards
		nil,
		nil,
	)
	tx.SetTransactionMeta(meta)

	go func() {
		err = service.applyTransactionToTip(tx)
	}()
	event := <-txCh
	if err != nil {
		t.Fatal("Cannot apply transaction to the tip")
	}
	confirmed := event.Txs[0]
	// The transaction was applied without an index so the chain gave it the
	// next index
	index := confirmed.GetMeta().Index
	if index == nil {
		t.Fatal("Did not set index after applying tx to tip")
	}
	if *index != *service.GetLatestIndex() {
		t.Fatal("Incorrect latest index")
	}
	if *index != nextIndex {
		t.Fatal("Incorrect index")
	}
}

func TestTransactionToTipTimestamps(t *testing.T) {
	service, txCh, _, err := newTestSyncService(false, nil)
	if err != nil {
		t.Fatal(err)
	}

	// Create two mock transactions with `nil` indices. This will allow
	// assertions around the indices being updated correctly. Set the timestamp
	// to 1 and 2 and assert that the timestamps in the sync service are updated
	// correctly
	tx1 := setMockTxL1Timestamp(mockTx(), 1)
	tx2 := setMockTxL1Timestamp(mockTx(), 2)

	txs := []*types.Transaction{
		tx1,
		tx2,
	}

	for _, tx := range txs {
		nextIndex := service.GetNextIndex()

		go func() {
			err = service.applyTransactionToTip(tx)
		}()
		event := <-txCh
		if err != nil {
			t.Fatal(err)
		}

		conf := event.Txs[0]
		// The index should be set to the next
		if conf.GetMeta().Index == nil {
			t.Fatal("Index is nil")
		}
		// The index that the sync service is tracking should be updated
		if *conf.GetMeta().Index != *service.GetLatestIndex() {
			t.Fatal("index on the service was not updated")
		}
		// The indexes should be incrementing by 1
		if *conf.GetMeta().Index != nextIndex {
			t.Fatalf("Mismatched index: got %d, expect %d", *conf.GetMeta().Index, nextIndex)
		}
		// The tx timestamp should be setting the services timestamp
		if conf.L1Timestamp() != service.GetLatestL1Timestamp() {
			t.Fatal("Mismatched timestamp")
		}
	}

	// Ensure that the timestamp was updated correctly
	ts := service.GetLatestL1Timestamp()
	if ts != tx2.L1Timestamp() {
		t.Fatal("timestamp not updated correctly")
	}

	// Send a transaction with no timestamp and then let it be updated
	// by the sync service. This will prevent monotonicity errors as well.
	// as give timestamps to queue origin sequencer transactions
	// Ensure that the timestamp is set to `time.Now`
	// when it is not set.
	tx3 := setMockTxL1Timestamp(mockTx(), 0)
	now := time.Now()
	go func() {
		err = service.applyTransactionToTip(tx3)
	}()
	result := <-txCh
	service.chainHeadCh <- core.ChainHeadEvent{}

	if result.Txs[0].L1Timestamp() != uint64(now.Unix()) {
		t.Fatal("Timestamp not updated correctly")
	}

	if service.GetLatestL1Timestamp() != uint64(now.Unix()) {
		t.Fatal("latest timestamp not updated correctly")
	}
}

func TestApplyIndexedTransaction(t *testing.T) {
	service, txCh, _, err := newTestSyncService(true, nil)
	if err != nil {
		t.Fatal(err)
	}

	// Create three transactions, two of which have a duplicate index.
	// The first two transactions can be ingested without a problem and the
	// third transaction has a duplicate index so it will not be ingested.
	// Expect an error for the third transaction and expect the SyncService
	// global index to be updated with the first two transactions
	tx0 := setMockTxIndex(mockTx(), 0)
	tx1 := setMockTxIndex(mockTx(), 1)
	tx1a := setMockTxIndex(mockTx(), 1)

	go func() {
		err = service.applyIndexedTransaction(tx0)
	}()
	<-txCh
	if err != nil {
		t.Fatal(err)
	}
	if *tx0.GetMeta().Index != *service.GetLatestIndex() {
		t.Fatal("Latest index mismatch")
	}

	go func() {
		err = service.applyIndexedTransaction(tx1)
	}()
	<-txCh
	if err != nil {
		t.Fatal(err)
	}
	if *tx1.GetMeta().Index != *service.GetLatestIndex() {
		t.Fatal("Latest index mismatch")
	}

	err = service.applyIndexedTransaction(tx1a)
	if err == nil {
		t.Fatal(err)
	}
}

func TestApplyBatchedTransaction(t *testing.T) {
	service, txCh, _, err := newTestSyncService(true, nil)
	if err != nil {
		t.Fatal(err)
	}

	// Create a transactoin with the index of 0
	tx0 := setMockTxIndex(mockTx(), 0)

	// Ingest through applyBatchedTransaction which should set the latest
	// verified index to the index of the transaction
	go func() {
		err = service.applyBatchedTransaction(tx0)
	}()
	service.chainHeadCh <- core.ChainHeadEvent{}
	<-txCh

	// Catch race conditions with the database write
	wg := new(sync.WaitGroup)
	wg.Add(1)
	go func() {
		for {
			if service.GetLatestVerifiedIndex() != nil {
				wg.Done()
				return
			}
			time.Sleep(100 * time.Millisecond)
		}
	}()
	wg.Wait()

	// Assert that the verified index is the same as the transaction index
	if *tx0.GetMeta().Index != *service.GetLatestVerifiedIndex() {
		t.Fatal("Latest verified index mismatch")
	}
}

func TestIsAtTip(t *testing.T) {
	service, _, _, err := newTestSyncService(true, nil)
	if err != nil {
		t.Fatal(err)
	}

	data := []struct {
		tip    *uint64
		get    indexGetter
		expect bool
		err    error
	}{
		{
			tip: newUint64(1),
			get: func() (*uint64, error) {
				return newUint64(1), nil
			},
			expect: true,
			err:    nil,
		},
		{
			tip: newUint64(0),
			get: func() (*uint64, error) {
				return newUint64(1), nil
			},
			expect: false,
			err:    nil,
		},
		{
			tip: newUint64(1),
			get: func() (*uint64, error) {
				return newUint64(0), nil
			},
			expect: false,
			err:    errShortRemoteTip,
		},
		{
			tip: nil,
			get: func() (*uint64, error) {
				return nil, nil
			},
			expect: true,
			err:    nil,
		},
		{
			tip: nil,
			get: func() (*uint64, error) {
				return nil, errElementNotFound
			},
			expect: true,
			err:    nil,
		},
		{
			tip: newUint64(0),
			get: func() (*uint64, error) {
				return nil, errElementNotFound
			},
			expect: false,
			err:    nil,
		},
	}

	for _, d := range data {
		isAtTip, err := service.isAtTip(d.tip, d.get)
		if isAtTip != d.expect {
			t.Fatal("expected does not match")
		}
		if !errors.Is(err, d.err) {
			t.Fatal("error no match")
		}
	}
}

func TestSyncQueue(t *testing.T) {
	service, txCh, _, err := newTestSyncService(true, nil)
	if err != nil {
		t.Fatal(err)
	}

	setupMockClient(service, map[string]interface{}{
		"GetEnqueue": []*types.Transaction{
			setMockQueueIndex(mockTx(), 0),
			setMockQueueIndex(mockTx(), 1),
			setMockQueueIndex(mockTx(), 2),
			setMockQueueIndex(mockTx(), 3),
		},
	})

	var tip *uint64
	go func() {
		tip, err = service.syncQueue()
	}()

	for i := 0; i < 4; i++ {
		service.chainHeadCh <- core.ChainHeadEvent{}
		event := <-txCh
		tx := event.Txs[0]
		if *tx.GetMeta().QueueIndex != uint64(i) {
			t.Fatal("queue index mismatch")
		}
	}

	wg := new(sync.WaitGroup)
	wg.Add(1)
	go func() {
		for {
			if tip != nil {
				wg.Done()
				return
			}
			time.Sleep(100 * time.Millisecond)
		}
	}()
	wg.Wait()
	if tip == nil {
		t.Fatal("tip is nil")
	}
	// There were a total of 4 transactions synced and the indexing starts at 0
	if *service.GetLatestIndex() != 3 {
		t.Fatalf("Latest index mismatch")
	}
	// All of the transactions are `enqueue()`s
	if *service.GetLatestEnqueueIndex() != 3 {
		t.Fatal("Latest queue index mismatch")
	}
	if *tip != 3 {
		t.Fatal("Tip mismatch")
	}
}

func TestSyncServiceL1GasPrice(t *testing.T) {
	service, _, _, err := newTestSyncService(true, nil)
	setupMockClient(service, map[string]interface{}{})

	if err != nil {
		t.Fatal(err)
	}

	gasBefore, err := service.RollupGpo.SuggestL1GasPrice(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	if gasBefore.Cmp(big.NewInt(0)) != 0 {
		t.Fatal("expected 0 gas price, got", gasBefore)
	}

	state, err := service.bc.State()
	if err != nil {
		t.Fatal("Cannot get state db")
	}
	l1GasPrice := big.NewInt(100000000000)
	state.SetState(rcfg.L2GasPriceOracleAddress, rcfg.L1GasPriceSlot, common.BigToHash(l1GasPrice))
	_, _ = state.Commit(false)

	// Update the gas price
	service.updateL1GasPrice(state)

	gasAfter, err := service.RollupGpo.SuggestL1GasPrice(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	if gasAfter.Cmp(l1GasPrice) != 0 {
		t.Fatal("expected 100 gas price, got", gasAfter)
	}
}

func TestSyncServiceL2GasPrice(t *testing.T) {
	service, _, _, err := newTestSyncService(true, nil)
	if err != nil {
		t.Fatal(err)
	}

	price, err := service.RollupGpo.SuggestL2GasPrice(context.Background())
	if err != nil {
		t.Fatal("Cannot fetch execution price")
	}

	if price.Cmp(common.Big0) != 0 {
		t.Fatal("Incorrect gas price")
	}

	state, err := service.bc.State()
	if err != nil {
		t.Fatal("Cannot get state db")
	}
	l2GasPrice := big.NewInt(100000000000)
	state.SetState(rcfg.L2GasPriceOracleAddress, rcfg.L2GasPriceSlot, common.BigToHash(l2GasPrice))
	_, _ = state.Commit(false)

	service.updateL2GasPrice(state)

	post, err := service.RollupGpo.SuggestL2GasPrice(context.Background())
	if err != nil {
		t.Fatal("Cannot fetch execution price")
	}

	if l2GasPrice.Cmp(post) != 0 {
		t.Fatal("Gas price not updated")
	}
}

func TestSyncServiceGasPriceOracleOwnerAddress(t *testing.T) {
	service, _, _, err := newTestSyncService(true, nil)
	if err != nil {
		t.Fatal(err)
	}
	// newTestSyncService doesn't set the initial owner address
	// so it initializes to the zero value
	owner := service.GasPriceOracleOwnerAddress()
	if *owner != (common.Address{}) {
		t.Fatal("address not initialized to 0")
	}

	state, err := service.bc.State()
	if err != nil {
		t.Fatal("cannot get state db")
	}

	// Update the owner in the state to a non zero address
	updatedOwner := common.HexToAddress("0xEA674fdDe714fd979de3EdF0F56AA9716B898ec8")
	state.SetState(rcfg.L2GasPriceOracleAddress, rcfg.L2GasPriceOracleOwnerSlot, updatedOwner.Hash())
	hash, _ := state.Commit(false)

	// Update the cache based on the latest state root
	if err := service.updateGasPriceOracleCache(&hash); err != nil {
		t.Fatal(err)
	}
	got := service.GasPriceOracleOwnerAddress()
	if *got != updatedOwner {
		t.Fatalf("mismatch:\ngot %s\nexpected %s", got.Hex(), updatedOwner.Hex())
	}
}

// Only the gas price oracle owner can send 0 gas price txs
// when fees are enforced
func TestFeeGasPriceOracleOwnerTransactions(t *testing.T) {
	service, _, _, err := newTestSyncService(true, nil)
	if err != nil {
		t.Fatal(err)
	}
	signer := types.NewEIP155Signer(big.NewInt(420))

	// Fees must be enforced for this test
	service.enforceFees = true
	// Generate a key
	key, _ := crypto.GenerateKey()
	owner := crypto.PubkeyToAddress(key.PublicKey)
	// Set as the owner on the SyncService
	service.gasPriceOracleOwnerAddress = owner
	if owner != *service.GasPriceOracleOwnerAddress() {
		t.Fatal("owner mismatch")
	}
	// Create a mock transaction and sign using the
	// owner's key
	tx := mockTx()
	// Make sure the gas price is 0 on the dummy tx
	if tx.GasPrice().Cmp(common.Big0) != 0 {
		t.Fatal("gas price not 0")
	}
	// Sign the dummy tx with the owner key
	signedTx, err := types.SignTx(tx, signer, key)
	if err != nil {
		t.Fatal(err)
	}
	// Verify the fee of the signed tx, ensure it does not error
	if err := service.verifyFee(signedTx); err != nil {
		t.Fatal(err)
	}
	// Generate a new random key that is not the owner
	badKey, _ := crypto.GenerateKey()
	// Ensure that it is not the owner
	if owner == crypto.PubkeyToAddress(badKey.PublicKey) {
		t.Fatal("key mismatch")
	}
	// Sign the transaction with the bad key
	badSignedTx, err := types.SignTx(tx, signer, badKey)
	if err != nil {
		t.Fatal(err)
	}
	// Attempt to verify the fee of the bad tx
	// It should error and be a errZeroGasPriceTx
	if err := service.verifyFee(badSignedTx); err != nil {
		if !errors.Is(errZeroGasPriceTx, err) {
			t.Fatal(err)
		}
	} else {
		t.Fatal("err is nil")
	}
}

// Pass true to set as a verifier
func TestSyncServiceSync(t *testing.T) {
	service, txCh, sub, err := newTestSyncService(true, nil)
	defer sub.Unsubscribe()
	if err != nil {
		t.Fatal(err)
	}

	timestamp := uint64(24)
	target := common.HexToAddress("0x04668ec2f57cc15c381b461b9fedab5d451c8f7f")
	l1TxOrigin := common.HexToAddress("0xEA674fdDe714fd979de3EdF0F56AA9716B898ec8")
	gasLimit := uint64(66)
	data := []byte{0x02, 0x92}
	l1BlockNumber := big.NewInt(100)
	queueIndex := uint64(0)
	index := uint64(0)
	tx := types.NewTransaction(0, target, big.NewInt(0), gasLimit, big.NewInt(0), data)
	txMeta := types.NewTransactionMeta(
		l1BlockNumber,
		timestamp,
		&l1TxOrigin,
		types.QueueOriginL1ToL2,
		&index,
		&queueIndex,
		nil,
	)
	tx.SetTransactionMeta(txMeta)

	setupMockClient(service, map[string]interface{}{
		"GetTransaction": []*types.Transaction{
			tx,
		},
	})

	err = nil
	go func() {
		err = service.syncTransactionsToTip()
	}()
	event := <-txCh
	if err != nil {
		t.Fatal("verification failed", err)
	}

	if len(event.Txs) != 1 {
		t.Fatal("Unexpected number of transactions")
	}
	confirmed := event.Txs[0]

	if !reflect.DeepEqual(tx, confirmed) {
		t.Fatal("different txs")
	}
}

func TestInitializeL1ContextPostGenesis(t *testing.T) {
	service, _, _, err := newTestSyncService(true, nil)
	if err != nil {
		t.Fatal(err)
	}

	timestamp := uint64(24)
	target := common.HexToAddress("0x04668ec2f57cc15c381b461b9fedab5d451c8f7f")
	l1TxOrigin := common.HexToAddress("0xEA674fdDe714fd979de3EdF0F56AA9716B898ec8")
	gasLimit := uint64(66)
	data := []byte{0x02, 0x92}
	l1BlockNumber := big.NewInt(100)
	queueIndex := uint64(100)
	index := uint64(120)
	tx := types.NewTransaction(0, target, big.NewInt(0), gasLimit, big.NewInt(0), data)
	txMeta := types.NewTransactionMeta(
		l1BlockNumber,
		timestamp,
		&l1TxOrigin,
		types.QueueOriginL1ToL2,
		&index,
		&queueIndex,
		nil,
	)
	tx.SetTransactionMeta(txMeta)

	setupMockClient(service, map[string]interface{}{
		"GetEnqueue": []*types.Transaction{
			tx,
		},
		"GetEthContext": []*EthContext{
			{
				BlockNumber: uint64(10),
				BlockHash:   common.Hash{},
				Timestamp:   timestamp,
			},
		},
	})

	header := types.Header{
		Number: big.NewInt(0),
		Time:   11,
	}

	number := uint64(10)
	tx.SetL1Timestamp(timestamp)
	tx.SetL1BlockNumber(number)
	block := types.NewBlock(&header, []*types.Transaction{tx}, []*types.Header{}, []*types.Receipt{})
	service.bc.SetCurrentBlock(block)

	err = service.initializeLatestL1(big.NewInt(0))
	if err != nil {
		t.Fatal(err)
	}

	latestL1Timestamp := service.GetLatestL1Timestamp()
	latestL1BlockNumber := service.GetLatestL1BlockNumber()
	if number != latestL1BlockNumber {
		t.Fatalf("number does not match, got %d, expected %d", latestL1BlockNumber, number)
	}
	if latestL1Timestamp != timestamp {
		t.Fatal("timestamp does not match")
	}
}

func TestBadFeeThresholds(t *testing.T) {
	// Create the deps for the sync service
	cfg, txPool, chain, db, err := newTestSyncServiceDeps(false, nil)
	if err != nil {
		t.Fatal(err)
	}

	tests := map[string]struct {
		thresholdUp   *big.Float
		thresholdDown *big.Float
		err           error
	}{
		"nil-values": {
			thresholdUp:   nil,
			thresholdDown: nil,
			err:           nil,
		},
		"good-values": {
			thresholdUp:   new(big.Float).SetFloat64(2),
			thresholdDown: new(big.Float).SetFloat64(0.8),
			err:           nil,
		},
		"bad-value-up": {
			thresholdUp:   new(big.Float).SetFloat64(0.8),
			thresholdDown: nil,
			err:           errBadConfig,
		},
		"bad-value-down": {
			thresholdUp:   nil,
			thresholdDown: new(big.Float).SetFloat64(1.1),
			err:           errBadConfig,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			cfg.FeeThresholdDown = tt.thresholdDown
			cfg.FeeThresholdUp = tt.thresholdUp

			_, err := NewSyncService(context.Background(), cfg, txPool, chain, db)
			if !errors.Is(err, tt.err) {
				t.Fatalf("%s: %s", name, err)
			}
		})
	}
}

func newTestSyncServiceDeps(isVerifier bool, alloc *common.Address) (Config, *core.TxPool, *core.BlockChain, ethdb.Database, error) {
	chainCfg := params.AllEthashProtocolChanges
	chainID := big.NewInt(420)
	chainCfg.ChainID = chainID

	engine := ethash.NewFaker()
	db := rawdb.NewMemoryDatabase()
	genesis := new(core.Genesis)
	if alloc != nil {
		genesis.Alloc = make(core.GenesisAlloc)
		genesis.Alloc[*alloc] = core.GenesisAccount{
			Balance: new(big.Int).SetUint64(100000000000000),
		}
	}
	_ = genesis.MustCommit(db)
	chain, err := core.NewBlockChain(db, nil, chainCfg, engine, vm.Config{}, nil)
	if err != nil {
		return Config{}, nil, nil, nil, fmt.Errorf("Cannot initialize blockchain: %w", err)
	}
	chaincfg := params.ChainConfig{ChainID: chainID}

	txPool := core.NewTxPool(core.TxPoolConfig{PriceLimit: 0}, &chaincfg, chain)
	cfg := Config{
		CanonicalTransactionChainDeployHeight: big.NewInt(0),
		IsVerifier:                            isVerifier,
		// Set as an empty string as this is a dummy value anyways.
		// The client needs to be mocked with a mockClient
		RollupClientHttp: "",
		Backend:          BackendL2,
	}
	return cfg, txPool, chain, db, nil
}

func newTestSyncService(isVerifier bool, alloc *common.Address) (*SyncService, chan core.NewTxsEvent, event.Subscription, error) {
	cfg, txPool, chain, db, err := newTestSyncServiceDeps(isVerifier, alloc)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("Cannot initialize syncservice: %w", err)
	}
	service, err := NewSyncService(context.Background(), cfg, txPool, chain, db)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("Cannot initialize syncservice: %w", err)
	}

	service.RollupGpo = gasprice.NewRollupOracle()
	txCh := make(chan core.NewTxsEvent, 1)
	sub := service.SubscribeNewTxsEvent(txCh)

	return service, txCh, sub, nil
}

type mockClient struct {
	getEnqueueCallCount            int
	getEnqueue                     []*types.Transaction
	getRawTransactionCallCount     int
	getRawTransaction              []*TransactionResponse
	getTransactionCallCount        int
	getTransaction                 []*types.Transaction
	getEthContextCallCount         int
	getEthContext                  []*EthContext
	getLatestEthContext            *EthContext
	getLatestEnqueueIndex          []func() (*uint64, error)
	getLatestEnqueueIndexCallCount int
}

func setupMockClient(service *SyncService, responses map[string]interface{}) {
	client := newMockClient(responses)
	service.client = client
	service.RollupGpo = gasprice.NewRollupOracle()
}

func newMockClient(responses map[string]interface{}) *mockClient {
	getEnqueueResponses := []*types.Transaction{}
	getRawTransactionResponses := []*TransactionResponse{}
	getTransactionResponses := []*types.Transaction{}
	getEthContextResponses := []*EthContext{}
	getLatestEthContextResponse := &EthContext{}
	getLatestEnqueueIndexResponses := []func() (*uint64, error){}

	enqueue, ok := responses["GetEnqueue"]
	if ok {
		getEnqueueResponses = enqueue.([]*types.Transaction)
	}
	getRawTx, ok := responses["GetRawTransaction"]
	if ok {
		getRawTransactionResponses = getRawTx.([]*TransactionResponse)
	}
	getTx, ok := responses["GetTransaction"]
	if ok {
		getTransactionResponses = getTx.([]*types.Transaction)
	}
	getCtx, ok := responses["GetEthContext"]
	if ok {
		getEthContextResponses = getCtx.([]*EthContext)
	}
	getLatestCtx, ok := responses["GetLatestEthContext"]
	if ok {
		getLatestEthContextResponse = getLatestCtx.(*EthContext)
	}
	getLatestEnqueueIdx, ok := responses["GetLatestEnqueueIndex"]
	if ok {
		getLatestEnqueueIndexResponses = getLatestEnqueueIdx.([]func() (*uint64, error))
	}

	return &mockClient{
		getEnqueue:            getEnqueueResponses,
		getRawTransaction:     getRawTransactionResponses,
		getTransaction:        getTransactionResponses,
		getEthContext:         getEthContextResponses,
		getLatestEthContext:   getLatestEthContextResponse,
		getLatestEnqueueIndex: getLatestEnqueueIndexResponses,
	}
}

func (m *mockClient) GetEnqueue(index uint64) (*types.Transaction, error) {
	if m.getEnqueueCallCount < len(m.getEnqueue) {
		tx := m.getEnqueue[m.getEnqueueCallCount]
		m.getEnqueueCallCount++
		return tx, nil
	}
	return nil, errors.New("")
}

func (m *mockClient) GetLatestEnqueue() (*types.Transaction, error) {
	if len(m.getEnqueue) == 0 {
		return &types.Transaction{}, errors.New("enqueue not found")
	}
	return m.getEnqueue[len(m.getEnqueue)-1], nil
}

func (m *mockClient) GetRawTransaction(index uint64, backend Backend) (*TransactionResponse, error) {
	if m.getRawTransactionCallCount < len(m.getRawTransaction) {
		tx := m.getRawTransaction[m.getRawTransactionCallCount]
		m.getRawTransactionCallCount++
		return tx, nil
	}
	return nil, fmt.Errorf("Cannot get raw transaction: mocks (%d), call count (%d)", len(m.getRawTransaction), m.getRawTransactionCallCount)
}

func (m *mockClient) GetTransaction(index uint64, backend Backend) (*types.Transaction, error) {
	if m.getTransactionCallCount < len(m.getTransaction) {
		tx := m.getTransaction[m.getTransactionCallCount]
		m.getTransactionCallCount++
		return tx, nil
	}
	return nil, fmt.Errorf("Cannot get transaction: mocks (%d), call count (%d)", len(m.getTransaction), m.getTransactionCallCount)
}

func (m *mockClient) GetLatestTransaction(backend Backend) (*types.Transaction, error) {
	if len(m.getTransaction) == 0 {
		return nil, errors.New("No transactions")
	}
	return m.getTransaction[len(m.getTransaction)-1], nil
}

func (m *mockClient) GetEthContext(index uint64) (*EthContext, error) {
	if m.getEthContextCallCount < len(m.getEthContext) {
		ctx := m.getEthContext[m.getEthContextCallCount]
		m.getEthContextCallCount++
		return ctx, nil
	}
	return nil, errors.New("Cannot get eth context")
}

func (m *mockClient) GetLatestEthContext() (*EthContext, error) {
	return m.getLatestEthContext, nil
}

func (m *mockClient) GetLastConfirmedEnqueue() (*types.Transaction, error) {
	return nil, errElementNotFound
}

func (m *mockClient) GetLatestTransactionBatch() (*Batch, []*types.Transaction, error) {
	return nil, nil, nil
}

func (m *mockClient) GetTransactionBatch(index uint64) (*Batch, []*types.Transaction, error) {
	return nil, nil, nil
}

func (m *mockClient) SyncStatus(backend Backend) (*SyncStatus, error) {
	return &SyncStatus{
		Syncing: false,
	}, nil
}

func (m *mockClient) GetLatestEnqueueIndex() (*uint64, error) {
	enqueue, err := m.GetLatestEnqueue()
	if err != nil {
		return nil, err
	}
	if enqueue == nil {
		return nil, errElementNotFound
	}
	return enqueue.GetMeta().QueueIndex, nil
}

func (m *mockClient) GetLatestTransactionBatchIndex() (*uint64, error) {
	return nil, nil
}

func (m *mockClient) GetLatestTransactionIndex(backend Backend) (*uint64, error) {
	tx, err := m.GetLatestTransaction(backend)
	if err != nil {
		return nil, err
	}
	return tx.GetMeta().Index, nil
}

func mockTx() *types.Transaction {
	address := make([]byte, 20)
	rand.Read(address)

	target := common.BytesToAddress(address)
	timestamp := uint64(0)

	rand.Read(address)
	l1TxOrigin := common.BytesToAddress(address)

	gasLimit := uint64(0)
	data := []byte{0x00, 0x00}
	l1BlockNumber := big.NewInt(0)

	tx := types.NewTransaction(0, target, big.NewInt(0), gasLimit, big.NewInt(0), data)
	meta := types.NewTransactionMeta(
		l1BlockNumber,
		timestamp,
		&l1TxOrigin,
		types.QueueOriginSequencer,
		nil,
		nil,
		nil,
	)
	tx.SetTransactionMeta(meta)
	return tx
}

func setMockTxL1Timestamp(tx *types.Transaction, ts uint64) *types.Transaction {
	meta := tx.GetMeta()
	meta.L1Timestamp = ts
	tx.SetTransactionMeta(meta)
	return tx
}

func setMockTxL1BlockNumber(tx *types.Transaction, bn *big.Int) *types.Transaction {
	meta := tx.GetMeta()
	meta.L1BlockNumber = bn
	tx.SetTransactionMeta(meta)
	return tx
}

func setMockTxIndex(tx *types.Transaction, index uint64) *types.Transaction {
	meta := tx.GetMeta()
	meta.Index = &index
	tx.SetTransactionMeta(meta)
	return tx
}

func setMockQueueIndex(tx *types.Transaction, index uint64) *types.Transaction {
	meta := tx.GetMeta()
	meta.QueueIndex = &index
	tx.SetTransactionMeta(meta)
	return tx
}

func setMockQueueOrigin(tx *types.Transaction, qo types.QueueOrigin) *types.Transaction {
	meta := tx.GetMeta()
	meta.QueueOrigin = qo
	tx.SetTransactionMeta(meta)
	return tx
}

func newUint64(n uint64) *uint64 {
	return &n
}
