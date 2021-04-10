package rollup

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"reflect"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/eth/gasprice"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/params"
)

func setupLatestEthContextTest() (*SyncService, *EthContext) {
	service, _, _, _ := newTestSyncService(false)
	resp := &EthContext{
		BlockNumber: uint64(10),
		BlockHash:   common.Hash{},
		Timestamp:   uint64(service.timestampRefreshThreshold.Seconds()) + 1,
	}
	setupMockClient(service, map[string]interface{}{
		"GetLatestEthContext": resp,
	})

	return service, resp
}

// Test that if applying a transaction fails
func TestSyncServiceContextUpdated(t *testing.T) {
	service, resp := setupLatestEthContextTest()

	// should get the expected context
	expectedCtx := &OVMContext{
		blockNumber: 0,
		timestamp:   0,
	}

	if service.OVMContext != *expectedCtx {
		t.Fatal("context was not instantiated to the expected value")
	}

	// run the update context call once
	err := service.updateContext()
	if err != nil {
		t.Fatal(err)
	}

	// should get the expected context
	expectedCtx = &OVMContext{
		blockNumber: resp.BlockNumber,
		timestamp:   resp.Timestamp,
	}

	if service.OVMContext != *expectedCtx {
		t.Fatal("context was not updated to the expected response even though enough time passed")
	}

	// updating the context should be a no-op if time advanced by less than
	// the refresh period
	resp.BlockNumber += 1
	resp.Timestamp += uint64(service.timestampRefreshThreshold.Seconds())
	setupMockClient(service, map[string]interface{}{
		"GetLatestEthContext": resp,
	})

	// call it again
	err = service.updateContext()
	if err != nil {
		t.Fatal(err)
	}

	// should not get the context from the response because it was too soon
	unexpectedCtx := &OVMContext{
		blockNumber: resp.BlockNumber,
		timestamp:   resp.Timestamp,
	}
	if service.OVMContext == *unexpectedCtx {
		t.Fatal("context should not be updated because not enough time passed")
	}
}

// Test that the `RollupTransaction` ends up in the transaction cache
// after the transaction enqueued event is emitted. Set `false` as
// the argument to start as a sequencer
func TestSyncServiceTransactionEnqueued(t *testing.T) {
	service, txCh, _, err := newTestSyncService(false)
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
	index := uint64(5)

	tx := types.NewTransaction(0, target, big.NewInt(0), gasLimit, big.NewInt(0), data)
	txMeta := types.NewTransactionMeta(
		l1BlockNumber,
		timestamp,
		&l1TxOrigin,
		types.SighashEIP155,
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
	err = service.sequence()
	if err != nil {
		t.Fatal("sequencing failed", err)
	}

	// Wait for the tx to be confirmed into the chain and then
	// make sure it is the transactions that was set up with in the mockclient
	event := <-txCh
	if len(event.Txs) != 1 {
		t.Fatal("Unexpected number of transactions")
	}
	confirmed := event.Txs[0]

	if !reflect.DeepEqual(tx, confirmed) {
		t.Fatal("different txs")
	}
}

func TestSyncServiceL1GasPrice(t *testing.T) {
	service, _, _, err := newTestSyncService(true)
	setupMockClient(service, map[string]interface{}{})
	service.L1gpo = gasprice.NewL1Oracle(big.NewInt(0))

	if err != nil {
		t.Fatal(err)
	}

	gasBefore, err := service.L1gpo.SuggestDataPrice(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	if gasBefore.Cmp(big.NewInt(0)) != 0 {
		t.Fatal("expected 0 gas price, got", gasBefore)
	}

	// run 1 iteration of the eloop
	service.sequence()

	gasAfter, err := service.L1gpo.SuggestDataPrice(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	if gasAfter.Cmp(big.NewInt(100*int64(params.GWei))) != 0 {
		t.Fatal("expected 100 gas price, got", gasAfter)
	}
}

// Pass true to set as a verifier
func TestSyncServiceSync(t *testing.T) {
	service, txCh, sub, err := newTestSyncService(true)
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
		types.SighashEIP155,
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

	err = service.verify()
	if err != nil {
		t.Fatal("verification failed", err)
	}

	event := <-txCh
	if len(event.Txs) != 1 {
		t.Fatal("Unexpected number of transactions")
	}
	confirmed := event.Txs[0]

	if !reflect.DeepEqual(tx, confirmed) {
		t.Fatal("different txs")
	}
}

func TestInitializeL1ContextPostGenesis(t *testing.T) {
	service, _, _, err := newTestSyncService(true)
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
		types.SighashEIP155,
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

func newTestSyncService(isVerifier bool) (*SyncService, chan core.NewTxsEvent, event.Subscription, error) {
	chainCfg := params.AllEthashProtocolChanges
	chainID := big.NewInt(420)
	chainCfg.ChainID = chainID

	engine := ethash.NewFaker()
	db := rawdb.NewMemoryDatabase()
	_ = new(core.Genesis).MustCommit(db)
	chain, err := core.NewBlockChain(db, nil, chainCfg, engine, vm.Config{}, nil)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("Cannot initialize blockchain: %w", err)
	}
	chaincfg := params.ChainConfig{ChainID: chainID}

	txPool := core.NewTxPool(core.TxPoolConfig{PriceLimit: 0}, &chaincfg, chain)
	cfg := Config{
		CanonicalTransactionChainDeployHeight: big.NewInt(0),
		IsVerifier:                            isVerifier,
		// Set as an empty string as this is a dummy value anyways.
		// The client needs to be mocked with a mockClient
		RollupClientHttp: "",
	}

	service, err := NewSyncService(context.Background(), cfg, txPool, chain, db)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("Cannot initialize syncservice: %w", err)
	}

	txCh := make(chan core.NewTxsEvent, 1)
	sub := service.SubscribeNewTxsEvent(txCh)

	return service, txCh, sub, nil
}

type mockClient struct {
	getEnqueueCallCount     int
	getEnqueue              []*types.Transaction
	getTransactionCallCount int
	getTransaction          []*types.Transaction
	getEthContextCallCount  int
	getEthContext           []*EthContext
	getLatestEthContext     *EthContext
}

func setupMockClient(service *SyncService, responses map[string]interface{}) {
	client := newMockClient(responses)
	service.client = client
	service.L1gpo = gasprice.NewL1Oracle(big.NewInt(0))
}

func newMockClient(responses map[string]interface{}) *mockClient {
	getEnqueueResponses := []*types.Transaction{}
	getTransactionResponses := []*types.Transaction{}
	getEthContextResponses := []*EthContext{}
	getLatestEthContextResponse := &EthContext{}

	enqueue, ok := responses["GetEnqueue"]
	if ok {
		getEnqueueResponses = enqueue.([]*types.Transaction)
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
	return &mockClient{
		getEnqueue:          getEnqueueResponses,
		getTransaction:      getTransactionResponses,
		getEthContext:       getEthContextResponses,
		getLatestEthContext: getLatestEthContextResponse,
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
		return &types.Transaction{}, errors.New("")
	}
	return m.getEnqueue[len(m.getEnqueue)-1], nil
}

func (m *mockClient) GetTransaction(index uint64) (*types.Transaction, error) {
	if m.getTransactionCallCount < len(m.getTransaction) {
		tx := m.getTransaction[m.getTransactionCallCount]
		m.getTransactionCallCount++
		return tx, nil
	}
	return nil, errors.New("")
}

func (m *mockClient) GetLatestTransaction() (*types.Transaction, error) {
	if len(m.getTransaction) == 0 {
		return nil, errors.New("")
	}
	return m.getTransaction[len(m.getTransaction)-1], nil
}

func (m *mockClient) GetEthContext(index uint64) (*EthContext, error) {
	if m.getEthContextCallCount < len(m.getEthContext) {
		ctx := m.getEthContext[m.getEthContextCallCount]
		m.getEthContextCallCount++
		return ctx, nil
	}
	return nil, errors.New("")
}

func (m *mockClient) GetLatestEthContext() (*EthContext, error) {
	return m.getLatestEthContext, nil
}

func (m *mockClient) GetLastConfirmedEnqueue() (*types.Transaction, error) {
	return nil, nil
}

func (m *mockClient) SyncStatus() (*SyncStatus, error) {
	return &SyncStatus{
		Syncing: false,
	}, nil
}

func (m *mockClient) GetL1GasPrice() (*big.Int, error) {
	return big.NewInt(100 * int64(params.GWei)), nil
}
